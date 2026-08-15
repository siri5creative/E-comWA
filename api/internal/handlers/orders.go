package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/siri5creative/E-comWA/api/internal/httpx"
	"github.com/siri5creative/E-comWA/api/internal/models"
	"github.com/siri5creative/E-comWA/api/internal/util"
)

type orderListItem struct {
	ID               string    `json:"id"`
	InvoiceNumber    string    `json:"invoice_number"`
	Channel          string    `json:"channel"`
	Status           string    `json:"status"`
	CustomerName     *string   `json:"customer_name"`
	CustomerWhatsApp *string   `json:"customer_whatsapp"`
	Subtotal         int64     `json:"subtotal"`
	DiscountAmount   int64     `json:"discount_amount"`
	ShippingCost     int64     `json:"shipping_cost"`
	Total            int64     `json:"total"`
	CreatedAt        time.Time `json:"created_at"`
}

// ListOrders handles GET /orders — admin only (PRD section 8), with
// optional ?status= and ?channel= filters plus page/page_size pagination.
func ListOrders(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		page := parsePositiveInt(r.URL.Query().Get("page"), 1)
		pageSize := parsePositiveInt(r.URL.Query().Get("page_size"), defaultPageSize)
		if pageSize > maxPageSize {
			pageSize = maxPageSize
		}
		offset := (page - 1) * pageSize

		var status, channel *string
		if v := r.URL.Query().Get("status"); v != "" {
			status = &v
		}
		if v := r.URL.Query().Get("channel"); v != "" {
			channel = &v
		}

		ctx := r.Context()
		rows, err := pool.Query(ctx, `
			SELECT o.id, o.invoice_number, o.channel::text, o.status::text,
			       c.name, c.whatsapp_number,
			       o.subtotal, o.discount_amount, o.shipping_cost, o.total, o.created_at,
			       count(*) OVER() AS total_count
			FROM orders o
			LEFT JOIN customers c ON c.id = o.customer_id
			WHERE ($1::text IS NULL OR o.status::text = $1)
			  AND ($2::text IS NULL OR o.channel::text = $2)
			ORDER BY o.created_at DESC
			LIMIT $3 OFFSET $4
		`, status, channel, pageSize, offset)
		if err != nil {
			httpx.WriteError(w, http.StatusInternalServerError, "internal_error", "failed to list orders")
			return
		}
		defer rows.Close()

		orders := []orderListItem{}
		var total int64
		for rows.Next() {
			var o orderListItem
			if err := rows.Scan(&o.ID, &o.InvoiceNumber, &o.Channel, &o.Status,
				&o.CustomerName, &o.CustomerWhatsApp,
				&o.Subtotal, &o.DiscountAmount, &o.ShippingCost, &o.Total, &o.CreatedAt, &total); err != nil {
				httpx.WriteError(w, http.StatusInternalServerError, "internal_error", "failed to read orders")
				return
			}
			orders = append(orders, o)
		}
		if err := rows.Err(); err != nil {
			httpx.WriteError(w, http.StatusInternalServerError, "internal_error", "failed to read orders")
			return
		}

		httpx.WriteJSON(w, http.StatusOK, map[string]any{
			"data": orders,
			"pagination": paginationMeta{
				Page:     page,
				PageSize: pageSize,
				Total:    total,
			},
		})
	}
}

type orderCustomer struct {
	Name           string `json:"name"`
	WhatsAppNumber string `json:"whatsapp_number"`
}

type orderDetailItem struct {
	ProductVariantID string `json:"product_variant_id"`
	ProductName      string `json:"product_name"`
	VariantName      string `json:"variant_name"`
	Quantity         int32  `json:"quantity"`
	PriceAtPurchase  int64  `json:"price_at_purchase"`
}

type orderDetail struct {
	ID               string               `json:"id"`
	InvoiceNumber    string               `json:"invoice_number"`
	Channel          string               `json:"channel"`
	Status           string               `json:"status"`
	NextStatuses     []models.OrderStatus `json:"next_statuses"`
	Customer         *orderCustomer       `json:"customer"`
	Subtotal         int64                `json:"subtotal"`
	DiscountAmount   int64                `json:"discount_amount"`
	ShippingCost     int64                `json:"shipping_cost"`
	Total            int64                `json:"total"`
	ShippingNote     *string              `json:"shipping_note"`
	PaymentProofNote *string              `json:"payment_proof_note"`
	Items            []orderDetailItem    `json:"items"`
	CreatedAt        time.Time            `json:"created_at"`
	UpdatedAt        time.Time            `json:"updated_at"`
}

// GetOrder handles GET /orders/:id — admin only.
func GetOrder(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")
		ctx := r.Context()

		var o orderDetail
		var customerName, customerWA *string

		err := pool.QueryRow(ctx, `
			SELECT o.id, o.invoice_number, o.channel::text, o.status::text,
			       o.subtotal, o.discount_amount, o.shipping_cost, o.total,
			       o.shipping_note, o.payment_proof_note, o.created_at, o.updated_at,
			       c.name, c.whatsapp_number
			FROM orders o
			LEFT JOIN customers c ON c.id = o.customer_id
			WHERE o.id = $1
		`, id).Scan(&o.ID, &o.InvoiceNumber, &o.Channel, &o.Status,
			&o.Subtotal, &o.DiscountAmount, &o.ShippingCost, &o.Total,
			&o.ShippingNote, &o.PaymentProofNote, &o.CreatedAt, &o.UpdatedAt,
			&customerName, &customerWA)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				httpx.WriteError(w, http.StatusNotFound, "not_found", "order tidak ditemukan")
				return
			}
			httpx.WriteError(w, http.StatusInternalServerError, "internal_error", "failed to load order")
			return
		}
		if customerWA != nil {
			o.Customer = &orderCustomer{Name: derefOrEmpty(customerName), WhatsAppNumber: *customerWA}
		}
		o.NextStatuses = models.NextOrderStatuses(models.OrderStatus(o.Status))

		items, err := fetchOrderItems(ctx, pool, id)
		if err != nil {
			httpx.WriteError(w, http.StatusInternalServerError, "internal_error", "failed to load order items")
			return
		}
		o.Items = items

		httpx.WriteJSON(w, http.StatusOK, o)
	}
}

func fetchOrderItems(ctx context.Context, pool *pgxpool.Pool, orderID string) ([]orderDetailItem, error) {
	rows, err := pool.Query(ctx, `
		SELECT oi.product_variant_id, p.name, pv.variant_name, oi.quantity, oi.price_at_purchase
		FROM order_items oi
		JOIN product_variants pv ON pv.id = oi.product_variant_id
		JOIN products p ON p.id = pv.product_id
		WHERE oi.order_id = $1
		ORDER BY p.name, pv.variant_name
	`, orderID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := []orderDetailItem{}
	for rows.Next() {
		var it orderDetailItem
		if err := rows.Scan(&it.ProductVariantID, &it.ProductName, &it.VariantName, &it.Quantity, &it.PriceAtPurchase); err != nil {
			return nil, err
		}
		items = append(items, it)
	}
	return items, rows.Err()
}

func derefOrEmpty(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

type updateOrderRequest struct {
	Status       *string `json:"status"`
	ShippingCost *int64  `json:"shipping_cost"`
}

// UpdateOrderStatus handles PATCH /orders/:id/status — admin only. Despite
// the "/status" route name (matching the PRD section 8 draft literally),
// it also accepts shipping_cost: PRD section 6.3 requires admins to be able
// to record the manually-agreed shipping cost from the order detail page,
// and the API draft doesn't define a separate endpoint for it. Either
// field may be sent alone or together; total is always recomputed as
// subtotal - discount_amount + shipping_cost.
func UpdateOrderStatus(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")

		var req updateOrderRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			httpx.WriteError(w, http.StatusBadRequest, "validation_error", "invalid request body")
			return
		}
		if req.Status == nil && req.ShippingCost == nil {
			httpx.WriteError(w, http.StatusBadRequest, "validation_error", "status atau shipping_cost wajib diisi")
			return
		}
		if req.ShippingCost != nil && *req.ShippingCost < 0 {
			httpx.WriteError(w, http.StatusBadRequest, "validation_error", "shipping_cost tidak boleh negatif")
			return
		}
		var requestedStatus models.OrderStatus
		if req.Status != nil {
			requestedStatus = models.OrderStatus(*req.Status)
			if !models.ValidOrderStatuses[requestedStatus] {
				httpx.WriteError(w, http.StatusBadRequest, "validation_error", "status tidak valid")
				return
			}
		}

		ctx := r.Context()
		tx, err := pool.Begin(ctx)
		if err != nil {
			httpx.WriteError(w, http.StatusInternalServerError, "internal_error", "failed to start transaction")
			return
		}
		defer tx.Rollback(ctx)

		var currentStatus models.OrderStatus
		var subtotal, discountAmount, shippingCost int64
		err = tx.QueryRow(ctx, `
			SELECT status::text, subtotal, discount_amount, shipping_cost
			FROM orders WHERE id = $1 FOR UPDATE
		`, id).Scan(&currentStatus, &subtotal, &discountAmount, &shippingCost)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				httpx.WriteError(w, http.StatusNotFound, "not_found", "order tidak ditemukan")
				return
			}
			httpx.WriteError(w, http.StatusInternalServerError, "internal_error", "failed to load order")
			return
		}

		if models.IsTerminalOrderStatus(currentStatus) {
			httpx.WriteError(w, http.StatusConflict, "order_final", "order sudah selesai/dibatalkan, tidak bisa diubah")
			return
		}

		newStatus := currentStatus
		if req.Status != nil {
			if !models.CanTransitionOrderStatus(currentStatus, requestedStatus) {
				httpx.WriteError(w, http.StatusConflict, "invalid_status_transition",
					fmt.Sprintf("tidak bisa mengubah status dari %s ke %s", currentStatus, requestedStatus))
				return
			}
			newStatus = requestedStatus
		}

		newShipping := shippingCost
		if req.ShippingCost != nil {
			newShipping = *req.ShippingCost
		}
		newTotal := subtotal - discountAmount + newShipping

		_, err = tx.Exec(ctx, `
			UPDATE orders SET status = $1, shipping_cost = $2, total = $3 WHERE id = $4
		`, newStatus, newShipping, newTotal, id)
		if err != nil {
			httpx.WriteError(w, http.StatusInternalServerError, "internal_error", "failed to update order")
			return
		}

		if err := tx.Commit(ctx); err != nil {
			httpx.WriteError(w, http.StatusInternalServerError, "internal_error", "failed to commit update")
			return
		}

		httpx.WriteJSON(w, http.StatusOK, map[string]any{
			"id":            id,
			"status":        newStatus,
			"shipping_cost": newShipping,
			"total":         newTotal,
			"next_statuses": models.NextOrderStatuses(newStatus),
		})
	}
}

// GenerateWAMessage handles GET /orders/:id/wa-message — admin only. Builds
// the status-appropriate message text and a ready-to-open wa.me link (PRD
// section 6.5): admin clicks through, WhatsApp opens with the text
// pre-filled, admin reviews and sends it themselves.
func GenerateWAMessage(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")
		ctx := r.Context()

		var status, invoiceNumber string
		var total int64
		var customerName, customerWA *string

		err := pool.QueryRow(ctx, `
			SELECT o.status::text, o.invoice_number, o.total, c.name, c.whatsapp_number
			FROM orders o
			LEFT JOIN customers c ON c.id = o.customer_id
			WHERE o.id = $1
		`, id).Scan(&status, &invoiceNumber, &total, &customerName, &customerWA)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				httpx.WriteError(w, http.StatusNotFound, "not_found", "order tidak ditemukan")
				return
			}
			httpx.WriteError(w, http.StatusInternalServerError, "internal_error", "failed to load order")
			return
		}
		if customerWA == nil {
			httpx.WriteError(w, http.StatusBadRequest, "no_whatsapp_number", "order ini tidak punya nomor WhatsApp customer")
			return
		}

		name := "Kak"
		if customerName != nil && *customerName != "" {
			name = *customerName
		}

		message := buildWhatsAppMessage(models.OrderStatus(status), name, invoiceNumber, total)
		waLink := fmt.Sprintf("https://wa.me/%s?text=%s", *customerWA, url.QueryEscape(message))

		httpx.WriteJSON(w, http.StatusOK, map[string]any{
			"whatsapp_number": *customerWA,
			"message":         message,
			"wa_link":         waLink,
		})
	}
}

func buildWhatsAppMessage(status models.OrderStatus, name, invoice string, total int64) string {
	formattedTotal := util.FormatRupiah(total)
	switch status {
	case models.OrderStatusMenungguKonfirmasi:
		return fmt.Sprintf("Halo %s, pesanan kamu dengan invoice %s sudah kami terima dan sedang kami cek. Kami akan segera menghubungi kamu kembali. Terima kasih!", name, invoice)
	case models.OrderStatusMenungguPembayaran:
		return fmt.Sprintf("Halo %s, pesanan kamu dengan invoice %s sudah kami konfirmasi. Total pembayaran: %s. Silakan transfer dan kirim bukti pembayaran ke chat ini ya. Terima kasih!", name, invoice, formattedTotal)
	case models.OrderStatusDiproses:
		return fmt.Sprintf("Halo %s, pembayaran untuk pesanan %s sudah kami terima. Pesanan kamu sedang kami proses. Terima kasih!", name, invoice)
	case models.OrderStatusDikirim:
		return fmt.Sprintf("Halo %s, pesanan kamu dengan invoice %s sudah dikirim. Terima kasih sudah belanja di toko kami!", name, invoice)
	case models.OrderStatusSelesai:
		return fmt.Sprintf("Halo %s, pesanan %s sudah selesai. Terima kasih sudah berbelanja, ditunggu order berikutnya!", name, invoice)
	case models.OrderStatusDibatalkan:
		return fmt.Sprintf("Halo %s, mohon maaf pesanan kamu dengan invoice %s kami batalkan. Kalau ada pertanyaan silakan hubungi kami ya.", name, invoice)
	default:
		return fmt.Sprintf("Halo %s, ada update untuk pesanan %s.", name, invoice)
	}
}
