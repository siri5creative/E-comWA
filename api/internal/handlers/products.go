package handlers

import (
	"context"
	"net/http"
	"strconv"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/siri5creative/E-comWA/api/internal/httpx"
	"github.com/siri5creative/E-comWA/api/internal/models"
)

const (
	defaultPageSize = 20
	maxPageSize     = 100
)

type paginationMeta struct {
	Page     int   `json:"page"`
	PageSize int   `json:"page_size"`
	Total    int64 `json:"total"`
}

type listProductsResponse struct {
	Data       []models.Product `json:"data"`
	Pagination paginationMeta   `json:"pagination"`
}

// ListProducts handles GET /products — public catalog listing with an
// optional ?category=<slug> filter and page/page_size pagination
// (PRD section 6.1, NFR section 9).
func ListProducts(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		page := parsePositiveInt(r.URL.Query().Get("page"), 1)
		pageSize := parsePositiveInt(r.URL.Query().Get("page_size"), defaultPageSize)
		if pageSize > maxPageSize {
			pageSize = maxPageSize
		}
		offset := (page - 1) * pageSize

		var categorySlug *string
		if v := r.URL.Query().Get("category"); v != "" {
			categorySlug = &v
		}

		ctx := r.Context()
		rows, err := pool.Query(ctx, `
			SELECT p.id, p.name, p.slug, p.description, p.cover_image_url,
			       c.id, c.name, c.slug,
			       count(*) OVER() AS total_count
			FROM products p
			LEFT JOIN categories c ON c.id = p.category_id
			WHERE $1::text IS NULL OR c.slug = $1
			ORDER BY p.created_at DESC
			LIMIT $2 OFFSET $3
		`, categorySlug, pageSize, offset)
		if err != nil {
			httpx.WriteError(w, http.StatusInternalServerError, "internal_error", "failed to list products")
			return
		}
		defer rows.Close()

		products := []models.Product{}
		var total int64

		for rows.Next() {
			var p models.Product
			var catID, catName, catSlug *string

			if err := rows.Scan(&p.ID, &p.Name, &p.Slug, &p.Description, &p.CoverImageURL,
				&catID, &catName, &catSlug, &total); err != nil {
				httpx.WriteError(w, http.StatusInternalServerError, "internal_error", "failed to read products")
				return
			}
			if catID != nil {
				p.Category = &models.Category{ID: *catID, Name: *catName, Slug: *catSlug}
			}
			p.Variants = []models.ProductVariant{}

			products = append(products, p)
		}
		if err := rows.Err(); err != nil {
			httpx.WriteError(w, http.StatusInternalServerError, "internal_error", "failed to read products")
			return
		}

		if err := attachVariants(ctx, pool, products); err != nil {
			httpx.WriteError(w, http.StatusInternalServerError, "internal_error", "failed to load product variants")
			return
		}

		httpx.WriteJSON(w, http.StatusOK, listProductsResponse{
			Data: products,
			Pagination: paginationMeta{
				Page:     page,
				PageSize: pageSize,
				Total:    total,
			},
		})
	}
}

// GetProductBySlug handles GET /products/:slug — public product detail.
func GetProductBySlug(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		slug := r.PathValue("slug")
		ctx := r.Context()

		var p models.Product
		var catID, catName, catSlug *string

		err := pool.QueryRow(ctx, `
			SELECT p.id, p.name, p.slug, p.description, p.cover_image_url,
			       c.id, c.name, c.slug
			FROM products p
			LEFT JOIN categories c ON c.id = p.category_id
			WHERE p.slug = $1
		`, slug).Scan(&p.ID, &p.Name, &p.Slug, &p.Description, &p.CoverImageURL, &catID, &catName, &catSlug)
		if err != nil {
			if err == pgx.ErrNoRows {
				httpx.WriteError(w, http.StatusNotFound, "not_found", "produk tidak ditemukan")
				return
			}
			httpx.WriteError(w, http.StatusInternalServerError, "internal_error", "failed to load product")
			return
		}
		if catID != nil {
			p.Category = &models.Category{ID: *catID, Name: *catName, Slug: *catSlug}
		}
		p.Variants = []models.ProductVariant{}

		products := []models.Product{p}
		if err := attachVariants(ctx, pool, products); err != nil {
			httpx.WriteError(w, http.StatusInternalServerError, "internal_error", "failed to load product variants")
			return
		}

		httpx.WriteJSON(w, http.StatusOK, products[0])
	}
}

// attachVariants batch-loads variants for a page of products (avoiding N+1
// queries) and assigns them back onto each product in place.
func attachVariants(ctx context.Context, pool *pgxpool.Pool, products []models.Product) error {
	if len(products) == 0 {
		return nil
	}

	ids := make([]string, len(products))
	for i, p := range products {
		ids[i] = p.ID
	}

	variantsByProduct, err := fetchVariantsForProducts(ctx, pool, ids)
	if err != nil {
		return err
	}

	for i := range products {
		if v, ok := variantsByProduct[products[i].ID]; ok {
			products[i].Variants = v
		}
	}
	return nil
}

func fetchVariantsForProducts(ctx context.Context, pool *pgxpool.Pool, productIDs []string) (map[string][]models.ProductVariant, error) {
	rows, err := pool.Query(ctx, `
		SELECT product_id, id, variant_name, sku, price, stock_quantity
		FROM product_variants
		WHERE product_id = ANY($1)
		ORDER BY variant_name
	`, productIDs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make(map[string][]models.ProductVariant)
	for rows.Next() {
		var productID string
		var v models.ProductVariant
		if err := rows.Scan(&productID, &v.ID, &v.VariantName, &v.SKU, &v.Price, &v.StockQuantity); err != nil {
			return nil, err
		}
		result[productID] = append(result[productID], v)
	}
	return result, rows.Err()
}

func parsePositiveInt(raw string, fallback int) int {
	if raw == "" {
		return fallback
	}
	n, err := strconv.Atoi(raw)
	if err != nil || n < 1 {
		return fallback
	}
	return n
}
