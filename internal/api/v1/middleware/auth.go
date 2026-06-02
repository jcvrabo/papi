package middleware

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/coreos/go-oidc/v3/oidc"
)

type contextKey string

const userContextKey contextKey = "user"

type Claims struct {
	Sub              string `json:"sub"`
	Email            string `json:"email"`
	PreferredUsername string `json:"preferred_username"`
}

func OIDCAuth(verifier *oidc.IDTokenVerifier) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
				writeError(w, http.StatusUnauthorized, "authentication_required", "Bearer token required")
				return
			}
			token := strings.TrimPrefix(authHeader, "Bearer ")
			idToken, err := verifier.Verify(r.Context(), token)
			if err != nil {
				writeError(w, http.StatusUnauthorized, "invalid_token", "Token validation failed")
				return
			}
			var claims Claims
			if err := idToken.Claims(&claims); err != nil {
				writeError(w, http.StatusUnauthorized, "invalid_claims", "Failed to parse token claims")
				return
			}
			username := claims.PreferredUsername
			if username == "" {
				username = claims.Email
			}
			if username == "" {
				username = claims.Sub
			}
			ctx := context.WithValue(r.Context(), userContextKey, username)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func UserFromContext(ctx context.Context) string {
	user, _ := ctx.Value(userContextKey).(string)
	return user
}

func SetUserInContext(ctx context.Context, user string) context.Context {
	return context.WithValue(ctx, userContextKey, user)
}

func writeError(w http.ResponseWriter, status int, code, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]string{"error": code, "message": message})
}
