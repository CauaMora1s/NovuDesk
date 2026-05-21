package middleware

import (
	"context"
	"net/http"
	"strings"

	authsvc "github.com/novudesk/novudesk/internal/application/auth"
	"github.com/novudesk/novudesk/internal/interfaces/http/respond"
)

type contextKey string

const (
	claimsKey contextKey = "claims"
)

// Authenticate validates the JWT access token and stores claims in context.
func Authenticate(svc *authsvc.Service) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			token := extractBearerToken(r)
			if token == "" {
				respond.Unauthorized(w, "missing authorization token")
				return
			}

			claims, err := svc.ValidateAccessToken(token)
			if err != nil {
				respond.Unauthorized(w, "invalid or expired token")
				return
			}

			ctx := context.WithValue(r.Context(), claimsKey, claims)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// RequirePermission checks that the authenticated user holds the given permission key.
func RequirePermission(perm string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			claims := ClaimsFromContext(r.Context())
			if claims == nil {
				respond.Unauthorized(w, "not authenticated")
				return
			}

			for _, p := range claims.Permissions {
				if p == perm {
					next.ServeHTTP(w, r)
					return
				}
			}

			respond.Forbidden(w, "insufficient permissions")
		})
	}
}

// ClaimsFromContext retrieves JWT claims stored by Authenticate middleware.
func ClaimsFromContext(ctx context.Context) *authsvc.Claims {
	claims, _ := ctx.Value(claimsKey).(*authsvc.Claims)
	return claims
}

// WithClaims returns a copy of ctx with the given claims stored under the
// same key used by the Authenticate middleware. Intended for use in tests.
func WithClaims(ctx context.Context, claims *authsvc.Claims) context.Context {
	return context.WithValue(ctx, claimsKey, claims)
}

// TestInjectClaims is an alias for WithClaims with an explicit name that
// signals test-only use at the call site.
var TestInjectClaims = WithClaims

func extractBearerToken(r *http.Request) string {
	header := r.Header.Get("Authorization")
	if !strings.HasPrefix(header, "Bearer ") {
		return ""
	}
	return strings.TrimPrefix(header, "Bearer ")
}
