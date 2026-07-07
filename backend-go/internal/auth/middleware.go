package auth

import (
	"context"
	"net/http"

	"github.com/matchlock/backend-go/internal/db"
)

type contextKey string

const userContextKey contextKey = "authUser"

// UserFromContext returns the authenticated user injected by middleware.
func UserFromContext(ctx context.Context) (db.User, bool) {
	user, ok := ctx.Value(userContextKey).(db.User)
	return user, ok
}

// Middleware validates access cookies and injects the user into request context.
func Middleware(svc *Service) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			access, ok := ReadCookie(r, AccessCookieName)
			if !ok {
				next.ServeHTTP(w, r)
				return
			}
			user, err := svc.UserFromAccessToken(r.Context(), access)
			if err != nil {
				next.ServeHTTP(w, r)
				return
			}
			ctx := context.WithValue(r.Context(), userContextKey, user)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// RequireAuth rejects requests without a valid session user.
func RequireAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if _, ok := UserFromContext(r.Context()); !ok {
			writeAuthError(w, http.StatusUnauthorized, "UNAUTHORIZED", "sign in required")
			return
		}
		next(w, r)
	}
}

func writeAuthError(w http.ResponseWriter, status int, code, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_, _ = w.Write([]byte(`{"error":"` + message + `","code":"` + code + `"}`))
}