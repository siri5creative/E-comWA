package middleware

import (
	"crypto/subtle"
	"net/http"

	"github.com/siri5creative/E-comWA/api/internal/httpx"
)

// RequirePOSAPIKey validates the static POS_API_KEY (api-pos-integration.md
// section 4) — a shared secret the POS app holds, not per-cashier auth
// (that stays internal to the POS app). Uses constant-time comparison so
// response timing can't leak the key.
//
// If apiKey is empty (POS_API_KEY not configured), every request is
// rejected regardless of what token is sent — this must never be treated
// as "any key is valid".
func RequirePOSAPIKey(apiKey string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			token, ok := bearerToken(r)
			if !ok {
				httpx.WriteError(w, http.StatusUnauthorized, "unauthorized", "missing bearer token")
				return
			}
			if apiKey == "" || subtle.ConstantTimeCompare([]byte(token), []byte(apiKey)) != 1 {
				httpx.WriteError(w, http.StatusUnauthorized, "unauthorized", "invalid POS API key")
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
