package testutil

import (
	"context"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// SeedProductVariant creates a product with a single variant and returns
// both IDs — the common case for tests that don't care about multi-variant
// products.
func SeedProductVariant(t *testing.T, pool *pgxpool.Pool, name, slug string, price int64, stock int32) (productID, variantID string) {
	t.Helper()
	ctx := context.Background()

	err := pool.QueryRow(ctx, `
		INSERT INTO products (name, slug) VALUES ($1, $2) RETURNING id
	`, name, slug).Scan(&productID)
	if err != nil {
		t.Fatalf("failed to seed product %s: %v", name, err)
	}

	err = pool.QueryRow(ctx, `
		INSERT INTO product_variants (product_id, variant_name, price, stock_quantity)
		VALUES ($1, 'Default', $2, $3) RETURNING id
	`, productID, price, stock).Scan(&variantID)
	if err != nil {
		t.Fatalf("failed to seed variant for product %s: %v", name, err)
	}

	return productID, variantID
}

// SeedAdmin creates a stub auth.users row plus the matching admins row.
func SeedAdmin(t *testing.T, pool *pgxpool.Pool, name string, role string) (adminID, authUserID string) {
	t.Helper()
	ctx := context.Background()

	err := pool.QueryRow(ctx, `INSERT INTO auth.users DEFAULT VALUES RETURNING id`).Scan(&authUserID)
	if err != nil {
		t.Fatalf("failed to seed auth.users row: %v", err)
	}

	err = pool.QueryRow(ctx, `
		INSERT INTO admins (auth_user_id, name, role) VALUES ($1, $2, $3) RETURNING id
	`, authUserID, name, role).Scan(&adminID)
	if err != nil {
		t.Fatalf("failed to seed admin %s: %v", name, err)
	}

	return adminID, authUserID
}

// CouponOptions configures SeedCoupon; zero values mean "no restriction"
// (and the coupon is active by default — set Inactive to opt out, so the
// common case doesn't need to remember to set a flag).
type CouponOptions struct {
	MinSpend              int64
	MaxTotalUsage         *int32
	MaxUsagePerCustomer   *int32
	Inactive              bool
	ValidFrom, ValidUntil time.Time
	ProductIDs            []string
}

// SeedCoupon creates a coupon row (and coupon_products rows if
// ProductIDs is set). ValidFrom/ValidUntil default to a wide-open window
// (yesterday to a year from now) if left zero.
func SeedCoupon(t *testing.T, pool *pgxpool.Pool, code, discountType, discountValueType string, discountValue float64, opts CouponOptions) string {
	t.Helper()
	ctx := context.Background()

	validFrom := opts.ValidFrom
	if validFrom.IsZero() {
		validFrom = time.Now().Add(-24 * time.Hour)
	}
	validUntil := opts.ValidUntil
	if validUntil.IsZero() {
		validUntil = time.Now().Add(365 * 24 * time.Hour)
	}

	var couponID string
	err := pool.QueryRow(ctx, `
		INSERT INTO coupons (code, discount_type, discount_value_type, discount_value, min_spend,
		                      valid_from, valid_until, max_total_usage, max_usage_per_customer, is_active)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		RETURNING id
	`, code, discountType, discountValueType, discountValue, opts.MinSpend,
		validFrom, validUntil, opts.MaxTotalUsage, opts.MaxUsagePerCustomer, !opts.Inactive).Scan(&couponID)
	if err != nil {
		t.Fatalf("failed to seed coupon %s: %v", code, err)
	}

	for _, productID := range opts.ProductIDs {
		if _, err := pool.Exec(ctx, `
			INSERT INTO coupon_products (coupon_id, product_id) VALUES ($1, $2)
		`, couponID, productID); err != nil {
			t.Fatalf("failed to link coupon %s to product %s: %v", code, productID, err)
		}
	}

	return couponID
}
