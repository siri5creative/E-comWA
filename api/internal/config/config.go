// Package config loads the environment variables listed in
// IMPLEMENTATION.md section 2 (api/.env) into a single struct.
package config

import (
	"fmt"
	"os"
	"strings"
)

type Config struct {
	// DatabaseURL is the Supavisor pooler connection string from Supabase.
	DatabaseURL string
	// SupabaseURL is the project's base URL (https://<ref>.supabase.co).
	// Not in IMPLEMENTATION.md's documented api/.env list — added because
	// admin JWT verification needs it: Supabase projects created with the
	// newer asymmetric "JWT Signing Keys" sign Auth tokens with ES256, not
	// the legacy HS256 shared secret, and verifying ES256 requires fetching
	// the project's public keys from
	// <SUPABASE_URL>/auth/v1/.well-known/jwks.json.
	SupabaseURL string
	// SupabaseJWTSecret verifies admin (Owner/Staff) Supabase Auth tokens
	// signed with the legacy HS256 shared secret. Projects using the newer
	// asymmetric signing keys (see SupabaseURL above) don't need this —
	// at least one of SupabaseJWTSecret or SupabaseURL must be set.
	SupabaseJWTSecret string
	// SupabaseServiceRoleKey is used for operations that must bypass RLS
	// via the Supabase HTTP APIs (e.g. Auth Admin API when creating admin
	// accounts) — not used by the direct Postgres connection above.
	SupabaseServiceRoleKey string
	// FirebaseServiceAccountKey is the Firebase Admin SDK JSON credential,
	// used to send push notifications to admins.
	FirebaseServiceAccountKey string
	// PaymentGatewayEncryptionKey encrypts Midtrans/Xendit credentials
	// stored by an Owner in payment_gateway_settings.
	PaymentGatewayEncryptionKey string
	// POSAPIKey authenticates requests from the POS application to /pos/*.
	POSAPIKey string
	// Port the HTTP server listens on.
	Port string
	// CORSAllowedOrigins restricts which browser origins may call public
	// endpoints directly (products, checkout). Not part of the documented
	// env var list, but required for the frontend to call this API
	// cross-origin; defaults to "*" (open) if unset.
	CORSAllowedOrigins []string
}

// Load reads configuration from the process environment. It does not read
// .env files itself — call godotenv.Load() (or similar) before Load() in
// local development.
func Load() (Config, error) {
	cfg := Config{
		DatabaseURL:                 os.Getenv("DATABASE_URL"),
		SupabaseURL:                 os.Getenv("SUPABASE_URL"),
		SupabaseJWTSecret:           os.Getenv("SUPABASE_JWT_SECRET"),
		SupabaseServiceRoleKey:      os.Getenv("SUPABASE_SERVICE_ROLE_KEY"),
		FirebaseServiceAccountKey:   os.Getenv("FIREBASE_SERVICE_ACCOUNT_KEY"),
		PaymentGatewayEncryptionKey: os.Getenv("PAYMENT_GATEWAY_ENCRYPTION_KEY"),
		POSAPIKey:                   os.Getenv("POS_API_KEY"),
		Port:                        os.Getenv("PORT"),
	}

	if cfg.Port == "" {
		cfg.Port = "3000"
	}

	origins := os.Getenv("CORS_ALLOWED_ORIGINS")
	if origins == "" {
		cfg.CORSAllowedOrigins = []string{"*"}
	} else {
		for _, part := range strings.Split(origins, ",") {
			if trimmed := strings.TrimSpace(part); trimmed != "" {
				cfg.CORSAllowedOrigins = append(cfg.CORSAllowedOrigins, trimmed)
			}
		}
	}

	// Only the config needed by today's endpoints (products, checkout,
	// admin auth) is required to start the server. Vars for later phases
	// (Firebase, payment gateway, POS) are validated when those handlers
	// are wired up.
	if cfg.DatabaseURL == "" {
		return cfg, fmt.Errorf("DATABASE_URL is required")
	}
	if cfg.SupabaseJWTSecret == "" && cfg.SupabaseURL == "" {
		return cfg, fmt.Errorf("at least one of SUPABASE_JWT_SECRET or SUPABASE_URL is required (to verify admin JWTs)")
	}

	return cfg, nil
}
