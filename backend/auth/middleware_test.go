package auth

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

type mockVerifier struct {
	sub string
	err error
}

func (m *mockVerifier) Verify(_ context.Context, _ string) (string, error) {
	return m.sub, m.err
}

func applyMiddleware(v TokenVerifier, next http.Handler) http.Handler {
	return AuthMiddleware(v)(next)
}

var okHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
})

func TestAuthMiddleware_MissingHeader(t *testing.T) {
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	applyMiddleware(&mockVerifier{sub: "user-123"}, okHandler).ServeHTTP(w, r)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("status: got %d, want %d", w.Code, http.StatusUnauthorized)
	}
}

func TestAuthMiddleware_NonBearerScheme(t *testing.T) {
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r.Header.Set("Authorization", "Basic dXNlcjpwYXNz")
	applyMiddleware(&mockVerifier{sub: "user-123"}, okHandler).ServeHTTP(w, r)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("status: got %d, want %d", w.Code, http.StatusUnauthorized)
	}
}

func TestAuthMiddleware_InvalidToken(t *testing.T) {
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r.Header.Set("Authorization", "Bearer bad-token")
	applyMiddleware(&mockVerifier{err: errors.New("token expired")}, okHandler).ServeHTTP(w, r)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("status: got %d, want %d", w.Code, http.StatusUnauthorized)
	}
}

func TestAuthMiddleware_ValidToken_PassesThrough(t *testing.T) {
	var gotSub string
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotSub, _ = SubFromContext(r.Context())
		w.WriteHeader(http.StatusOK)
	})

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r.Header.Set("Authorization", "Bearer valid-token")
	applyMiddleware(&mockVerifier{sub: "user-uuid-abc123"}, next).ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Errorf("status: got %d, want %d", w.Code, http.StatusOK)
	}
	if gotSub != "user-uuid-abc123" {
		t.Errorf("sub in context: got %q, want %q", gotSub, "user-uuid-abc123")
	}
}

func TestSubFromContext_Missing(t *testing.T) {
	_, ok := SubFromContext(context.Background())
	if ok {
		t.Error("expected ok=false for context with no authenticated user")
	}
}
