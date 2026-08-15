package handlers

import (
	"net/http"

	"github.com/siri5creative/E-comWA/api/internal/httpx"
	"github.com/siri5creative/E-comWA/api/internal/middleware"
)

// Me handles GET /admin/me — returns the caller's own admin record. Used by
// the frontend after Supabase Auth login to know which role (Owner/Staff)
// to render, without granting it direct read access to the `admins` table
// (see the RLS notes in migrations/0001_init.sql).
func Me() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		admin, ok := middleware.AdminFromContext(r.Context())
		if !ok {
			httpx.WriteError(w, http.StatusInternalServerError, "internal_error", "admin missing from context")
			return
		}
		httpx.WriteJSON(w, http.StatusOK, admin)
	}
}
