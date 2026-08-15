package handlers_test

import (
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/siri5creative/E-comWA/api/internal/handlers"
	"github.com/siri5creative/E-comWA/api/internal/testutil"
)

func getJSON(t *testing.T, h http.HandlerFunc, path string, pathValues map[string]string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, path, nil)
	for k, v := range pathValues {
		req.SetPathValue(k, v)
	}
	rec := httptest.NewRecorder()
	h(rec, req)
	return rec
}

func TestCreatePOSOrderWalkIn(t *testing.T) {
	pool := testutil.NewTestDB(t)
	_, variantID := testutil.SeedProductVariant(t, pool, "Kaos", "kaos", 50000, 10)

	rec := postJSON(t, handlers.CreatePOSOrder(pool), "/pos/orders", map[string]any{
		"items":          []map[string]any{{"product_variant_id": variantID, "quantity": 2}},
		"payment_method": "cash",
	})

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, body = %s; want 201", rec.Code, rec.Body.String())
	}
	body := decodeJSON(t, rec)
	if body["status"] != "selesai" {
		t.Errorf("status = %v; want selesai (POS orders are final immediately)", body["status"])
	}
	if body["channel"] != "pos" {
		t.Errorf("channel = %v; want pos", body["channel"])
	}
	if body["total"].(float64) != 100000 {
		t.Errorf("total = %v; want 100000", body["total"])
	}

	orderID := body["order_id"].(string)
	var customerID *string
	var shippingCost int64
	var paymentMethod *string
	err := pool.QueryRow(t.Context(), `
		SELECT customer_id, shipping_cost, payment_method FROM orders WHERE id = $1
	`, orderID).Scan(&customerID, &shippingCost, &paymentMethod)
	if err != nil {
		t.Fatalf("failed to read order: %v", err)
	}
	if customerID != nil {
		t.Errorf("customer_id = %v; want nil for walk-in", *customerID)
	}
	if shippingCost != 0 {
		t.Errorf("shipping_cost = %d; want 0 (not applicable to POS)", shippingCost)
	}
	if paymentMethod == nil || *paymentMethod != "cash" {
		t.Errorf("payment_method = %v; want cash", paymentMethod)
	}
}

func TestCreatePOSOrderWithCustomer(t *testing.T) {
	pool := testutil.NewTestDB(t)
	_, variantID := testutil.SeedProductVariant(t, pool, "Kaos", "kaos", 50000, 10)

	rec := postJSON(t, handlers.CreatePOSOrder(pool), "/pos/orders", map[string]any{
		"items":             []map[string]any{{"product_variant_id": variantID, "quantity": 1}},
		"customer_name":     "Budi",
		"customer_whatsapp": "081234567890",
	})

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, body = %s; want 201", rec.Code, rec.Body.String())
	}

	var name, waNumber string
	err := pool.QueryRow(t.Context(), `SELECT name, whatsapp_number FROM customers`).Scan(&name, &waNumber)
	if err != nil {
		t.Fatalf("failed to read customer: %v", err)
	}
	if name != "Budi" || waNumber != "6281234567890" {
		t.Errorf("customer = (%q, %q); want (Budi, 6281234567890)", name, waNumber)
	}
}

func TestCreatePOSOrderInsufficientStock(t *testing.T) {
	pool := testutil.NewTestDB(t)
	_, variantID := testutil.SeedProductVariant(t, pool, "Sepatu", "sepatu", 250000, 1)

	rec := postJSON(t, handlers.CreatePOSOrder(pool), "/pos/orders", map[string]any{
		"items": []map[string]any{{"product_variant_id": variantID, "quantity": 5}},
	})

	if rec.Code != http.StatusConflict {
		t.Fatalf("status = %d, body = %s; want 409", rec.Code, rec.Body.String())
	}

	var stock int32
	err := pool.QueryRow(t.Context(), `SELECT stock_quantity FROM product_variants WHERE id = $1`, variantID).Scan(&stock)
	if err != nil {
		t.Fatalf("failed to read stock: %v", err)
	}
	if stock != 1 {
		t.Errorf("stock_quantity = %d; want unchanged at 1", stock)
	}
}

func TestGetPOSOrder(t *testing.T) {
	pool := testutil.NewTestDB(t)
	_, variantID := testutil.SeedProductVariant(t, pool, "Kaos", "kaos", 85000, 10)

	created := decodeJSON(t, postJSON(t, handlers.CreatePOSOrder(pool), "/pos/orders", map[string]any{
		"items":          []map[string]any{{"product_variant_id": variantID, "quantity": 2}},
		"payment_method": "qris",
	}))
	orderID := created["order_id"].(string)

	rec := getJSON(t, handlers.GetPOSOrder(pool), "/pos/orders/"+orderID, map[string]string{"order_id": orderID})
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s; want 200", rec.Code, rec.Body.String())
	}
	body := decodeJSON(t, rec)
	if body["payment_method"] != "qris" {
		t.Errorf("payment_method = %v; want qris", body["payment_method"])
	}
	items, ok := body["items"].([]any)
	if !ok || len(items) != 1 {
		t.Fatalf("items = %v; want 1 item", body["items"])
	}
}

func TestGetPOSOrderNotFound(t *testing.T) {
	pool := testutil.NewTestDB(t)
	rec := getJSON(t, handlers.GetPOSOrder(pool), "/pos/orders/00000000-0000-0000-0000-000000000000",
		map[string]string{"order_id": "00000000-0000-0000-0000-000000000000"})
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d; want 404", rec.Code)
	}
}

func TestGetPOSOrderExcludesOnlineOrders(t *testing.T) {
	pool := testutil.NewTestDB(t)
	_, variantID := testutil.SeedProductVariant(t, pool, "Kaos", "kaos", 85000, 10)

	onlineOrder := decodeJSON(t, postJSON(t, handlers.Checkout(pool, nil), "/checkout", map[string]any{
		"name":            "Test",
		"whatsapp_number": "081111111111",
		"items":           []map[string]any{{"product_variant_id": variantID, "quantity": 1}},
	}))
	orderID := onlineOrder["order_id"].(string)

	rec := getJSON(t, handlers.GetPOSOrder(pool), "/pos/orders/"+orderID, map[string]string{"order_id": orderID})
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d; want 404 — GET /pos/orders/:id must not return an online-channel order", rec.Code)
	}
}

func TestGetVariantStock(t *testing.T) {
	pool := testutil.NewTestDB(t)
	_, variantID := testutil.SeedProductVariant(t, pool, "Kaos", "kaos", 85000, 7)

	rec := getJSON(t, handlers.GetVariantStock(pool), "/products/variants/"+variantID+"/stock", map[string]string{"variant_id": variantID})
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d; want 200", rec.Code)
	}
	body := decodeJSON(t, rec)
	if body["stock_quantity"].(float64) != 7 {
		t.Errorf("stock_quantity = %v; want 7", body["stock_quantity"])
	}
}

// TestCrossChannelStockRaceCondition is the single most important test in
// this suite: PRD section 7A requires that a simultaneous online checkout
// and POS sale for the same variant, with only 1 unit of stock left, can
// never both succeed. This fires 5 online checkouts and 5 POS orders at
// once against stock of exactly 1 and asserts exactly one order wins,
// stock lands at exactly 0 (never negative), and exactly one order_items
// row was written — the same scenario manually verified during
// development, now automated.
func TestCrossChannelStockRaceCondition(t *testing.T) {
	pool := testutil.NewTestDB(t)
	_, variantID := testutil.SeedProductVariant(t, pool, "Sneakers Terbatas", "sneakers-terbatas", 300000, 1)

	const attemptsPerChannel = 5
	var wg sync.WaitGroup
	statusCodes := make([]int, 0, attemptsPerChannel*2)
	var mu sync.Mutex

	record := func(code int) {
		mu.Lock()
		statusCodes = append(statusCodes, code)
		mu.Unlock()
	}

	for i := 0; i < attemptsPerChannel; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			rec := postJSON(t, handlers.Checkout(pool, nil), "/checkout", map[string]any{
				"name":            "Race",
				"whatsapp_number": "08111111111",
				"items":           []map[string]any{{"product_variant_id": variantID, "quantity": 1}},
			})
			record(rec.Code)
		}(i)
	}
	for i := 0; i < attemptsPerChannel; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			rec := postJSON(t, handlers.CreatePOSOrder(pool), "/pos/orders", map[string]any{
				"items": []map[string]any{{"product_variant_id": variantID, "quantity": 1}},
			})
			record(rec.Code)
		}(i)
	}
	wg.Wait()

	successCount := 0
	conflictCount := 0
	for _, code := range statusCodes {
		switch code {
		case http.StatusCreated:
			successCount++
		case http.StatusConflict:
			conflictCount++
		default:
			t.Errorf("unexpected status code %d in race", code)
		}
	}
	if successCount != 1 {
		t.Errorf("successCount = %d; want exactly 1 (stock was 1, %d total attempts)", successCount, attemptsPerChannel*2)
	}
	if conflictCount != attemptsPerChannel*2-1 {
		t.Errorf("conflictCount = %d; want %d", conflictCount, attemptsPerChannel*2-1)
	}

	var finalStock int32
	err := pool.QueryRow(t.Context(), `SELECT stock_quantity FROM product_variants WHERE id = $1`, variantID).Scan(&finalStock)
	if err != nil {
		t.Fatalf("failed to read final stock: %v", err)
	}
	if finalStock != 0 {
		t.Fatalf("final stock_quantity = %d; want exactly 0 (never negative, never left over)", finalStock)
	}

	var orderItemCount int
	err = pool.QueryRow(t.Context(), `SELECT count(*) FROM order_items WHERE product_variant_id = $1`, variantID).Scan(&orderItemCount)
	if err != nil {
		t.Fatalf("failed to count order_items: %v", err)
	}
	if orderItemCount != 1 {
		t.Fatalf("order_items count = %d; want exactly 1", orderItemCount)
	}
}
