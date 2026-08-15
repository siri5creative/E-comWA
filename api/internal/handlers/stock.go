package handlers

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
)

// orderItemRequest is the shared {product_variant_id, quantity} request
// shape used by both POST /checkout and POST /pos/orders.
type orderItemRequest struct {
	ProductVariantID string `json:"product_variant_id"`
	Quantity         int32  `json:"quantity"`
}

type insufficientStockDetail struct {
	ProductVariantID string `json:"product_variant_id"`
	Requested        int32  `json:"requested"`
	Available        int32  `json:"available"`
}

type reservedItem struct {
	ProductVariantID string
	ProductID        string
	Price            int64
	Quantity         int32
}

// reserveStockItems atomically decrements stock for every requested item
// within tx — one UPDATE ... WHERE stock_quantity >= qty per item, never a
// separate check-then-update (PRD section 7A). Shared by online checkout
// and POS orders since both need the identical all-or-nothing guarantee:
// if any item can't be fulfilled, this returns those failures in
// `insufficient` and the caller must roll back the whole transaction
// rather than applying a partial order (api-pos-integration.md 5.3).
func reserveStockItems(ctx context.Context, tx pgx.Tx, items []orderItemRequest) (reserved []reservedItem, insufficient []insufficientStockDetail, err error) {
	reserved = make([]reservedItem, 0, len(items))

	for _, item := range items {
		var price int64
		var productID string
		err := tx.QueryRow(ctx, `
			UPDATE product_variants
			SET stock_quantity = stock_quantity - $1
			WHERE id = $2 AND stock_quantity >= $1
			RETURNING price, product_id
		`, item.Quantity, item.ProductVariantID).Scan(&price, &productID)

		if err == nil {
			reserved = append(reserved, reservedItem{
				ProductVariantID: item.ProductVariantID,
				ProductID:        productID,
				Price:            price,
				Quantity:         item.Quantity,
			})
			continue
		}
		if !errors.Is(err, pgx.ErrNoRows) {
			return nil, nil, err
		}

		var available int32
		_ = tx.QueryRow(ctx, `SELECT stock_quantity FROM product_variants WHERE id = $1`, item.ProductVariantID).Scan(&available)
		insufficient = append(insufficient, insufficientStockDetail{
			ProductVariantID: item.ProductVariantID,
			Requested:        item.Quantity,
			Available:        available,
		})
	}

	return reserved, insufficient, nil
}
