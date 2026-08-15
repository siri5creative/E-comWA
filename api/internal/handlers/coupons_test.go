package handlers_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/siri5creative/E-comWA/api/internal/handlers"
	"github.com/siri5creative/E-comWA/api/internal/middleware"
	"github.com/siri5creative/E-comWA/api/internal/testutil"
)

func timeDaysAgo(days int) time.Time {
	return time.Now().Add(-time.Duration(days) * 24 * time.Hour)
}

func timeDaysFromNow(days int) time.Time {
	return time.Now().Add(time.Duration(days) * 24 * time.Hour)
}

func TestValidateCouponTotalBelanjaFixed(t *testing.T) {
	pool := testutil.NewTestDB(t)
	_, variantID := testutil.SeedProductVariant(t, pool, "Kaos", "kaos", 100000, 10)
	testutil.SeedCoupon(t, pool, "HEMAT20K", "total_belanja", "fixed", 20000, testutil.CouponOptions{MinSpend: 150000})

	rec := postJSON(t, handlers.ValidateCoupon(pool), "/coupons/validate", map[string]any{
		"code":            "HEMAT20K",
		"whatsapp_number": "081111111111",
		"items":           []map[string]any{{"product_variant_id": variantID, "quantity": 2}},
	})

	body := decodeJSON(t, rec)
	if valid, _ := body["valid"].(bool); !valid {
		t.Fatalf("expected valid=true, got body: %v", body)
	}
	if body["discount_amount"].(float64) != 20000 {
		t.Errorf("discount_amount = %v; want 20000", body["discount_amount"])
	}
}

func TestValidateCouponMinSpendNotMet(t *testing.T) {
	pool := testutil.NewTestDB(t)
	_, variantID := testutil.SeedProductVariant(t, pool, "Kaos", "kaos", 100000, 10)
	testutil.SeedCoupon(t, pool, "HEMAT20K", "total_belanja", "fixed", 20000, testutil.CouponOptions{MinSpend: 150000})

	rec := postJSON(t, handlers.ValidateCoupon(pool), "/coupons/validate", map[string]any{
		"code":            "HEMAT20K",
		"whatsapp_number": "081111111111",
		"items":           []map[string]any{{"product_variant_id": variantID, "quantity": 1}},
	})

	body := decodeJSON(t, rec)
	assertInvalidCoupon(t, body, "min_spend_not_met")
}

func TestValidateCouponPercentageDiscount(t *testing.T) {
	pool := testutil.NewTestDB(t)
	_, variantID := testutil.SeedProductVariant(t, pool, "Kaos", "kaos", 100000, 10)
	testutil.SeedCoupon(t, pool, "DISKON15", "event", "percentage", 15, testutil.CouponOptions{})

	rec := postJSON(t, handlers.ValidateCoupon(pool), "/coupons/validate", map[string]any{
		"code":            "DISKON15",
		"whatsapp_number": "081111111111",
		"items":           []map[string]any{{"product_variant_id": variantID, "quantity": 2}},
	})

	body := decodeJSON(t, rec)
	if valid, _ := body["valid"].(bool); !valid {
		t.Fatalf("expected valid=true, got body: %v", body)
	}
	// 15% of (100000*2 = 200000) = 30000
	if body["discount_amount"].(float64) != 30000 {
		t.Errorf("discount_amount = %v; want 30000", body["discount_amount"])
	}
}

func TestValidateCouponExpired(t *testing.T) {
	pool := testutil.NewTestDB(t)
	_, variantID := testutil.SeedProductVariant(t, pool, "Kaos", "kaos", 100000, 10)
	testutil.SeedCoupon(t, pool, "EXPIRED", "event", "percentage", 10, testutil.CouponOptions{
		ValidFrom:  timeDaysAgo(10),
		ValidUntil: timeDaysAgo(1),
	})

	rec := postJSON(t, handlers.ValidateCoupon(pool), "/coupons/validate", map[string]any{
		"code":            "EXPIRED",
		"whatsapp_number": "081111111111",
		"items":           []map[string]any{{"product_variant_id": variantID, "quantity": 1}},
	})

	body := decodeJSON(t, rec)
	assertInvalidCoupon(t, body, "expired")
}

func TestValidateCouponNotYetValid(t *testing.T) {
	pool := testutil.NewTestDB(t)
	_, variantID := testutil.SeedProductVariant(t, pool, "Kaos", "kaos", 100000, 10)
	testutil.SeedCoupon(t, pool, "FUTURE", "event", "percentage", 10, testutil.CouponOptions{
		ValidFrom:  timeDaysFromNow(1),
		ValidUntil: timeDaysFromNow(10),
	})

	rec := postJSON(t, handlers.ValidateCoupon(pool), "/coupons/validate", map[string]any{
		"code":            "FUTURE",
		"whatsapp_number": "081111111111",
		"items":           []map[string]any{{"product_variant_id": variantID, "quantity": 1}},
	})

	body := decodeJSON(t, rec)
	assertInvalidCoupon(t, body, "not_yet_valid")
}

func TestValidateCouponInactive(t *testing.T) {
	pool := testutil.NewTestDB(t)
	_, variantID := testutil.SeedProductVariant(t, pool, "Kaos", "kaos", 100000, 10)
	testutil.SeedCoupon(t, pool, "NONAKTIF", "total_belanja", "fixed", 5000, testutil.CouponOptions{Inactive: true})

	rec := postJSON(t, handlers.ValidateCoupon(pool), "/coupons/validate", map[string]any{
		"code":            "NONAKTIF",
		"whatsapp_number": "081111111111",
		"items":           []map[string]any{{"product_variant_id": variantID, "quantity": 1}},
	})

	body := decodeJSON(t, rec)
	assertInvalidCoupon(t, body, "inactive")
}

func TestValidateCouponNotFound(t *testing.T) {
	pool := testutil.NewTestDB(t)

	rec := postJSON(t, handlers.ValidateCoupon(pool), "/coupons/validate", map[string]any{
		"code":            "TIDAKADA",
		"whatsapp_number": "081111111111",
		"items":           []map[string]any{},
	})

	body := decodeJSON(t, rec)
	assertInvalidCoupon(t, body, "not_found")
}

func TestValidateCouponQuotaExhausted(t *testing.T) {
	pool := testutil.NewTestDB(t)
	_, variantID := testutil.SeedProductVariant(t, pool, "Kaos", "kaos", 100000, 10)
	maxUsage := int32(1)
	testutil.SeedCoupon(t, pool, "SEKALI", "total_belanja", "fixed", 5000, testutil.CouponOptions{MaxTotalUsage: &maxUsage})

	// Use the coupon once via checkout to bump current_usage_count to 1.
	rec1 := postJSON(t, handlers.Checkout(pool, nil), "/checkout", map[string]any{
		"name":            "Pertama",
		"whatsapp_number": "081111111111",
		"items":           []map[string]any{{"product_variant_id": variantID, "quantity": 1}},
		"coupon_code":     "SEKALI",
	})
	if rec1.Code != http.StatusCreated {
		t.Fatalf("setup checkout failed: status = %d, body = %s", rec1.Code, rec1.Body.String())
	}

	rec2 := postJSON(t, handlers.ValidateCoupon(pool), "/coupons/validate", map[string]any{
		"code":            "SEKALI",
		"whatsapp_number": "082222222222",
		"items":           []map[string]any{{"product_variant_id": variantID, "quantity": 1}},
	})
	body := decodeJSON(t, rec2)
	assertInvalidCoupon(t, body, "quota_exhausted")
}

func TestValidateCouponAlreadyUsedByThisCustomer(t *testing.T) {
	pool := testutil.NewTestDB(t)
	_, variantID := testutil.SeedProductVariant(t, pool, "Kaos", "kaos", 100000, 10)
	maxPerCustomer := int32(1)
	testutil.SeedCoupon(t, pool, "SEKALIORANG", "total_belanja", "fixed", 5000, testutil.CouponOptions{MaxUsagePerCustomer: &maxPerCustomer})

	rec1 := postJSON(t, handlers.Checkout(pool, nil), "/checkout", map[string]any{
		"name":            "Sama",
		"whatsapp_number": "081111111111",
		"items":           []map[string]any{{"product_variant_id": variantID, "quantity": 1}},
		"coupon_code":     "SEKALIORANG",
	})
	if rec1.Code != http.StatusCreated {
		t.Fatalf("setup checkout failed: status = %d, body = %s", rec1.Code, rec1.Body.String())
	}

	// Same customer (same WhatsApp number) tries again — must be rejected.
	rec2 := postJSON(t, handlers.ValidateCoupon(pool), "/coupons/validate", map[string]any{
		"code":            "SEKALIORANG",
		"whatsapp_number": "081111111111",
		"items":           []map[string]any{{"product_variant_id": variantID, "quantity": 1}},
	})
	body := decodeJSON(t, rec2)
	assertInvalidCoupon(t, body, "already_used")

	// A different customer must still be able to use it.
	rec3 := postJSON(t, handlers.ValidateCoupon(pool), "/coupons/validate", map[string]any{
		"code":            "SEKALIORANG",
		"whatsapp_number": "089999999999",
		"items":           []map[string]any{{"product_variant_id": variantID, "quantity": 1}},
	})
	body3 := decodeJSON(t, rec3)
	if valid, _ := body3["valid"].(bool); !valid {
		t.Errorf("a different customer should still be able to use the coupon, got: %v", body3)
	}
}

func TestValidateCouponItemTertentuRequiresMatchingProduct(t *testing.T) {
	pool := testutil.NewTestDB(t)
	kaosProductID, kaosVariantID := testutil.SeedProductVariant(t, pool, "Kaos", "kaos", 100000, 10)
	_, celanaVariantID := testutil.SeedProductVariant(t, pool, "Celana", "celana", 100000, 10)
	testutil.SeedCoupon(t, pool, "KAOSDISKON", "item_tertentu", "percentage", 10, testutil.CouponOptions{
		ProductIDs: []string{kaosProductID},
	})

	// Cart has only the non-matching product.
	recMiss := postJSON(t, handlers.ValidateCoupon(pool), "/coupons/validate", map[string]any{
		"code":            "KAOSDISKON",
		"whatsapp_number": "081111111111",
		"items":           []map[string]any{{"product_variant_id": celanaVariantID, "quantity": 1}},
	})
	assertInvalidCoupon(t, decodeJSON(t, recMiss), "product_not_eligible")

	// Cart has both — discount should only apply to the Kaos line.
	recMatch := postJSON(t, handlers.ValidateCoupon(pool), "/coupons/validate", map[string]any{
		"code":            "KAOSDISKON",
		"whatsapp_number": "081111111111",
		"items": []map[string]any{
			{"product_variant_id": kaosVariantID, "quantity": 2},
			{"product_variant_id": celanaVariantID, "quantity": 1},
		},
	})
	body := decodeJSON(t, recMatch)
	if valid, _ := body["valid"].(bool); !valid {
		t.Fatalf("expected valid=true, got: %v", body)
	}
	// 10% of just the Kaos line (100000*2=200000) = 20000, not the Celana line.
	if body["discount_amount"].(float64) != 20000 {
		t.Errorf("discount_amount = %v; want 20000 (scoped to matching product only)", body["discount_amount"])
	}
}

func TestValidateCouponBundleRequiresAllProducts(t *testing.T) {
	pool := testutil.NewTestDB(t)
	kaosProductID, kaosVariantID := testutil.SeedProductVariant(t, pool, "Kaos", "kaos", 100000, 10)
	celanaProductID, celanaVariantID := testutil.SeedProductVariant(t, pool, "Celana", "celana", 100000, 10)
	testutil.SeedCoupon(t, pool, "PAKETHEMAT", "bundle", "fixed", 30000, testutil.CouponOptions{
		ProductIDs: []string{kaosProductID, celanaProductID},
	})

	// Only one of the two bundle products in the cart.
	recIncomplete := postJSON(t, handlers.ValidateCoupon(pool), "/coupons/validate", map[string]any{
		"code":            "PAKETHEMAT",
		"whatsapp_number": "081111111111",
		"items":           []map[string]any{{"product_variant_id": kaosVariantID, "quantity": 1}},
	})
	assertInvalidCoupon(t, decodeJSON(t, recIncomplete), "bundle_incomplete")

	// Both bundle products present.
	recComplete := postJSON(t, handlers.ValidateCoupon(pool), "/coupons/validate", map[string]any{
		"code":            "PAKETHEMAT",
		"whatsapp_number": "081111111111",
		"items": []map[string]any{
			{"product_variant_id": kaosVariantID, "quantity": 1},
			{"product_variant_id": celanaVariantID, "quantity": 1},
		},
	})
	body := decodeJSON(t, recComplete)
	if valid, _ := body["valid"].(bool); !valid {
		t.Fatalf("expected valid=true once bundle is complete, got: %v", body)
	}
	if body["discount_amount"].(float64) != 30000 {
		t.Errorf("discount_amount = %v; want 30000", body["discount_amount"])
	}
}

func TestValidateCouponCodeIsCaseInsensitive(t *testing.T) {
	pool := testutil.NewTestDB(t)
	_, variantID := testutil.SeedProductVariant(t, pool, "Kaos", "kaos", 100000, 10)
	testutil.SeedCoupon(t, pool, "UPPERCASE10", "total_belanja", "fixed", 10000, testutil.CouponOptions{})

	rec := postJSON(t, handlers.ValidateCoupon(pool), "/coupons/validate", map[string]any{
		"code":            "uppercase10",
		"whatsapp_number": "081111111111",
		"items":           []map[string]any{{"product_variant_id": variantID, "quantity": 1}},
	})

	body := decodeJSON(t, rec)
	if valid, _ := body["valid"].(bool); !valid {
		t.Fatalf("expected lowercase input to match uppercase-stored code, got: %v", body)
	}
}

// TestCreateCouponFullAuthChain exercises the real RequireOwner middleware
// (not just the handler in isolation) to prove role enforcement and coupon
// creation work together end to end — a Staff admin must be rejected, and
// an Owner's request must persist the coupon and its product links.
func TestCreateCouponFullAuthChain(t *testing.T) {
	pool := testutil.NewTestDB(t)
	const secret = "test-secret"
	adminAuth := &middleware.AdminAuth{Pool: pool, JWTSecret: secret}

	_, ownerAuthID := testutil.SeedAdmin(t, pool, "Owner Toko", "owner")
	_, staffAuthID := testutil.SeedAdmin(t, pool, "Staff Toko", "staff")
	productID, _ := testutil.SeedProductVariant(t, pool, "Kaos", "kaos", 100000, 10)

	ownerToken := testutil.MintAdminJWT(t, ownerAuthID, secret)
	staffToken := testutil.MintAdminJWT(t, staffAuthID, secret)

	reqBody := map[string]any{
		"code":                "BARUOWNER",
		"discount_type":       "item_tertentu",
		"discount_value_type": "fixed",
		"discount_value":      5000,
		"valid_from":          "2026-01-01T00:00:00Z",
		"valid_until":         "2026-12-31T00:00:00Z",
		"product_ids":         []string{productID},
	}
	handler := adminAuth.RequireOwner(handlers.CreateCoupon(pool))

	staffRec := postJSONWithAuth(t, handler.ServeHTTP, "/coupons", reqBody, staffToken)
	if staffRec.Code != http.StatusForbidden {
		t.Fatalf("staff create coupon status = %d, body = %s; want 403", staffRec.Code, staffRec.Body.String())
	}

	ownerRec := postJSONWithAuth(t, handler.ServeHTTP, "/coupons", reqBody, ownerToken)
	if ownerRec.Code != http.StatusCreated {
		t.Fatalf("owner create coupon status = %d, body = %s; want 201", ownerRec.Code, ownerRec.Body.String())
	}

	var linkedProducts int
	err := pool.QueryRow(t.Context(), `
		SELECT count(*) FROM coupon_products cp
		JOIN coupons c ON c.id = cp.coupon_id
		WHERE c.code = 'BARUOWNER'
	`).Scan(&linkedProducts)
	if err != nil {
		t.Fatalf("failed to count coupon_products: %v", err)
	}
	if linkedProducts != 1 {
		t.Errorf("coupon_products rows = %d; want 1", linkedProducts)
	}
}

func assertInvalidCoupon(t *testing.T, body map[string]any, wantReason string) {
	t.Helper()
	if valid, _ := body["valid"].(bool); valid {
		t.Fatalf("expected valid=false with reason %q, got valid=true: %v", wantReason, body)
	}
	if body["reason"] != wantReason {
		t.Errorf("reason = %v; want %q (body: %v)", body["reason"], wantReason, body)
	}
}

func postJSONWithAuth(t *testing.T, h http.HandlerFunc, path string, body any, token string) *httptest.ResponseRecorder {
	t.Helper()
	raw, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("failed to marshal request body: %v", err)
	}
	req := httptest.NewRequest(http.MethodPost, path, bytes.NewReader(raw))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	h(rec, req)
	return rec
}
