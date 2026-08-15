package middleware

import (
	"context"
	"errors"
	"net/http"
	"strings"

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
type AdminAuth struct {
	Pool      *pgxpool.Pool
	JWTSecret string
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

func (a *AdminAuth) authenticate(w http.ResponseWriter, r *http.Request) (*models.Admin, bool) {
	token, ok := bearerToken(r)
	if !ok {
		httpx.WriteError(w, http.StatusUnauthorized, "unauthorized", "missing bearer token")
		return nil, false
	}

	claims := &supabaseClaims{}
	parsed, err := jwt.ParseWithClaims(token, claims, func(t *jwt.Token) (any, error) {
		return []byte(a.JWTSecret), nil
	}, jwt.WithValidMethods([]string{"HS256"}), jwt.WithAudience("authenticated"))
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
