package handlers_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/siri5creative/E-comWA/api/internal/handlers"
	"github.com/siri5creative/E-comWA/api/internal/testutil"
)

func postJSON(t *testing.T, h http.HandlerFunc, path string, body any) *httptest.ResponseRecorder {
	t.Helper()
	raw, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("failed to marshal request body: %v", err)
	}
	req := httptest.NewRequest(http.MethodPost, path, bytes.NewReader(raw))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	h(rec, req)
	return rec
}

func decodeJSON(t *testing.T, rec *httptest.ResponseRecorder) map[string]any {
	t.Helper()
	var out map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatalf("failed to decode response body %q: %v", rec.Body.String(), err)
	}
	return out
}

func TestCheckoutSuccess(t *testing.T) {
	pool := testutil.NewTestDB(t)
	_, variantID := testutil.SeedProductVariant(t, pool, "Kaos Polos", "kaos-polos", 85000, 5)

	rec := postJSON(t, handlers.Checkout(pool, nil), "/checkout", map[string]any{
		"name":            "Rivky",
		"whatsapp_number": "081234567890",
		"items": []map[string]any{
			{"product_variant_id": variantID, "quantity": 2},
		},
	})

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, body = %s; want 201", rec.Code, rec.Body.String())
	}
	body := decodeJSON(t, rec)

	if body["status"] != "menunggu_konfirmasi" {
		t.Errorf("status = %v; want menunggu_konfirmasi", body["status"])
	}
	if body["subtotal"].(float64) != 170000 {
		t.Errorf("subtotal = %v; want 170000", body["subtotal"])
	}
	if body["total"].(float64) != 170000 {
		t.Errorf("total = %v; want 170000", body["total"])
	}

	var stock int32
	err := pool.QueryRow(t.Context(), `SELECT stock_quantity FROM product_variants WHERE id = $1`, variantID).Scan(&stock)
	if err != nil {
		t.Fatalf("failed to read stock: %v", err)
	}
	if stock != 3 {
		t.Errorf("stock_quantity = %d; want 3 (5 - 2)", stock)
	}

	var normalizedNumber string
	err = pool.QueryRow(t.Context(), `SELECT whatsapp_number FROM customers WHERE name = 'Rivky'`).Scan(&normalizedNumber)
	if err != nil {
		t.Fatalf("failed to read customer: %v", err)
	}
	if normalizedNumber != "6281234567890" {
		t.Errorf("whatsapp_number = %q; want 6281234567890 (normalized)", normalizedNumber)
	}
}

func TestCheckoutInsufficientStock(t *testing.T) {
	pool := testutil.NewTestDB(t)
	_, variantID := testutil.SeedProductVariant(t, pool, "Sneakers", "sneakers", 250000, 1)

	rec := postJSON(t, handlers.Checkout(pool, nil), "/checkout", map[string]any{
		"name":            "Budi",
		"whatsapp_number": "081111111111",
		"items": []map[string]any{
			{"product_variant_id": variantID, "quantity": 5},
		},
	})

	if rec.Code != http.StatusConflict {
		t.Fatalf("status = %d, body = %s; want 409", rec.Code, rec.Body.String())
	}
	body := decodeJSON(t, rec)
	if body["error"] != "insufficient_stock" {
		t.Errorf("error = %v; want insufficient_stock", body["error"])
	}

	var stock int32
	err := pool.QueryRow(t.Context(), `SELECT stock_quantity FROM product_variants WHERE id = $1`, variantID).Scan(&stock)
	if err != nil {
		t.Fatalf("failed to read stock: %v", err)
	}
	if stock != 1 {
		t.Errorf("stock_quantity = %d; want unchanged at 1 (whole checkout must roll back)", stock)
	}
}

func TestCheckoutValidationErrors(t *testing.T) {
	pool := testutil.NewTestDB(t)
	_, variantID := testutil.SeedProductVariant(t, pool, "Kaos", "kaos", 50000, 10)

	cases := []struct {
		name string
		body map[string]any
	}{
		{
			name: "missing name",
			body: map[string]any{
				"whatsapp_number": "081111111111",
				"items":           []map[string]any{{"product_variant_id": variantID, "quantity": 1}},
			},
		},
		{
			name: "no items",
			body: map[string]any{
				"name":            "Test",
				"whatsapp_number": "081111111111",
				"items":           []map[string]any{},
			},
		},
		{
			name: "invalid whatsapp number",
			body: map[string]any{
				"name":            "Test",
				"whatsapp_number": "abc123",
				"items":           []map[string]any{{"product_variant_id": variantID, "quantity": 1}},
			},
		},
		{
			name: "zero quantity item",
			body: map[string]any{
				"name":            "Test",
				"whatsapp_number": "081111111111",
				"items":           []map[string]any{{"product_variant_id": variantID, "quantity": 0}},
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rec := postJSON(t, handlers.Checkout(pool, nil), "/checkout", tc.body)
			if rec.Code != http.StatusBadRequest {
				t.Fatalf("status = %d, body = %s; want 400", rec.Code, rec.Body.String())
			}
		})
	}
}

func TestCheckoutWithValidCoupon(t *testing.T) {
	pool := testutil.NewTestDB(t)
	_, variantID := testutil.SeedProductVariant(t, pool, "Kaos", "kaos", 100000, 10)
	testutil.SeedCoupon(t, pool, "HEMAT20K", "total_belanja", "fixed", 20000, testutil.CouponOptions{})

	rec := postJSON(t, handlers.Checkout(pool, nil), "/checkout", map[string]any{
		"name":            "Citra",
		"whatsapp_number": "081111111111",
		"items": []map[string]any{
			{"product_variant_id": variantID, "quantity": 2},
		},
		"coupon_code": "hemat20k",
	})

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, body = %s; want 201", rec.Code, rec.Body.String())
	}
	body := decodeJSON(t, rec)
	if body["discount_amount"].(float64) != 20000 {
		t.Errorf("discount_amount = %v; want 20000", body["discount_amount"])
	}
	if body["total"].(float64) != 180000 {
		t.Errorf("total = %v; want 180000 (200000 - 20000)", body["total"])
	}
	if body["coupon_code"] != "HEMAT20K" {
		t.Errorf("coupon_code = %v; want HEMAT20K", body["coupon_code"])
	}

	var usageCount int32
	err := pool.QueryRow(t.Context(), `SELECT current_usage_count FROM coupons WHERE code = 'HEMAT20K'`).Scan(&usageCount)
	if err != nil {
		t.Fatalf("failed to read coupon usage count: %v", err)
	}
	if usageCount != 1 {
		t.Errorf("current_usage_count = %d; want 1", usageCount)
	}

	var usageRows int
	err = pool.QueryRow(t.Context(), `SELECT count(*) FROM coupon_usages WHERE coupon_id = (SELECT id FROM coupons WHERE code = 'HEMAT20K')`).Scan(&usageRows)
	if err != nil {
		t.Fatalf("failed to count coupon_usages: %v", err)
	}
	if usageRows != 1 {
		t.Errorf("coupon_usages rows = %d; want 1", usageRows)
	}
}

func TestCheckoutWithInvalidCouponRollsBackWholeOrder(t *testing.T) {
	pool := testutil.NewTestDB(t)
	_, variantID := testutil.SeedProductVariant(t, pool, "Kaos", "kaos", 100000, 10)

	rec := postJSON(t, handlers.Checkout(pool, nil), "/checkout", map[string]any{
		"name":            "Dedi",
		"whatsapp_number": "081111111111",
		"items": []map[string]any{
			{"product_variant_id": variantID, "quantity": 2},
		},
		"coupon_code": "TIDAKADA",
	})

	if rec.Code != http.StatusConflict {
		t.Fatalf("status = %d, body = %s; want 409", rec.Code, rec.Body.String())
	}
	body := decodeJSON(t, rec)
	if body["error"] != "coupon_invalid" {
		t.Errorf("error = %v; want coupon_invalid", body["error"])
	}

	// Stock must be rolled back along with the coupon failure — an invalid
	// coupon fails the whole order, not just the discount.
	var stock int32
	err := pool.QueryRow(t.Context(), `SELECT stock_quantity FROM product_variants WHERE id = $1`, variantID).Scan(&stock)
	if err != nil {
		t.Fatalf("failed to read stock: %v", err)
	}
	if stock != 10 {
		t.Errorf("stock_quantity = %d; want unchanged at 10", stock)
	}

	var orderCount int
	err = pool.QueryRow(t.Context(), `SELECT count(*) FROM orders`).Scan(&orderCount)
	if err != nil {
		t.Fatalf("failed to count orders: %v", err)
	}
	if orderCount != 0 {
		t.Errorf("orders count = %d; want 0 (no partial order should be created)", orderCount)
	}
}
