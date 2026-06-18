package auth

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/coreos/go-oidc/v3/oidc"
)

type contextKey string

const contextKeySub contextKey = "sub"

// TokenVerifier validates a raw Bearer token and returns the subject claim.
// *OIDCVerifier implements this; tests use a mock.
type TokenVerifier interface {
	Verify(ctx context.Context, rawToken string) (sub string, err error)
}

// OIDCVerifier wraps *oidc.IDTokenVerifier to implement TokenVerifier.
type OIDCVerifier struct {
	inner *oidc.IDTokenVerifier
}

func NewOIDCVerifier(inner *oidc.IDTokenVerifier) *OIDCVerifier {
	return &OIDCVerifier{inner: inner}
}

func (v *OIDCVerifier) Verify(ctx context.Context, rawToken string) (string, error) {
	token, err := v.inner.Verify(ctx, rawToken)
	if err != nil {
		return "", err
	}
	return token.Subject, nil
}

// AuthMiddleware is a chi-compatible middleware that requires a valid Keycloak Bearer token.
// On success it injects the token's subject claim into the request context via SubFromContext.
func AuthMiddleware(v TokenVerifier) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			header := r.Header.Get("Authorization")
			if !strings.HasPrefix(header, "Bearer ") {
				writeError(w, http.StatusUnauthorized, "authorization header required")
				return
			}

			sub, err := v.Verify(r.Context(), strings.TrimPrefix(header, "Bearer "))
			if err != nil {
				writeError(w, http.StatusUnauthorized, "invalid or expired token")
				return
			}

			next.ServeHTTP(w, r.WithContext(SubToContext(r.Context(), sub)))
		})
	}
}

// SubFromContext retrieves the authenticated user's Keycloak subject (UUID) from the request context.
// Returns ("", false) if no authenticated user is present.
func SubFromContext(ctx context.Context) (string, bool) {
	sub, ok := ctx.Value(contextKeySub).(string)
	return sub, ok
}

// SubToContext injects a Keycloak subject claim into a context.
// Used by AuthMiddleware and in tests to simulate an authenticated user.
func SubToContext(ctx context.Context, sub string) context.Context {
	return context.WithValue(ctx, contextKeySub, sub)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]string{"error": msg})
}
