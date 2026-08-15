package handlers

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/siri5creative/E-comWA/api/internal/httpx"
	"github.com/siri5creative/E-comWA/api/internal/middleware"
)

type registerAdminDeviceRequest struct {
	FCMDeviceToken string `json:"fcm_device_token"`
}

// RegisterAdminDevice handles POST /admin-devices — admin only, any role
// (PRD section 6.6: browser asks for notification permission on first
// login, regardless of Owner/Staff). Re-registering a token that already
// belongs to a different admin reassigns it — the same browser/device may
// be used by more than one admin account over time.
func RegisterAdminDevice(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req registerAdminDeviceRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			httpx.WriteError(w, http.StatusBadRequest, "validation_error", "invalid request body")
			return
		}
		if strings.TrimSpace(req.FCMDeviceToken) == "" {
			httpx.WriteError(w, http.StatusBadRequest, "validation_error", "fcm_device_token wajib diisi")
			return
		}

		admin, _ := middleware.AdminFromContext(r.Context())

		_, err := pool.Exec(r.Context(), `
			INSERT INTO admin_devices (admin_id, fcm_device_token)
			VALUES ($1, $2)
			ON CONFLICT (fcm_device_token) DO UPDATE SET admin_id = EXCLUDED.admin_id
		`, admin.ID, req.FCMDeviceToken)
		if err != nil {
			httpx.WriteError(w, http.StatusInternalServerError, "internal_error", "failed to save device token")
			return
		}

		w.WriteHeader(http.StatusNoContent)
	}
}
