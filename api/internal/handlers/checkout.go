package handlers

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/siri5creative/E-comWA/api/internal/httpx"
	"github.com/siri5creative/E-comWA/api/internal/models"
	"github.com/siri5creative/E-comWA/api/internal/notify"
	"github.com/siri5creative/E-comWA/api/internal/util"
)

type checkoutRequest struct {
	Name           string             `json:"name"`
	WhatsAppNumber string             `json:"whatsapp_number"`
	Items          []orderItemRequest `json:"items"`
	CouponCode     *string            `json:"coupon_code"`
}

// Checkout handles POST /checkout — public order creation (PRD section 6.2,
// 6.3). Registers/updates the lightweight customer record, then atomically
// decrements stock for every item in one transaction: if any item can't be
// fulfilled, the entire order is rolled back (same all-or-nothing invariant
// POS transactions use — PRD section 7A / api-pos-integration.md 5.3). An
// optional coupon_code is re-validated authoritatively inside the same
// transaction (never trusting a client-side pre-check) — if it's no longer
// valid (e.g. quota exhausted by a concurrent checkout), the whole order is
// rolled back too, same all-or-nothing principle.
func Checkout(pool *pgxpool.Pool, fcm *notify.FCMClient) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req checkoutRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			httpx.WriteError(w, http.StatusBadRequest, "validation_error", "invalid request body")
			return
		}

		req.Name = strings.TrimSpace(req.Name)
		if req.Name == "" {
			httpx.WriteError(w, http.StatusBadRequest, "validation_error", "nama wajib diisi")
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

		waNumber, err := util.NormalizeWhatsAppNumber(req.WhatsAppNumber)
		if err != nil {
			httpx.WriteError(w, http.StatusBadRequest, "validation_error", err.Error())
			return
		}

		ctx := r.Context()
		tx, err := pool.Begin(ctx)
		if err != nil {
			httpx.WriteError(w, http.StatusInternalServerError, "internal_error", "failed to start transaction")
			return
		}
		defer tx.Rollback(ctx)

		var customerID string
		err = tx.QueryRow(ctx, `
			INSERT INTO customers (name, whatsapp_number)
			VALUES ($1, $2)
			ON CONFLICT (whatsapp_number) DO UPDATE SET name = EXCLUDED.name
			RETURNING id
		`, req.Name, waNumber).Scan(&customerID)
		if err != nil {
			httpx.WriteError(w, http.StatusInternalServerError, "internal_error", "failed to save customer")
			return
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
		orderItems := make([]models.OrderItem, 0, len(reserved))
		couponItems := make([]couponCartItem, 0, len(reserved))
		for _, item := range reserved {
			subtotal += item.Price * int64(item.Quantity)
			orderItems = append(orderItems, models.OrderItem{
				ProductVariantID: item.ProductVariantID,
				Quantity:         item.Quantity,
				PriceAtPurchase:  item.Price,
			})
			couponItems = append(couponItems, couponCartItem{
				ProductID: item.ProductID,
				Price:     item.Price,
				Quantity:  item.Quantity,
			})
		}

		var couponResult *couponEvalResult
		if req.CouponCode != nil && strings.TrimSpace(*req.CouponCode) != "" {
			result, reason, err := evaluateCoupon(ctx, tx, *req.CouponCode, waNumber, couponItems)
			if err != nil {
				httpx.WriteError(w, http.StatusInternalServerError, "internal_error", "failed to validate coupon")
				return
			}
			if reason != "" {
				httpx.WriteJSON(w, http.StatusConflict, map[string]any{
					"error":   "coupon_invalid",
					"reason":  reason,
					"message": couponInvalidReasonMessage[reason],
				})
				return
			}
			couponResult = result
		}

		discountAmount := int64(0)
		var couponID *string
		if couponResult != nil {
			discountAmount = couponResult.DiscountAmount
			couponID = &couponResult.CouponID
		}
		total := subtotal - discountAmount

		invoiceNumber, err := util.GenerateInvoiceNumber("INV")
		if err != nil {
			httpx.WriteError(w, http.StatusInternalServerError, "internal_error", "failed to generate invoice number")
			return
		}

		var order models.Order
		order.InvoiceNumber = invoiceNumber
		order.Channel = models.OrderChannelOnline
		order.Status = models.OrderStatusMenungguKonfirmasi
		order.Subtotal = subtotal
		order.DiscountAmount = discountAmount
		order.Total = total
		if couponResult != nil {
			order.CouponCode = &couponResult.Code
		}

		err = tx.QueryRow(ctx, `
			INSERT INTO orders (invoice_number, customer_id, channel, status, coupon_id, subtotal, discount_amount, total)
			VALUES ($1, $2, 'online', 'menunggu_konfirmasi', $3, $4, $5, $6)
			RETURNING id, created_at
		`, invoiceNumber, customerID, couponID, subtotal, discountAmount, total).Scan(&order.ID, &order.CreatedAt)
		if err != nil {
			httpx.WriteError(w, http.StatusInternalServerError, "internal_error", "failed to create order")
			return
		}

		batch := &pgx.Batch{}
		for _, item := range orderItems {
			batch.Queue(`
				INSERT INTO order_items (order_id, product_variant_id, quantity, price_at_purchase)
				VALUES ($1, $2, $3, $4)
			`, order.ID, item.ProductVariantID, item.Quantity, item.PriceAtPurchase)
		}
		br := tx.SendBatch(ctx, batch)
		if err := br.Close(); err != nil {
			httpx.WriteError(w, http.StatusInternalServerError, "internal_error", "failed to save order items")
			return
		}

		if couponResult != nil {
			tag, err := tx.Exec(ctx, `
				UPDATE coupons SET current_usage_count = current_usage_count + 1 WHERE id = $1
			`, couponResult.CouponID)
			if err != nil || tag.RowsAffected() == 0 {
				httpx.WriteError(w, http.StatusInternalServerError, "internal_error", "failed to record coupon usage")
				return
			}
			_, err = tx.Exec(ctx, `
				INSERT INTO coupon_usages (coupon_id, customer_id, order_id) VALUES ($1, $2, $3)
			`, couponResult.CouponID, customerID, order.ID)
			if err != nil {
				httpx.WriteError(w, http.StatusInternalServerError, "internal_error", "failed to record coupon usage")
				return
			}
		}

		if err := tx.Commit(ctx); err != nil {
			httpx.WriteError(w, http.StatusInternalServerError, "internal_error", "failed to commit order")
			return
		}

		notifyAdminsOfNewOrder(pool, fcm, order.InvoiceNumber, order.Total)

		httpx.WriteJSON(w, http.StatusCreated, order)
	}
}

// notifyAdminsOfNewOrder pushes a "new order" notification to every
// registered admin device (PRD section 6.6). Best-effort and
// fire-and-forget: it runs after the response is on its way back to the
// customer, on its own bounded context (not r.Context(), which is
// cancelled once the handler returns), and never affects the checkout
// outcome — a Firebase outage must not break checkout.
func notifyAdminsOfNewOrder(pool *pgxpool.Pool, fcm *notify.FCMClient, invoiceNumber string, total int64) {
	if fcm == nil {
		return
	}

	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		rows, err := pool.Query(ctx, `SELECT fcm_device_token FROM admin_devices`)
		if err != nil {
			slog.Warn("failed to load admin devices for notification", "error", err)
			return
		}
		defer rows.Close()

		var tokens []string
		for rows.Next() {
			var token string
			if err := rows.Scan(&token); err != nil {
				slog.Warn("failed to read admin device token", "error", err)
				continue
			}
			tokens = append(tokens, token)
		}
		if err := rows.Err(); err != nil {
			slog.Warn("failed to read admin devices for notification", "error", err)
			return
		}

		fcm.SendToAll(ctx, tokens,
			"Order Baru Masuk",
			invoiceNumber+" - "+util.FormatRupiah(total),
			map[string]string{"invoice_number": invoiceNumber},
		)
	}()
}
