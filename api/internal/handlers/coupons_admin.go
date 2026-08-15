package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/siri5creative/E-comWA/api/internal/httpx"
	"github.com/siri5creative/E-comWA/api/internal/middleware"
	"github.com/siri5creative/E-comWA/api/internal/models"
)

// ListCoupons handles GET /coupons — Owner only. Not in the PRD section 8
// draft table (only POST/PUT/DELETE are listed), but a management UI can't
// function without a way to list what exists.
func ListCoupons(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		rows, err := pool.Query(r.Context(), `
			SELECT id, code, discount_type::text, discount_value_type::text, discount_value, min_spend,
			       valid_from, valid_until, max_total_usage, max_usage_per_customer, current_usage_count,
			       is_active, created_at
			FROM coupons
			ORDER BY created_at DESC
		`)
		if err != nil {
			httpx.WriteError(w, http.StatusInternalServerError, "internal_error", "failed to list coupons")
			return
		}
		defer rows.Close()

		coupons := []models.Coupon{}
		for rows.Next() {
			var c models.Coupon
			if err := rows.Scan(&c.ID, &c.Code, &c.DiscountType, &c.DiscountValueType, &c.DiscountValue, &c.MinSpend,
				&c.ValidFrom, &c.ValidUntil, &c.MaxTotalUsage, &c.MaxUsagePerCustomer, &c.CurrentUsageCount,
				&c.IsActive, &c.CreatedAt); err != nil {
				httpx.WriteError(w, http.StatusInternalServerError, "internal_error", "failed to read coupons")
				return
			}
			coupons = append(coupons, c)
		}
		if err := rows.Err(); err != nil {
			httpx.WriteError(w, http.StatusInternalServerError, "internal_error", "failed to read coupons")
			return
		}

		httpx.WriteJSON(w, http.StatusOK, map[string]any{"data": coupons})
	}
}

// GetCoupon handles GET /coupons/:id — Owner only.
func GetCoupon(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")
		ctx := r.Context()

		c, err := loadCoupon(ctx, pool, id)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				httpx.WriteError(w, http.StatusNotFound, "not_found", "kupon tidak ditemukan")
				return
			}
			httpx.WriteError(w, http.StatusInternalServerError, "internal_error", "failed to load coupon")
			return
		}

		httpx.WriteJSON(w, http.StatusOK, c)
	}
}

func loadCoupon(ctx context.Context, pool *pgxpool.Pool, id string) (*models.Coupon, error) {
	var c models.Coupon
	err := pool.QueryRow(ctx, `
		SELECT id, code, discount_type::text, discount_value_type::text, discount_value, min_spend,
		       valid_from, valid_until, max_total_usage, max_usage_per_customer, current_usage_count,
		       is_active, created_at
		FROM coupons WHERE id = $1
	`, id).Scan(&c.ID, &c.Code, &c.DiscountType, &c.DiscountValueType, &c.DiscountValue, &c.MinSpend,
		&c.ValidFrom, &c.ValidUntil, &c.MaxTotalUsage, &c.MaxUsagePerCustomer, &c.CurrentUsageCount,
		&c.IsActive, &c.CreatedAt)
	if err != nil {
		return nil, err
	}

	productIDs, err := fetchCouponProductIDs(ctx, pool, c.ID)
	if err != nil {
		return nil, err
	}
	c.ProductIDs = productIDs

	return &c, nil
}

type couponRequest struct {
	Code                string   `json:"code"`
	DiscountType        string   `json:"discount_type"`
	DiscountValueType   string   `json:"discount_value_type"`
	DiscountValue       float64  `json:"discount_value"`
	MinSpend            int64    `json:"min_spend"`
	ValidFrom           string   `json:"valid_from"`
	ValidUntil          string   `json:"valid_until"`
	MaxTotalUsage       *int32   `json:"max_total_usage"`
	MaxUsagePerCustomer *int32   `json:"max_usage_per_customer"`
	IsActive            *bool    `json:"is_active"`
	ProductIDs          []string `json:"product_ids"`
}

func (req *couponRequest) validate() (string, bool) {
	if strings.TrimSpace(req.Code) == "" {
		return "code wajib diisi", false
	}
	if !models.ValidCouponDiscountTypes[models.CouponDiscountType(req.DiscountType)] {
		return "discount_type tidak valid", false
	}
	if !models.ValidCouponDiscountValueTypes[models.CouponDiscountValueType(req.DiscountValueType)] {
		return "discount_value_type tidak valid", false
	}
	if req.DiscountValue <= 0 {
		return "discount_value harus lebih dari 0", false
	}
	if req.DiscountValueType == string(models.CouponDiscountValueTypePercentage) && req.DiscountValue > 100 {
		return "discount_value persentase tidak boleh lebih dari 100", false
	}
	if req.MinSpend < 0 {
		return "min_spend tidak boleh negatif", false
	}
	if _, err := time.Parse(time.RFC3339, req.ValidFrom); err != nil {
		return "valid_from harus format tanggal RFC3339", false
	}
	if _, err := time.Parse(time.RFC3339, req.ValidUntil); err != nil {
		return "valid_until harus format tanggal RFC3339", false
	}
	needsProducts := req.DiscountType == string(models.CouponDiscountTypeItemTertentu) ||
		req.DiscountType == string(models.CouponDiscountTypeBundle)
	if needsProducts && len(req.ProductIDs) == 0 {
		return "product_ids wajib diisi untuk discount_type item_tertentu/bundle", false
	}
	if req.MaxTotalUsage != nil && *req.MaxTotalUsage <= 0 {
		return "max_total_usage harus lebih dari 0", false
	}
	if req.MaxUsagePerCustomer != nil && *req.MaxUsagePerCustomer <= 0 {
		return "max_usage_per_customer harus lebih dari 0", false
	}
	return "", true
}

// CreateCoupon handles POST /coupons — Owner only (PRD section 6.4: "Alur
// bikin kupon (Admin - Owner only)"). New coupons are active immediately
// per their own valid_from/valid_until, matching the PRD's "langsung aktif
// sesuai tanggal yang diset".
func CreateCoupon(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req couponRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			httpx.WriteError(w, http.StatusBadRequest, "validation_error", "invalid request body")
			return
		}
		if msg, ok := req.validate(); !ok {
			httpx.WriteError(w, http.StatusBadRequest, "validation_error", msg)
			return
		}

		admin, _ := middleware.AdminFromContext(r.Context())
		code := strings.ToUpper(strings.TrimSpace(req.Code))
		isActive := true
		if req.IsActive != nil {
			isActive = *req.IsActive
		}

		ctx := r.Context()
		tx, err := pool.Begin(ctx)
		if err != nil {
			httpx.WriteError(w, http.StatusInternalServerError, "internal_error", "failed to start transaction")
			return
		}
		defer tx.Rollback(ctx)

		var couponID string
		err = tx.QueryRow(ctx, `
			INSERT INTO coupons (code, discount_type, discount_value_type, discount_value, min_spend,
			                      valid_from, valid_until, max_total_usage, max_usage_per_customer, is_active, created_by)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
			RETURNING id
		`, code, req.DiscountType, req.DiscountValueType, req.DiscountValue, req.MinSpend,
			req.ValidFrom, req.ValidUntil, req.MaxTotalUsage, req.MaxUsagePerCustomer, isActive, admin.ID).Scan(&couponID)
		if err != nil {
			if isUniqueViolation(err) {
				httpx.WriteError(w, http.StatusConflict, "duplicate_code", "kode kupon sudah dipakai")
				return
			}
			httpx.WriteError(w, http.StatusInternalServerError, "internal_error", "failed to create coupon")
			return
		}

		if err := replaceCouponProducts(ctx, tx, couponID, req.ProductIDs); err != nil {
			httpx.WriteError(w, http.StatusInternalServerError, "internal_error", "failed to save coupon products")
			return
		}

		if err := tx.Commit(ctx); err != nil {
			httpx.WriteError(w, http.StatusInternalServerError, "internal_error", "failed to commit coupon")
			return
		}

		httpx.WriteJSON(w, http.StatusCreated, map[string]any{"id": couponID, "code": code})
	}
}

// UpdateCoupon handles PUT /coupons/:id — Owner only.
func UpdateCoupon(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")

		var req couponRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			httpx.WriteError(w, http.StatusBadRequest, "validation_error", "invalid request body")
			return
		}
		if msg, ok := req.validate(); !ok {
			httpx.WriteError(w, http.StatusBadRequest, "validation_error", msg)
			return
		}

		code := strings.ToUpper(strings.TrimSpace(req.Code))
		isActive := true
		if req.IsActive != nil {
			isActive = *req.IsActive
		}

		ctx := r.Context()
		tx, err := pool.Begin(ctx)
		if err != nil {
			httpx.WriteError(w, http.StatusInternalServerError, "internal_error", "failed to start transaction")
			return
		}
		defer tx.Rollback(ctx)

		tag, err := tx.Exec(ctx, `
			UPDATE coupons SET
				code = $1, discount_type = $2, discount_value_type = $3, discount_value = $4, min_spend = $5,
				valid_from = $6, valid_until = $7, max_total_usage = $8, max_usage_per_customer = $9, is_active = $10
			WHERE id = $11
		`, code, req.DiscountType, req.DiscountValueType, req.DiscountValue, req.MinSpend,
			req.ValidFrom, req.ValidUntil, req.MaxTotalUsage, req.MaxUsagePerCustomer, isActive, id)
		if err != nil {
			if isUniqueViolation(err) {
				httpx.WriteError(w, http.StatusConflict, "duplicate_code", "kode kupon sudah dipakai")
				return
			}
			httpx.WriteError(w, http.StatusInternalServerError, "internal_error", "failed to update coupon")
			return
		}
		if tag.RowsAffected() == 0 {
			httpx.WriteError(w, http.StatusNotFound, "not_found", "kupon tidak ditemukan")
			return
		}

		if err := replaceCouponProducts(ctx, tx, id, req.ProductIDs); err != nil {
			httpx.WriteError(w, http.StatusInternalServerError, "internal_error", "failed to save coupon products")
			return
		}

		if err := tx.Commit(ctx); err != nil {
			httpx.WriteError(w, http.StatusInternalServerError, "internal_error", "failed to commit coupon")
			return
		}

		httpx.WriteJSON(w, http.StatusOK, map[string]any{"id": id, "code": code})
	}
}

// DeleteCoupon handles DELETE /coupons/:id — Owner only. Past orders keep
// their discount_amount (a stored snapshot); orders.coupon_id just becomes
// NULL (ON DELETE SET NULL in the migration), so financial history is
// unaffected.
func DeleteCoupon(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")
		tag, err := pool.Exec(r.Context(), `DELETE FROM coupons WHERE id = $1`, id)
		if err != nil {
			httpx.WriteError(w, http.StatusInternalServerError, "internal_error", "failed to delete coupon")
			return
		}
		if tag.RowsAffected() == 0 {
			httpx.WriteError(w, http.StatusNotFound, "not_found", "kupon tidak ditemukan")
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}

func replaceCouponProducts(ctx context.Context, tx pgx.Tx, couponID string, productIDs []string) error {
	if _, err := tx.Exec(ctx, `DELETE FROM coupon_products WHERE coupon_id = $1`, couponID); err != nil {
		return err
	}
	if len(productIDs) == 0 {
		return nil
	}

	batch := &pgx.Batch{}
	for _, productID := range productIDs {
		batch.Queue(`INSERT INTO coupon_products (coupon_id, product_id) VALUES ($1, $2)`, couponID, productID)
	}
	return tx.SendBatch(ctx, batch).Close()
}

func isUniqueViolation(err error) bool {
	var pgErr interface{ SQLState() string }
	if errors.As(err, &pgErr) {
		return pgErr.SQLState() == "23505"
	}
	return false
}
