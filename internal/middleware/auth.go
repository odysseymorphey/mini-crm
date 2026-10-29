package middleware

import (
	"context"
	"net/http"
	"strings"

	"mini-crm/internal/domain"
	"mini-crm/internal/service"
)

type contextKey string

const (
	UserIDKey    contextKey = "user_id"
	UserEmailKey contextKey = "user_email"
)

func Auth(authService *service.AuthService) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				http.Error(w, `{"error":"missing authorization header"}`, http.StatusUnauthorized)
				return
			}

			parts := strings.SplitN(authHeader, " ", 2)
			if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
				http.Error(w, `{"error":"invalid authorization format, expected 'Bearer <token>'"}`, http.StatusUnauthorized)
				return
			}

			claims, err := authService.ValidateToken(parts[1])
			if err != nil {
				http.Error(w, `{"error":"invalid or expired token"}`, http.StatusUnauthorized)
				return
			}

			ctx := context.WithValue(r.Context(), UserIDKey, claims.UserID)
			ctx = context.WithValue(ctx, UserEmailKey, claims.Email)

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func GetUserID(ctx context.Context) (int64, error) {
	val, ok := ctx.Value(UserIDKey).(int64)
	if !ok || val <= 0 {
		return 0, domain.ErrUnauthorized
	}
	return val, nil
}

func GetUserEmail(ctx context.Context) (string, error) {
	val, ok := ctx.Value(UserEmailKey).(string)
	if !ok || val == "" {
		return "", domain.ErrUnauthorized
	}
	return val, nil
}
