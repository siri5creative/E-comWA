package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"math"
	"net/http"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/siri5creative/E-comWA/api/internal/httpx"
	"github.com/siri5creative/E-comWA/api/internal/models"
	"github.com/siri5creative/E-comWA/api/internal/util"
)

// queryer is satisfied by both *pgxpool.Pool and pgx.Tx, so coupon
// evaluation can run as a read-only preview (POST /coupons/validate,
// against the pool) or as the authoritative, lock-holding check inside a
// checkout transaction (against a tx) using the same code path.
type queryer interface {
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
}

type couponCartItem struct {
	ProductID string
	Price     int64
	Quantity  int32
}

type couponEvalResult struct {
	CouponID       string
	Code           string
	DiscountType   models.CouponDiscountType
	DiscountAmount int64
}

var couponInvalidReasonMessage = map[string]string{
	"not_found":            "Kode kupon tidak ditemukan",
	"inactive":             "Kupon tidak aktif",
	"not_yet_valid":        "Kupon belum berlaku",
	"expired":              "Kupon sudah kadaluarsa",
	"quota_exhausted":      "Kuota kupon sudah habis",
	"already_used":         "Kamu sudah pernah pakai kupon ini",
	"min_spend_not_met":    "Belanja belum mencapai minimum untuk kupon ini",
	"product_not_eligible": "Kupon ini tidak berlaku untuk produk di keranjangmu",
	"bundle_incomplete":    "Kupon paket ini butuh semua produk dalam paketnya di keranjang",
}

// evaluateCoupon checks a coupon code against all restrictions (PRD section
// 6.4) and computes the discount for the given cart, using a row lock
// (FOR UPDATE) so concurrent checkouts against the same coupon serialize
// correctly — the same atomicity principle as the stock decrement in
// checkout.go (PRD section 7A). Returns (result, "", nil) when valid, or
// (nil, reason, nil) when the coupon is legitimately invalid (not a system
// error — reason is a key into couponInvalidReasonMessage).
func evaluateCoupon(ctx context.Context, q queryer, rawCode, whatsappNumber string, items []couponCartItem) (*couponEvalResult, string, error) {
	code := strings.ToUpper(strings.TrimSpace(rawCode))
	if code == "" {
		return nil, "not_found", nil
	}

	var (
		couponID                        string
		discountType, discountValueType string
		discountValue                   float64
		minSpend                        int64
		validFrom, validUntil           time.Time
		maxTotalUsage                   *int32
		maxUsagePerCustomer             *int32
		currentUsageCount               int32
		isActive                        bool
	)

	err := q.QueryRow(ctx, `
		SELECT id, discount_type::text, discount_value_type::text, discount_value, min_spend,
		       valid_from, valid_until, max_total_usage, max_usage_per_customer, current_usage_count, is_active
		FROM coupons WHERE code = $1 FOR UPDATE
	`, code).Scan(&couponID, &discountType, &discountValueType, &discountValue, &minSpend,
		&validFrom, &validUntil, &maxTotalUsage, &maxUsagePerCustomer, &currentUsageCount, &isActive)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, "not_found", nil
		}
		return nil, "", err
	}

	if !isActive {
		return nil, "inactive", nil
	}
	now := time.Now()
	if now.Before(validFrom) {
		return nil, "not_yet_valid", nil
	}
	if now.After(validUntil) {
		return nil, "expired", nil
	}
	if maxTotalUsage != nil && currentUsageCount >= *maxTotalUsage {
		return nil, "quota_exhausted", nil
	}

	if maxUsagePerCustomer != nil && whatsappNumber != "" {
		var usageCount int32
		err := q.QueryRow(ctx, `
			SELECT count(*) FROM coupon_usages cu
			JOIN customers cust ON cust.id = cu.customer_id
			WHERE cu.coupon_id = $1 AND cust.whatsapp_number = $2
		`, couponID, whatsappNumber).Scan(&usageCount)
		if err != nil {
			return nil, "", err
		}
		if usageCount >= *maxUsagePerCustomer {
			return nil, "already_used", nil
		}
	}

	var cartSubtotal int64
	for _, it := range items {
		cartSubtotal += it.Price * int64(it.Quantity)
	}
	if minSpend > 0 && cartSubtotal < minSpend {
		return nil, "min_spend_not_met", nil
	}

	var applicableSubtotal int64
	switch models.CouponDiscountType(discountType) {
	case models.CouponDiscountTypeTotalBelanja, models.CouponDiscountTypeEvent:
		applicableSubtotal = cartSubtotal

	case models.CouponDiscountTypeItemTertentu:
		productIDs, err := fetchCouponProductIDs(ctx, q, couponID)
		if err != nil {
			return nil, "", err
		}
		matched := false
		for _, it := range items {
			if containsString(productIDs, it.ProductID) {
				applicableSubtotal += it.Price * int64(it.Quantity)
				matched = true
			}
		}
		if !matched {
			return nil, "product_not_eligible", nil
		}

	case models.CouponDiscountTypeBundle:
		productIDs, err := fetchCouponProductIDs(ctx, q, couponID)
		if err != nil {
			return nil, "", err
		}
		for _, pid := range productIDs {
			has := false
			for _, it := range items {
				if it.ProductID == pid {
					has = true
					applicableSubtotal += it.Price * int64(it.Quantity)
					break
				}
			}
			if !has {
				return nil, "bundle_incomplete", nil
			}
		}

	default:
		return nil, "", errors.New("unknown discount_type: " + discountType)
	}

	var discount int64
	if models.CouponDiscountValueType(discountValueType) == models.CouponDiscountValueTypePercentage {
		discount = int64(math.Round(float64(applicableSubtotal) * discountValue / 100))
	} else {
		discount = int64(math.Round(discountValue))
	}
	if discount > applicableSubtotal {
		discount = applicableSubtotal
	}
	if discount < 0 {
		discount = 0
	}

	return &couponEvalResult{
		CouponID:       couponID,
		Code:           code,
		DiscountType:   models.CouponDiscountType(discountType),
		DiscountAmount: discount,
	}, "", nil
}

func fetchCouponProductIDs(ctx context.Context, q queryer, couponID string) ([]string, error) {
	rows, err := q.Query(ctx, `SELECT product_id FROM coupon_products WHERE coupon_id = $1`, couponID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

func containsString(list []string, target string) bool {
	for _, s := range list {
		if s == target {
			return true
		}
	}
	return false
}

type validateCouponRequest struct {
	Code           string `json:"code"`
	WhatsAppNumber string `json:"whatsapp_number"`
	Items          []struct {
		ProductVariantID string `json:"product_variant_id"`
		Quantity         int32  `json:"quantity"`
	} `json:"items"`
}

// ValidateCoupon handles POST /coupons/validate — public (PRD section 6.4).
// Always responds 200 with a `valid` boolean; "invalid" is an expected
// business outcome (kadaluarsa/sudah dipakai/kuota habis), not an HTTP
// error. This is a read-only preview — it does not reserve/consume the
// coupon; only POST /checkout does that authoritatively.
func ValidateCoupon(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req validateCouponRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			httpx.WriteError(w, http.StatusBadRequest, "validation_error", "invalid request body")
			return
		}
		if strings.TrimSpace(req.Code) == "" {
			httpx.WriteError(w, http.StatusBadRequest, "validation_error", "code wajib diisi")
			return
		}

		waNumber := ""
		if req.WhatsAppNumber != "" {
			normalized, err := util.NormalizeWhatsAppNumber(req.WhatsAppNumber)
			if err != nil {
				httpx.WriteError(w, http.StatusBadRequest, "validation_error", err.Error())
				return
			}
			waNumber = normalized
		}

		ctx := r.Context()
		items, err := resolveCartItemsForCoupon(ctx, pool, req.Items)
		if err != nil {
			httpx.WriteError(w, http.StatusInternalServerError, "internal_error", "failed to resolve cart items")
			return
		}

		result, reason, err := evaluateCoupon(ctx, pool, req.Code, waNumber, items)
		if err != nil {
			httpx.WriteError(w, http.StatusInternalServerError, "internal_error", "failed to validate coupon")
			return
		}
		if reason != "" {
			httpx.WriteJSON(w, http.StatusOK, map[string]any{
				"valid":   false,
				"reason":  reason,
				"message": couponInvalidReasonMessage[reason],
			})
			return
		}

		httpx.WriteJSON(w, http.StatusOK, map[string]any{
			"valid":           true,
			"code":            result.Code,
			"discount_type":   result.DiscountType,
			"discount_amount": result.DiscountAmount,
		})
	}
}

// resolveCartItemsForCoupon looks up real price/product_id for each
// requested variant — never trusts client-supplied prices. Variant IDs
// that don't resolve are silently skipped (this endpoint is an advisory
// preview; POST /checkout is the authoritative check).
func resolveCartItemsForCoupon(ctx context.Context, pool *pgxpool.Pool, requested []struct {
	ProductVariantID string `json:"product_variant_id"`
	Quantity         int32  `json:"quantity"`
}) ([]couponCartItem, error) {
	if len(requested) == 0 {
		return nil, nil
	}

	ids := make([]string, len(requested))
	for i, it := range requested {
		ids[i] = it.ProductVariantID
	}

	rows, err := pool.Query(ctx, `SELECT id, product_id, price FROM product_variants WHERE id = ANY($1)`, ids)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	priceByVariant := map[string]int64{}
	productByVariant := map[string]string{}
	for rows.Next() {
		var variantID, productID string
		var price int64
		if err := rows.Scan(&variantID, &productID, &price); err != nil {
			return nil, err
		}
		priceByVariant[variantID] = price
		productByVariant[variantID] = productID
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	items := make([]couponCartItem, 0, len(requested))
	for _, it := range requested {
		price, ok := priceByVariant[it.ProductVariantID]
		if !ok {
			continue
		}
		items = append(items, couponCartItem{
			ProductID: productByVariant[it.ProductVariantID],
			Price:     price,
			Quantity:  it.Quantity,
		})
	}
	return items, nil
}
