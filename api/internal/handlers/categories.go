package handlers

import (
	"net/http"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/siri5creative/E-comWA/api/internal/httpx"
	"github.com/siri5creative/E-comWA/api/internal/models"
)

// ListCategories handles GET /categories — public, used to build the
// category filter on the product catalog (PRD section 6.1).
func ListCategories(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		rows, err := pool.Query(r.Context(), `SELECT id, name, slug FROM categories ORDER BY name`)
		if err != nil {
			httpx.WriteError(w, http.StatusInternalServerError, "internal_error", "failed to list categories")
			return
		}
		defer rows.Close()

		categories := []models.Category{}
		for rows.Next() {
			var c models.Category
			if err := rows.Scan(&c.ID, &c.Name, &c.Slug); err != nil {
				httpx.WriteError(w, http.StatusInternalServerError, "internal_error", "failed to read categories")
				return
			}
			categories = append(categories, c)
		}
		if err := rows.Err(); err != nil {
			httpx.WriteError(w, http.StatusInternalServerError, "internal_error", "failed to read categories")
			return
		}

		httpx.WriteJSON(w, http.StatusOK, map[string]any{"data": categories})
	}
}
