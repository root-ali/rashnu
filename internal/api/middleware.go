package api

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/golang-jwt/jwt/v5"
	"go.uber.org/zap"
)

// AuthMiddleware validates the Bearer JWT and injects role + email into the request context.
// Must be applied before RequireRole.
func (h *Handler) AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h.logger.Debug("auth middleware: validating token for request", zap.String("method", r.Method),
			zap.String("path", r.URL.Path),
			zap.Any("headers", r.Header))
		authHeader := r.Header.Get("Authorization")
		if !strings.HasPrefix(authHeader, "Bearer ") {
			writeError(w, http.StatusUnauthorized, errors.New("missing or invalid authorization header"))
			return
		}

		tokenStr := strings.TrimPrefix(authHeader, "Bearer ")
		token, err := jwt.Parse(tokenStr, func(t *jwt.Token) (interface{}, error) {
			if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, errors.New("unexpected signing method")
			}
			return h.jwtSecret, nil
		})
		if err != nil || !token.Valid {
			writeError(w, http.StatusUnauthorized, errors.New("invalid or expired token"))
			return
		}

		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			writeError(w, http.StatusUnauthorized, errors.New("invalid token claims"))
			return
		}

		role, _ := claims["role"].(string)
		email, _ := claims["email"].(string)

		if !h.identity.IsTokenValid(email, tokenStr) {
			writeError(w, http.StatusUnauthorized, errors.New("token has been revoked"))
			return
		}

		ctx := context.WithValue(r.Context(), ContextKeyRole, role)
		ctx = context.WithValue(ctx, ContextKeyEmail, email)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// RequireRole returns a middleware that allows only requests whose token role
// matches one of the given roles. AuthMiddleware must run first.
func (h *Handler) RequireRole(roles ...string) func(http.Handler) http.Handler {
	allowed := make(map[string]struct{}, len(roles))
	for _, r := range roles {
		allowed[r] = struct{}{}
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			role, _ := r.Context().Value(ContextKeyRole).(string)
			if _, ok := allowed[role]; !ok {
				writeError(w, http.StatusForbidden, errors.New("insufficient permissions"))
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
