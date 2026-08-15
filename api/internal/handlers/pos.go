package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/siri5creative/E-comWA/api/internal/httpx"
	"github.com/siri5creative/E-comWA/api/internal/util"
)

// GetVariantStock handles GET /products/variants/:variant_id/stock — public
// (api-pos-integration.md section 5.2), same trust level as GET /products;
// only /pos/* itself needs POS_API_KEY (section 4).
func GetVariantStock(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		variantID := r.PathValue("variant_id")

		var stock int32
		err := pool.QueryRow(r.Context(), `
			SELECT stock_quantity FROM product_variants WHERE id = $1
		`, variantID).Scan(&stock)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				httpx.WriteError(w, http.StatusNotFound, "not_found", "varian produk tidak ditemukan")
				return
			}
			httpx.WriteError(w, http.StatusInternalServerError, "internal_error", "failed to load stock")
			return
		}

		httpx.WriteJSON(w, http.StatusOK, map[string]any{
			"variant_id":     variantID,
			"stock_quantity": stock,
		})
	}
}

type createPOSOrderRequest struct {
	Items            []orderItemRequest `json:"items"`
	PaymentMethod    string             `json:"payment_method"`
	CustomerName     string             `json:"customer_name"`
	CustomerWhatsApp string             `json:"customer_whatsapp"`
}

type posOrderResponse struct {
	OrderID       string    `json:"order_id"`
	InvoiceNumber string    `json:"invoice_number"`
	Channel       string    `json:"channel"`
	Status        string    `json:"status"`
	Subtotal      int64     `json:"subtotal"`
	Total         int64     `json:"total"`
	CreatedAt     time.Time `json:"created_at"`
}

// CreatePOSOrder handles POST /pos/orders — auth via POS_API_KEY
// (api-pos-integration.md section 5.3). Unlike online checkout, a POS sale
// is final the moment it's rung up: no confirmation/payment-wait states,
// no shipping, and customer info is optional (PRD section 7A — walk-in
// buyers aren't required to register). Stock is reserved with the same
// atomic, all-or-nothing helper checkout.go uses.
func CreatePOSOrder(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req createPOSOrderRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			httpx.WriteError(w, http.StatusBadRequest, "validation_error", "invalid request body")
			return
		}
		if len(req.Items) == 0 {
			httpx.WriteError(w, http.StatusBadRequest, "validation_error", "minimal 1 item wajib diisi")
			return
		}
		for _, item := range req.Items {
			if item.ProductVariantID == "" || item.Quantity < 1 {
				httpx.WriteError(w, http.StatusBadRequest, "validation_error", "item tidak valid")
				return
			}
		}

		var waNumber string
		if strings.TrimSpace(req.CustomerWhatsApp) != "" {
			normalized, err := util.NormalizeWhatsAppNumber(req.CustomerWhatsApp)
			if err != nil {
				httpx.WriteError(w, http.StatusBadRequest, "validation_error", err.Error())
				return
			}
			waNumber = normalized
		}

		ctx := r.Context()
		tx, err := pool.Begin(ctx)
		if err != nil {
			httpx.WriteError(w, http.StatusInternalServerError, "internal_error", "failed to start transaction")
			return
		}
		defer tx.Rollback(ctx)

		var customerID *string
		if waNumber != "" {
			name := strings.TrimSpace(req.CustomerName)
			if name == "" {
				name = "Pelanggan POS"
			}
			var id string
			err = tx.QueryRow(ctx, `
				INSERT INTO customers (name, whatsapp_number)
				VALUES ($1, $2)
				ON CONFLICT (whatsapp_number) DO UPDATE SET name = EXCLUDED.name
				RETURNING id
			`, name, waNumber).Scan(&id)
			if err != nil {
				httpx.WriteError(w, http.StatusInternalServerError, "internal_error", "failed to save customer")
				return
			}
			customerID = &id
		}

		reserved, insufficient, err := reserveStockItems(ctx, tx, req.Items)
		if err != nil {
			httpx.WriteError(w, http.StatusInternalServerError, "internal_error", "failed to reserve stock")
			return
		}
		if len(insufficient) > 0 {
			httpx.WriteJSON(w, http.StatusConflict, map[string]any{
				"error":   "insufficient_stock",
				"message": "Stok tidak cukup untuk salah satu item",
				"details": insufficient,
			})
			return
		}

		var subtotal int64
		for _, item := range reserved {
			subtotal += item.Price * int64(item.Quantity)
		}

		invoiceNumber, err := util.GenerateInvoiceNumber("POS")
		if err != nil {
			httpx.WriteError(w, http.StatusInternalServerError, "internal_error", "failed to generate invoice number")
			return
		}

		var paymentMethod *string
		if pm := strings.TrimSpace(req.PaymentMethod); pm != "" {
			paymentMethod = &pm
		}

		var resp posOrderResponse
		resp.InvoiceNumber = invoiceNumber
		resp.Channel = "pos"
		resp.Status = "selesai"
		resp.Subtotal = subtotal
		resp.Total = subtotal

		err = tx.QueryRow(ctx, `
			INSERT INTO orders (invoice_number, customer_id, channel, status, subtotal, total, payment_method)
			VALUES ($1, $2, 'pos', 'selesai', $3, $3, $4)
			RETURNING id, created_at
		`, invoiceNumber, customerID, subtotal, paymentMethod).Scan(&resp.OrderID, &resp.CreatedAt)
		if err != nil {
			httpx.WriteError(w, http.StatusInternalServerError, "internal_error", "failed to create order")
			return
		}

		batch := &pgx.Batch{}
		for _, item := range reserved {
			batch.Queue(`
				INSERT INTO order_items (order_id, product_variant_id, quantity, price_at_purchase)
				VALUES ($1, $2, $3, $4)
			`, resp.OrderID, item.ProductVariantID, item.Quantity, item.Price)
		}
		br := tx.SendBatch(ctx, batch)
		if err := br.Close(); err != nil {
			httpx.WriteError(w, http.StatusInternalServerError, "internal_error", "failed to save order items")
			return
		}

		if err := tx.Commit(ctx); err != nil {
			httpx.WriteError(w, http.StatusInternalServerError, "internal_error", "failed to commit order")
			return
		}

		httpx.WriteJSON(w, http.StatusCreated, resp)
	}
}

// GetPOSOrder handles GET /pos/orders/:order_id — auth via POS_API_KEY
// (api-pos-integration.md section 5.4), for reprinting a receipt or
// looking up transaction history from the POS side.
func GetPOSOrder(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		orderID := r.PathValue("order_id")
		ctx := r.Context()

		var invoiceNumber, status string
		var subtotal, total int64
		var paymentMethod *string
		var createdAt time.Time

		err := pool.QueryRow(ctx, `
			SELECT invoice_number, status::text, subtotal, total, payment_method, created_at
			FROM orders
			WHERE id = $1 AND channel = 'pos'
		`, orderID).Scan(&invoiceNumber, &status, &subtotal, &total, &paymentMethod, &createdAt)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				httpx.WriteError(w, http.StatusNotFound, "not_found", "transaksi POS tidak ditemukan")
				return
			}
			httpx.WriteError(w, http.StatusInternalServerError, "internal_error", "failed to load order")
			return
		}

		items, err := fetchOrderItems(ctx, pool, orderID)
		if err != nil {
			httpx.WriteError(w, http.StatusInternalServerError, "internal_error", "failed to load order items")
			return
		}

		httpx.WriteJSON(w, http.StatusOK, map[string]any{
			"order_id":       orderID,
			"invoice_number": invoiceNumber,
			"channel":        "pos",
			"status":         status,
			"items":          items,
			"subtotal":       subtotal,
			"total":          total,
			"payment_method": paymentMethod,
			"created_at":     createdAt,
		})
	}
}
