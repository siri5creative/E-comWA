package handlers_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/siri5creative/E-comWA/api/internal/handlers"
	"github.com/siri5creative/E-comWA/api/internal/testutil"
)

func jsonReq(t *testing.T, method string, h http.HandlerFunc, path string, pathValues map[string]string, body any) *httptest.ResponseRecorder {
	t.Helper()
	raw, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("failed to marshal request body: %v", err)
	}
	req := httptest.NewRequest(method, path, bytes.NewReader(raw))
	req.Header.Set("Content-Type", "application/json")
	for k, v := range pathValues {
		req.SetPathValue(k, v)
	}
	rec := httptest.NewRecorder()
	h(rec, req)
	return rec
}

func seedOrderItem(t *testing.T, pool *pgxpool.Pool, variantID string) {
	t.Helper()
	orderID := seedOrder(t, pool, "selesai", 100000, 0, 0)
	_, err := pool.Exec(t.Context(), `
		INSERT INTO order_items (order_id, product_variant_id, quantity, price_at_purchase)
		VALUES ($1, $2, 1, 100000)
	`, orderID, variantID)
	if err != nil {
		t.Fatalf("failed to seed order item: %v", err)
	}
}

func TestCreateProductSuccess(t *testing.T) {
	pool := testutil.NewTestDB(t)

	rec := jsonReq(t, http.MethodPost, handlers.CreateProduct(pool), "/products", nil, map[string]any{
		"name": "Kaos Polos",
		"slug": "kaos-polos",
	})
	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, body = %s; want 201", rec.Code, rec.Body.String())
	}

	var count int
	if err := pool.QueryRow(t.Context(), `SELECT count(*) FROM products WHERE slug = 'kaos-polos'`).Scan(&count); err != nil {
		t.Fatalf("failed to count products: %v", err)
	}
	if count != 1 {
		t.Errorf("products with slug kaos-polos = %d; want 1", count)
	}
}

func TestCreateProductDuplicateSlug(t *testing.T) {
	pool := testutil.NewTestDB(t)
	testutil.SeedProductVariant(t, pool, "Kaos Polos", "kaos-polos", 100000, 1)

	rec := jsonReq(t, http.MethodPost, handlers.CreateProduct(pool), "/products", nil, map[string]any{
		"name": "Kaos Polos Lagi",
		"slug": "kaos-polos",
	})
	if rec.Code != http.StatusConflict {
		t.Fatalf("status = %d, body = %s; want 409", rec.Code, rec.Body.String())
	}
}

func TestCreateProductMissingName(t *testing.T) {
	pool := testutil.NewTestDB(t)

	rec := jsonReq(t, http.MethodPost, handlers.CreateProduct(pool), "/products", nil, map[string]any{
		"slug": "tanpa-nama",
	})
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, body = %s; want 400", rec.Code, rec.Body.String())
	}
}

func TestUpdateProductSuccess(t *testing.T) {
	pool := testutil.NewTestDB(t)
	productID, _ := testutil.SeedProductVariant(t, pool, "Kaos Polos", "kaos-polos", 100000, 1)

	rec := jsonReq(t, http.MethodPut, handlers.UpdateProduct(pool), "/products/"+productID,
		map[string]string{"id": productID}, map[string]any{
			"name": "Kaos Polos v2",
			"slug": "kaos-polos-v2",
		})
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s; want 200", rec.Code, rec.Body.String())
	}

	var name string
	if err := pool.QueryRow(t.Context(), `SELECT name FROM products WHERE id = $1`, productID).Scan(&name); err != nil {
		t.Fatalf("failed to read product: %v", err)
	}
	if name != "Kaos Polos v2" {
		t.Errorf("name = %q; want %q", name, "Kaos Polos v2")
	}
}

func TestUpdateProductNotFound(t *testing.T) {
	pool := testutil.NewTestDB(t)

	rec := jsonReq(t, http.MethodPut, handlers.UpdateProduct(pool), "/products/00000000-0000-0000-0000-000000000000",
		map[string]string{"id": "00000000-0000-0000-0000-000000000000"}, map[string]any{
			"name": "Tidak Ada",
			"slug": "tidak-ada",
		})
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, body = %s; want 404", rec.Code, rec.Body.String())
	}
}

func TestDeleteProductSuccess(t *testing.T) {
	pool := testutil.NewTestDB(t)
	productID, _ := testutil.SeedProductVariant(t, pool, "Kaos Polos", "kaos-polos", 100000, 1)

	rec := jsonReq(t, http.MethodDelete, handlers.DeleteProduct(pool), "/products/"+productID,
		map[string]string{"id": productID}, nil)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("status = %d, body = %s; want 204", rec.Code, rec.Body.String())
	}
}

// TestDeleteProductWithOrderHistoryFails proves the RESTRICT foreign key
// (order_items.product_variant_id) surfaces as a friendly 409, not a raw 500.
func TestDeleteProductWithOrderHistoryFails(t *testing.T) {
	pool := testutil.NewTestDB(t)
	productID, variantID := testutil.SeedProductVariant(t, pool, "Kaos Polos", "kaos-polos", 100000, 1)
	seedOrderItem(t, pool, variantID)

	rec := jsonReq(t, http.MethodDelete, handlers.DeleteProduct(pool), "/products/"+productID,
		map[string]string{"id": productID}, nil)
	if rec.Code != http.StatusConflict {
		t.Fatalf("status = %d, body = %s; want 409", rec.Code, rec.Body.String())
	}
}

func TestCreateVariantSuccess(t *testing.T) {
	pool := testutil.NewTestDB(t)
	productID, _ := testutil.SeedProductVariant(t, pool, "Kaos Polos", "kaos-polos", 100000, 1)

	rec := jsonReq(t, http.MethodPost, handlers.CreateVariant(pool), "/products/"+productID+"/variants",
		map[string]string{"id": productID}, map[string]any{
			"variant_name":   "Merah / L",
			"price":          120000,
			"stock_quantity": 10,
		})
	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, body = %s; want 201", rec.Code, rec.Body.String())
	}

	var count int
	if err := pool.QueryRow(t.Context(), `SELECT count(*) FROM product_variants WHERE product_id = $1`, productID).Scan(&count); err != nil {
		t.Fatalf("failed to count variants: %v", err)
	}
	if count != 2 {
		t.Errorf("variants for product = %d; want 2 (1 seeded + 1 created)", count)
	}
}

func TestUpdateVariantStockQuantity(t *testing.T) {
	pool := testutil.NewTestDB(t)
	_, variantID := testutil.SeedProductVariant(t, pool, "Kaos Polos", "kaos-polos", 100000, 1)

	rec := jsonReq(t, http.MethodPut, handlers.UpdateVariant(pool), "/products/variants/"+variantID,
		map[string]string{"variant_id": variantID}, map[string]any{
			"variant_name":   "Default",
			"price":          100000,
			"stock_quantity": 50,
		})
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s; want 200", rec.Code, rec.Body.String())
	}

	var stock int32
	if err := pool.QueryRow(t.Context(), `SELECT stock_quantity FROM product_variants WHERE id = $1`, variantID).Scan(&stock); err != nil {
		t.Fatalf("failed to read variant: %v", err)
	}
	if stock != 50 {
		t.Errorf("stock_quantity = %d; want 50", stock)
	}
}

func TestDeleteVariantWithOrderHistoryFails(t *testing.T) {
	pool := testutil.NewTestDB(t)
	_, variantID := testutil.SeedProductVariant(t, pool, "Kaos Polos", "kaos-polos", 100000, 1)
	seedOrderItem(t, pool, variantID)

	rec := jsonReq(t, http.MethodDelete, handlers.DeleteVariant(pool), "/products/variants/"+variantID,
		map[string]string{"variant_id": variantID}, nil)
	if rec.Code != http.StatusConflict {
		t.Fatalf("status = %d, body = %s; want 409", rec.Code, rec.Body.String())
	}
}
