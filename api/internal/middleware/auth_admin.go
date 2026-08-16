package middleware

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/MicahParks/keyfunc/v3"
	"github.com/golang-jwt/jwt/v5"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/siri5creative/E-comWA/api/internal/httpx"
	"github.com/siri5creative/E-comWA/api/internal/models"
)

type contextKey string

const adminContextKey contextKey = "admin"

// supabaseClaims are the standard claims on a Supabase Auth JWT. Only
// RegisteredClaims.Subject (the auth.users.id) is used to look up the
// matching row in `admins`.
type supabaseClaims struct {
	jwt.RegisteredClaims
}

// AdminAuth verifies Supabase Auth tokens and checks the caller against the
// `admins` table (Owner/Staff), per PRD section 8: "role dicek di setiap
// endpoint yang sensitif — bukan hanya dicek di frontend."
//
// Supabase projects sign Auth tokens one of two ways: the legacy HS256
// shared secret (JWTSecret), or — for projects created with the newer
// asymmetric "JWT Signing Keys" — ES256, verified against the project's
// published JWKS (JWKS). Which one a given token uses is read from its
// header, so both can be configured at once for compatibility.
type AdminAuth struct {
	Pool      *pgxpool.Pool
	JWTSecret string
	JWKS      keyfunc.Keyfunc
}

// RequireAdmin accepts any authenticated admin, regardless of role.
func (a *AdminAuth) RequireAdmin(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		admin, ok := a.authenticate(w, r)
		if !ok {
			return
		}
		ctx := context.WithValue(r.Context(), adminContextKey, *admin)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// RequireOwner accepts only admins with role "owner" (e.g. coupons, admin
// account management, payment settings, financial reports — PRD 6.7).
func (a *AdminAuth) RequireOwner(next http.Handler) http.Handler {
	return a.RequireAdmin(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		admin, _ := AdminFromContext(r.Context())
		if admin.Role != models.AdminRoleOwner {
			httpx.WriteError(w, http.StatusForbidden, "forbidden", "requires owner role")
			return
		}
		next.ServeHTTP(w, r)
	}))
}

// verificationKey picks the right key material for the token's own
// signing algorithm, so a single AdminAuth works regardless of which
// signing mode the Supabase project uses.
func (a *AdminAuth) verificationKey(t *jwt.Token) (any, error) {
	switch t.Method.Alg() {
	case "HS256":
		if a.JWTSecret == "" {
			return nil, fmt.Errorf("token is HS256 but SUPABASE_JWT_SECRET is not configured")
		}
		return []byte(a.JWTSecret), nil
	case "ES256":
		if a.JWKS == nil {
			return nil, fmt.Errorf("token is ES256 but SUPABASE_URL (JWKS) is not configured")
		}
		return a.JWKS.Keyfunc(t)
	default:
		return nil, fmt.Errorf("unsupported signing algorithm %q", t.Method.Alg())
	}
}

func (a *AdminAuth) authenticate(w http.ResponseWriter, r *http.Request) (*models.Admin, bool) {
	token, ok := bearerToken(r)
	if !ok {
		httpx.WriteError(w, http.StatusUnauthorized, "unauthorized", "missing bearer token")
		return nil, false
	}

	claims := &supabaseClaims{}
	parsed, err := jwt.ParseWithClaims(token, claims, a.verificationKey,
		jwt.WithValidMethods([]string{"HS256", "ES256"}), jwt.WithAudience("authenticated"))
	if err != nil || !parsed.Valid || claims.Subject == "" {
		httpx.WriteError(w, http.StatusUnauthorized, "unauthorized", "invalid or expired token")
		return nil, false
	}

	var admin models.Admin
	err = a.Pool.QueryRow(r.Context(),
		`SELECT id, name, role, created_at FROM admins WHERE auth_user_id = $1`,
		claims.Subject,
	).Scan(&admin.ID, &admin.Name, &admin.Role, &admin.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			httpx.WriteError(w, http.StatusForbidden, "forbidden", "not an admin account")
			return nil, false
		}
		httpx.WriteError(w, http.StatusInternalServerError, "internal_error", "failed to verify admin")
		return nil, false
	}

	return &admin, true
}

func bearerToken(r *http.Request) (string, bool) {
	header := r.Header.Get("Authorization")
	const prefix = "Bearer "
	if !strings.HasPrefix(header, prefix) {
		return "", false
	}
	token := strings.TrimSpace(strings.TrimPrefix(header, prefix))
	return token, token != ""
}

// AdminFromContext retrieves the admin attached by RequireAdmin/RequireOwner.
func AdminFromContext(ctx context.Context) (models.Admin, bool) {
	admin, ok := ctx.Value(adminContextKey).(models.Admin)
	return admin, ok
}
