package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/siri5creative/E-comWA/api/internal/crypto"
	"github.com/siri5creative/E-comWA/api/internal/httpx"
	"github.com/siri5creative/E-comWA/api/internal/middleware"
)

var validPaymentProviders = map[string]bool{"midtrans": true, "xendit": true}

type paymentSettingsView struct {
	Configured     bool       `json:"configured"`
	Provider       *string    `json:"provider,omitempty"`
	IsSandbox      *bool      `json:"is_sandbox,omitempty"`
	IsActive       *bool      `json:"is_active,omitempty"`
	HasCredentials bool       `json:"has_credentials"`
	UpdatedAt      *time.Time `json:"updated_at,omitempty"`
}

// GetPaymentSettings handles GET /payment-settings — Owner only. Never
// returns credentials, even encrypted (PRD NFR: "tidak pernah diekspos ke
// frontend") — only whether some are currently stored.
func GetPaymentSettings(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		view, err := loadPaymentSettingsView(r.Context(), pool)
		if err != nil {
			httpx.WriteError(w, http.StatusInternalServerError, "internal_error", "failed to load payment settings")
			return
		}
		httpx.WriteJSON(w, http.StatusOK, view)
	}
}

func loadPaymentSettingsView(ctx context.Context, pool *pgxpool.Pool) (paymentSettingsView, error) {
	var provider string
	var isSandbox, isActive, hasCredentials bool
	var updatedAt time.Time

	err := pool.QueryRow(ctx, `
		SELECT provider::text, is_sandbox, is_active, encrypted_credentials IS NOT NULL, updated_at
		FROM payment_gateway_settings
		LIMIT 1
	`).Scan(&provider, &isSandbox, &isActive, &hasCredentials, &updatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return paymentSettingsView{Configured: false}, nil
	}
	if err != nil {
		return paymentSettingsView{}, err
	}

	return paymentSettingsView{
		Configured:     true,
		Provider:       &provider,
		IsSandbox:      &isSandbox,
		IsActive:       &isActive,
		HasCredentials: hasCredentials,
		UpdatedAt:      &updatedAt,
	}, nil
}

type paymentSettingsRequest struct {
	Provider    *string            `json:"provider"`
	IsSandbox   *bool              `json:"is_sandbox"`
	Credentials *map[string]string `json:"credentials"`
	IsActive    *bool              `json:"is_active"`
}

// SavePaymentSettings handles POST /payment-settings — Owner only (PRD
// section 6.8). Fields are a partial update: provider/is_sandbox/is_active
// can be changed without re-submitting credentials, so toggling active/
// inactive doesn't force re-entering secrets. There is only ever one
// settings row — this is a single settings page, not a list.
func SavePaymentSettings(pool *pgxpool.Pool, box *crypto.Box) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req paymentSettingsRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			httpx.WriteError(w, http.StatusBadRequest, "validation_error", "invalid request body")
			return
		}

		ctx := r.Context()
		admin, _ := middleware.AdminFromContext(ctx)

		var existingID, existingProvider string
		var existingSandbox, existingActive bool
		err := pool.QueryRow(ctx, `
			SELECT id, provider::text, is_sandbox, is_active FROM payment_gateway_settings LIMIT 1
		`).Scan(&existingID, &existingProvider, &existingSandbox, &existingActive)
		exists := err == nil
		if err != nil && !errors.Is(err, pgx.ErrNoRows) {
			httpx.WriteError(w, http.StatusInternalServerError, "internal_error", "failed to load payment settings")
			return
		}

		provider := existingProvider
		if req.Provider != nil {
			provider = *req.Provider
		}
		if !validPaymentProviders[provider] {
			httpx.WriteError(w, http.StatusBadRequest, "validation_error", "provider harus midtrans atau xendit")
			return
		}

		isSandbox := existingSandbox
		if req.IsSandbox != nil {
			isSandbox = *req.IsSandbox
		}

		isActive := existingActive
		if req.IsActive != nil {
			isActive = *req.IsActive
		}

		var encrypted *string
		if req.Credentials != nil {
			if box == nil {
				httpx.WriteError(w, http.StatusServiceUnavailable, "encryption_not_configured", "PAYMENT_GATEWAY_ENCRYPTION_KEY belum diset di server")
				return
			}
			credJSON, err := json.Marshal(*req.Credentials)
			if err != nil {
				httpx.WriteError(w, http.StatusInternalServerError, "internal_error", "failed to encode credentials")
				return
			}
			enc, err := box.Encrypt(string(credJSON))
			if err != nil {
				httpx.WriteError(w, http.StatusInternalServerError, "internal_error", "failed to encrypt credentials")
				return
			}
			encrypted = &enc
		} else if !exists {
			httpx.WriteError(w, http.StatusBadRequest, "validation_error", "credentials wajib diisi saat pertama kali menyimpan")
			return
		}

		if exists {
			if encrypted != nil {
				_, err = pool.Exec(ctx, `
					UPDATE payment_gateway_settings
					SET provider = $1, is_sandbox = $2, encrypted_credentials = $3, is_active = $4, updated_by = $5, updated_at = now()
					WHERE id = $6
				`, provider, isSandbox, *encrypted, isActive, admin.ID, existingID)
			} else {
				_, err = pool.Exec(ctx, `
					UPDATE payment_gateway_settings
					SET provider = $1, is_sandbox = $2, is_active = $3, updated_by = $4, updated_at = now()
					WHERE id = $5
				`, provider, isSandbox, isActive, admin.ID, existingID)
			}
		} else {
			_, err = pool.Exec(ctx, `
				INSERT INTO payment_gateway_settings (provider, is_sandbox, encrypted_credentials, is_active, updated_by)
				VALUES ($1, $2, $3, $4, $5)
			`, provider, isSandbox, *encrypted, isActive, admin.ID)
		}
		if err != nil {
			httpx.WriteError(w, http.StatusInternalServerError, "internal_error", "failed to save payment settings")
			return
		}

		view, err := loadPaymentSettingsView(ctx, pool)
		if err != nil {
			httpx.WriteError(w, http.StatusInternalServerError, "internal_error", "failed to load saved payment settings")
			return
		}
		httpx.WriteJSON(w, http.StatusOK, view)
	}
}

type testPaymentConnectionRequest struct {
	Provider    string            `json:"provider"`
	IsSandbox   bool              `json:"is_sandbox"`
	Credentials map[string]string `json:"credentials"`
}

// TestPaymentConnection handles POST /payment-settings/test — Owner only
// (PRD section 6.8: "uji coba sambungan"). Tests the credentials as typed
// in the form, before they're saved — never reads from the database.
func TestPaymentConnection() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req testPaymentConnectionRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			httpx.WriteError(w, http.StatusBadRequest, "validation_error", "invalid request body")
			return
		}
		if !validPaymentProviders[req.Provider] {
			httpx.WriteError(w, http.StatusBadRequest, "validation_error", "provider harus midtrans atau xendit")
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
		defer cancel()

		var ok bool
		var message string
		switch req.Provider {
		case "midtrans":
			ok, message = testMidtransConnection(ctx, req.IsSandbox, req.Credentials)
		case "xendit":
			ok, message = testXenditConnection(ctx, req.Credentials)
		}

		httpx.WriteJSON(w, http.StatusOK, map[string]any{"success": ok, "message": message})
	}
}

// testMidtransConnection checks a server_key by requesting the status of a
// nonexistent order: a 404 means the key authenticated successfully (the
// order just doesn't exist); 401 means the key itself was rejected. This
// avoids creating any real transaction just to validate credentials.
func testMidtransConnection(ctx context.Context, isSandbox bool, credentials map[string]string) (bool, string) {
	serverKey := credentials["server_key"]
	if serverKey == "" {
		return false, "server_key wajib diisi"
	}

	baseURL := "https://api.midtrans.com"
	if isSandbox {
		baseURL = "https://api.sandbox.midtrans.com"
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, baseURL+"/v2/00000000-0000-0000-0000-000000000000/status", nil)
	if err != nil {
		return false, "gagal membuat request"
	}
	req.SetBasicAuth(serverKey, "")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return false, "gagal terhubung ke Midtrans: " + err.Error()
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusNotFound:
		return true, "Kredensial valid (order uji coba tidak ditemukan, ini normal)"
	case http.StatusUnauthorized, http.StatusForbidden:
		return false, "server_key tidak valid"
	default:
		return false, fmt.Sprintf("Respon tidak terduga dari Midtrans: %d", resp.StatusCode)
	}
}

// testXenditConnection checks a secret_key against the balance endpoint —
// a safe, read-only call that requires valid auth to succeed.
func testXenditConnection(ctx context.Context, credentials map[string]string) (bool, string) {
	secretKey := credentials["secret_key"]
	if secretKey == "" {
		return false, "secret_key wajib diisi"
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://api.xendit.co/balance", nil)
	if err != nil {
		return false, "gagal membuat request"
	}
	req.SetBasicAuth(secretKey, "")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return false, "gagal terhubung ke Xendit: " + err.Error()
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusOK:
		return true, "Kredensial valid"
	case http.StatusUnauthorized:
		return false, "secret_key tidak valid"
	default:
		return false, fmt.Sprintf("Respon tidak terduga dari Xendit: %d", resp.StatusCode)
	}
}
