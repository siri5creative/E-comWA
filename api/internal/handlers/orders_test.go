package handlers_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/siri5creative/E-comWA/api/internal/handlers"
	"github.com/siri5creative/E-comWA/api/internal/testutil"
)

// seedOrder inserts an order directly (bypassing Checkout/CreatePOSOrder)
// so tests can control the starting status and amounts precisely.
func seedOrder(t *testing.T, pool *pgxpool.Pool, status string, subtotal, discount, shipping int64) string {
	t.Helper()
	invoiceNumber := "INV-TEST-" + status + "-" + time.Now().Format("150405.000000000")
	total := subtotal - discount + shipping
	var orderID string
	err := pool.QueryRow(t.Context(), `
		INSERT INTO orders (invoice_number, channel, status, subtotal, discount_amount, shipping_cost, total)
		VALUES ($1, 'online', $2, $3, $4, $5, $6)
		RETURNING id
	`, invoiceNumber, status, subtotal, discount, shipping, total).Scan(&orderID)
	if err != nil {
		t.Fatalf("failed to seed order: %v", err)
	}
	return orderID
}

func patchJSON(t *testing.T, h http.HandlerFunc, path string, pathValues map[string]string, body any) *httptest.ResponseRecorder {
	t.Helper()
	raw, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("failed to marshal request body: %v", err)
	}
	req := httptest.NewRequest(http.MethodPatch, path, bytes.NewReader(raw))
	req.Header.Set("Content-Type", "application/json")
	for k, v := range pathValues {
		req.SetPathValue(k, v)
	}
	rec := httptest.NewRecorder()
	h(rec, req)
	return rec
}

func TestUpdateOrderStatusCannotSkipPaymentConfirmation(t *testing.T) {
	pool := testutil.NewTestDB(t)
	orderID := seedOrder(t, pool, "menunggu_konfirmasi", 100000, 0, 0)

	rec := patchJSON(t, handlers.UpdateOrderStatus(pool), "/orders/"+orderID+"/status",
		map[string]string{"id": orderID}, map[string]any{"status": "diproses"})

	if rec.Code != http.StatusConflict {
		t.Fatalf("status = %d, body = %s; want 409 — must not allow skipping straight to diproses", rec.Code, rec.Body.String())
	}
	body := decodeJSON(t, rec)
	if body["error"] != "invalid_status_transition" {
		t.Errorf("error = %v; want invalid_status_transition", body["error"])
	}
}

func TestUpdateOrderStatusValidTransition(t *testing.T) {
	pool := testutil.NewTestDB(t)
	orderID := seedOrder(t, pool, "menunggu_konfirmasi", 100000, 0, 0)

	rec := patchJSON(t, handlers.UpdateOrderStatus(pool), "/orders/"+orderID+"/status",
		map[string]string{"id": orderID}, map[string]any{"status": "menunggu_pembayaran"})

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s; want 200", rec.Code, rec.Body.String())
	}

	var status string
	err := pool.QueryRow(t.Context(), `SELECT status::text FROM orders WHERE id = $1`, orderID).Scan(&status)
	if err != nil {
		t.Fatalf("failed to read order status: %v", err)
	}
	if status != "menunggu_pembayaran" {
		t.Errorf("status in DB = %q; want menunggu_pembayaran", status)
	}
}

func TestUpdateOrderStatusDiprosesAfterPaymentConfirmed(t *testing.T) {
	pool := testutil.NewTestDB(t)
	orderID := seedOrder(t, pool, "menunggu_pembayaran", 100000, 0, 0)

	rec := patchJSON(t, handlers.UpdateOrderStatus(pool), "/orders/"+orderID+"/status",
		map[string]string{"id": orderID}, map[string]any{"status": "diproses"})

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s; want 200 (diproses is valid once payment confirmed)", rec.Code, rec.Body.String())
	}
}

func TestUpdateOrderStatusShippingCostRecalculatesTotal(t *testing.T) {
	pool := testutil.NewTestDB(t)
	orderID := seedOrder(t, pool, "menunggu_pembayaran", 100000, 10000, 0)

	rec := patchJSON(t, handlers.UpdateOrderStatus(pool), "/orders/"+orderID+"/status",
		map[string]string{"id": orderID}, map[string]any{"shipping_cost": 15000})

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s; want 200", rec.Code, rec.Body.String())
	}
	body := decodeJSON(t, rec)
	if body["total"].(float64) != 105000 {
		t.Errorf("total = %v; want 105000 (100000 - 10000 + 15000)", body["total"])
	}

	// Status must be unaffected by a shipping-cost-only update.
	var status string
	err := pool.QueryRow(t.Context(), `SELECT status::text FROM orders WHERE id = $1`, orderID).Scan(&status)
	if err != nil {
		t.Fatalf("failed to read order status: %v", err)
	}
	if status != "menunggu_pembayaran" {
		t.Errorf("status = %q; want unchanged at menunggu_pembayaran", status)
	}
}

func TestUpdateOrderStatusFinalOrderIsLocked(t *testing.T) {
	pool := testutil.NewTestDB(t)

	for _, terminalStatus := range []string{"selesai", "dibatalkan"} {
		t.Run(terminalStatus, func(t *testing.T) {
			orderID := seedOrder(t, pool, terminalStatus, 100000, 0, 0)

			rec := patchJSON(t, handlers.UpdateOrderStatus(pool), "/orders/"+orderID+"/status",
				map[string]string{"id": orderID}, map[string]any{"shipping_cost": 99999})

			if rec.Code != http.StatusConflict {
				t.Fatalf("status = %d, body = %s; want 409 — %s orders must be locked", rec.Code, rec.Body.String(), terminalStatus)
			}
			body := decodeJSON(t, rec)
			if body["error"] != "order_final" {
				t.Errorf("error = %v; want order_final", body["error"])
			}
		})
	}
}

func TestUpdateOrderStatusRequiresAtLeastOneField(t *testing.T) {
	pool := testutil.NewTestDB(t)
	orderID := seedOrder(t, pool, "menunggu_konfirmasi", 100000, 0, 0)

	rec := patchJSON(t, handlers.UpdateOrderStatus(pool), "/orders/"+orderID+"/status",
		map[string]string{"id": orderID}, map[string]any{})

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d; want 400", rec.Code)
	}
}

func TestGetOrderIncludesNextStatuses(t *testing.T) {
	pool := testutil.NewTestDB(t)
	orderID := seedOrder(t, pool, "menunggu_konfirmasi", 100000, 0, 0)

	req := httptest.NewRequest(http.MethodGet, "/orders/"+orderID, nil)
	req.SetPathValue("id", orderID)
	rec := httptest.NewRecorder()
	handlers.GetOrder(pool)(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s; want 200", rec.Code, rec.Body.String())
	}
	body := decodeJSON(t, rec)
	next, ok := body["next_statuses"].([]any)
	if !ok {
		t.Fatalf("next_statuses = %v (%T); want a JSON array, not null", body["next_statuses"], body["next_statuses"])
	}
	if len(next) != 2 {
		t.Errorf("next_statuses = %v; want 2 entries (menunggu_pembayaran, dibatalkan)", next)
	}
}

func TestGetOrderTerminalStatusHasEmptyNextStatusesArray(t *testing.T) {
	pool := testutil.NewTestDB(t)
	orderID := seedOrder(t, pool, "selesai", 100000, 0, 0)

	req := httptest.NewRequest(http.MethodGet, "/orders/"+orderID, nil)
	req.SetPathValue("id", orderID)
	rec := httptest.NewRecorder()
	handlers.GetOrder(pool)(rec, req)

	body := decodeJSON(t, rec)
	// Must decode as `[]any{}`, not nil — a JSON `null` would break
	// frontend code calling .length on it.
	next, ok := body["next_statuses"].([]any)
	if !ok {
		t.Fatalf("next_statuses = %v (%T); want an empty JSON array, not null", body["next_statuses"], body["next_statuses"])
	}
	if len(next) != 0 {
		t.Errorf("next_statuses = %v; want empty", next)
	}
}
