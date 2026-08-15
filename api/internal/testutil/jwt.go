package testutil

import (
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// MintAdminJWT builds a Supabase-Auth-shaped HS256 JWT for authUserID,
// signed with secret — the same shape middleware.AdminAuth verifies. Used
// to exercise the real RequireAdmin/RequireOwner middleware chain in
// tests, rather than only testing handlers in isolation.
func MintAdminJWT(t *testing.T, authUserID, secret string) string {
	t.Helper()
	claims := jwt.RegisteredClaims{
		Subject:   authUserID,
		Audience:  jwt.ClaimStrings{"authenticated"},
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString([]byte(secret))
	if err != nil {
		t.Fatalf("failed to mint test JWT: %v", err)
	}
	return signed
}
