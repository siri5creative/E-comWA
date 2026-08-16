package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/siri5creative/E-comWA/api/internal/httpx"
)

type productRequest struct {
	Name          string  `json:"name"`
	Slug          string  `json:"slug"`
	Description   *string `json:"description"`
	CategoryID    *string `json:"category_id"`
	CoverImageURL *string `json:"cover_image_url"`
}

func (req *productRequest) validate() (string, bool) {
	if strings.TrimSpace(req.Name) == "" {
		return "name wajib diisi", false
	}
	if strings.TrimSpace(req.Slug) == "" {
		return "slug wajib diisi", false
	}
	return "", true
}

// CreateProduct handles POST /products — admin, any role (PRD section 6.7:
// product/stock management isn't Owner-only; section 8: "Admin").
func CreateProduct(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req productRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			httpx.WriteError(w, http.StatusBadRequest, "validation_error", "invalid request body")
			return
		}
		if msg, ok := req.validate(); !ok {
			httpx.WriteError(w, http.StatusBadRequest, "validation_error", msg)
			return
		}

		var id string
		err := pool.QueryRow(r.Context(), `
			INSERT INTO products (name, slug, description, category_id, cover_image_url)
			VALUES ($1, $2, $3, $4, $5)
			RETURNING id
		`, req.Name, req.Slug, req.Description, req.CategoryID, req.CoverImageURL).Scan(&id)
		if err != nil {
			if isUniqueViolation(err) {
				httpx.WriteError(w, http.StatusConflict, "duplicate_slug", "slug produk sudah dipakai")
				return
			}
			httpx.WriteError(w, http.StatusInternalServerError, "internal_error", "failed to create product")
			return
		}

		httpx.WriteJSON(w, http.StatusCreated, map[string]any{"id": id})
	}
}

// UpdateProduct handles PUT /products/:id — admin, any role.
func UpdateProduct(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")

		var req productRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			httpx.WriteError(w, http.StatusBadRequest, "validation_error", "invalid request body")
			return
		}
		if msg, ok := req.validate(); !ok {
			httpx.WriteError(w, http.StatusBadRequest, "validation_error", msg)
			return
		}

		tag, err := pool.Exec(r.Context(), `
			UPDATE products
			SET name = $1, slug = $2, description = $3, category_id = $4, cover_image_url = $5, updated_at = now()
			WHERE id = $6
		`, req.Name, req.Slug, req.Description, req.CategoryID, req.CoverImageURL, id)
		if err != nil {
			if isUniqueViolation(err) {
				httpx.WriteError(w, http.StatusConflict, "duplicate_slug", "slug produk sudah dipakai")
				return
			}
			httpx.WriteError(w, http.StatusInternalServerError, "internal_error", "failed to update product")
			return
		}
		if tag.RowsAffected() == 0 {
			httpx.WriteError(w, http.StatusNotFound, "not_found", "produk tidak ditemukan")
			return
		}

		httpx.WriteJSON(w, http.StatusOK, map[string]any{"id": id})
	}
}

// DeleteProduct handles DELETE /products/:id — admin, any role. Variants
// cascade-delete, but a variant with order history is protected
// (order_items.product_variant_id is ON DELETE RESTRICT) — deleting such a
// product fails with a clear error instead of a raw 500.
func DeleteProduct(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")

		tag, err := pool.Exec(r.Context(), `DELETE FROM products WHERE id = $1`, id)
		if err != nil {
			if isForeignKeyViolation(err) {
				httpx.WriteError(w, http.StatusConflict, "has_order_history",
					"produk ini tidak bisa dihapus karena variannya sudah punya riwayat order")
				return
			}
			httpx.WriteError(w, http.StatusInternalServerError, "internal_error", "failed to delete product")
			return
		}
		if tag.RowsAffected() == 0 {
			httpx.WriteError(w, http.StatusNotFound, "not_found", "produk tidak ditemukan")
			return
		}

		w.WriteHeader(http.StatusNoContent)
	}
}

type variantRequest struct {
	VariantName   string  `json:"variant_name"`
	SKU           *string `json:"sku"`
	Price         int64   `json:"price"`
	StockQuantity int32   `json:"stock_quantity"`
}

func (req *variantRequest) validate() (string, bool) {
	if strings.TrimSpace(req.VariantName) == "" {
		return "variant_name wajib diisi", false
	}
	if req.Price < 0 {
		return "price tidak boleh negatif", false
	}
	if req.StockQuantity < 0 {
		return "stock_quantity tidak boleh negatif", false
	}
	return "", true
}

// CreateVariant handles POST /products/:id/variants — admin, any role.
func CreateVariant(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		productID := r.PathValue("id")

		var req variantRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			httpx.WriteError(w, http.StatusBadRequest, "validation_error", "invalid request body")
			return
		}
		if msg, ok := req.validate(); !ok {
			httpx.WriteError(w, http.StatusBadRequest, "validation_error", msg)
			return
		}

		var id string
		err := pool.QueryRow(r.Context(), `
			INSERT INTO product_variants (product_id, variant_name, sku, price, stock_quantity)
			VALUES ($1, $2, $3, $4, $5)
			RETURNING id
		`, productID, req.VariantName, req.SKU, req.Price, req.StockQuantity).Scan(&id)
		if err != nil {
			if isUniqueViolation(err) {
				httpx.WriteError(w, http.StatusConflict, "duplicate_sku", "SKU sudah dipakai")
				return
			}
			if isForeignKeyViolation(err) {
				httpx.WriteError(w, http.StatusNotFound, "not_found", "produk tidak ditemukan")
				return
			}
			httpx.WriteError(w, http.StatusInternalServerError, "internal_error", "failed to create variant")
			return
		}

		httpx.WriteJSON(w, http.StatusCreated, map[string]any{"id": id})
	}
}

// UpdateVariant handles PUT /products/variants/:variant_id — admin, any
// role. stock_quantity is a plain replace (the admin enters the counted
// total after restocking), not a delta — simple and sufficient for MVP
// scope; concurrent customer orders still decrement atomically regardless
// (see internal/handlers/stock.go).
func UpdateVariant(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		variantID := r.PathValue("variant_id")

		var req variantRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			httpx.WriteError(w, http.StatusBadRequest, "validation_error", "invalid request body")
			return
		}
		if msg, ok := req.validate(); !ok {
			httpx.WriteError(w, http.StatusBadRequest, "validation_error", msg)
			return
		}

		tag, err := pool.Exec(r.Context(), `
			UPDATE product_variants
			SET variant_name = $1, sku = $2, price = $3, stock_quantity = $4, updated_at = now()
			WHERE id = $5
		`, req.VariantName, req.SKU, req.Price, req.StockQuantity, variantID)
		if err != nil {
			if isUniqueViolation(err) {
				httpx.WriteError(w, http.StatusConflict, "duplicate_sku", "SKU sudah dipakai")
				return
			}
			httpx.WriteError(w, http.StatusInternalServerError, "internal_error", "failed to update variant")
			return
		}
		if tag.RowsAffected() == 0 {
			httpx.WriteError(w, http.StatusNotFound, "not_found", "varian tidak ditemukan")
			return
		}

		httpx.WriteJSON(w, http.StatusOK, map[string]any{"id": variantID})
	}
}

// DeleteVariant handles DELETE /products/variants/:variant_id — admin, any
// role. Protected the same way as DeleteProduct: a variant with order
// history can't be deleted (ON DELETE RESTRICT).
func DeleteVariant(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		variantID := r.PathValue("variant_id")

		tag, err := pool.Exec(r.Context(), `DELETE FROM product_variants WHERE id = $1`, variantID)
		if err != nil {
			if isForeignKeyViolation(err) {
				httpx.WriteError(w, http.StatusConflict, "has_order_history",
					"varian ini tidak bisa dihapus karena sudah punya riwayat order")
				return
			}
			httpx.WriteError(w, http.StatusInternalServerError, "internal_error", "failed to delete variant")
			return
		}
		if tag.RowsAffected() == 0 {
			httpx.WriteError(w, http.StatusNotFound, "not_found", "varian tidak ditemukan")
			return
		}

		w.WriteHeader(http.StatusNoContent)
	}
}

// isForeignKeyViolation reports whether err is a Postgres foreign-key
// error: either a normal FK violation (23503, e.g. a variant referencing a
// nonexistent product) or a RESTRICT-specific violation (23001, raised when
// deleting a row that another row still references with ON DELETE RESTRICT
// — the case order_items → product_variants hits here).
func isForeignKeyViolation(err error) bool {
	var pgErr interface{ SQLState() string }
	if errors.As(err, &pgErr) {
		state := pgErr.SQLState()
		return state == "23503" || state == "23001"
	}
	return false
}
