package auth

import (
	"context"
	"net/http"
	"strings"

	"github.com/danii1kq/srunal/internal/platform/httpapi"
)

type contextKey string

const userContextKey contextKey = "srunal.user"

// UserFrom returns the authenticated user stored by Middleware.
func UserFrom(ctx context.Context) (User, bool) {
	user, ok := ctx.Value(userContextKey).(User)
	return user, ok
}

// BearerToken pulls the token out of the Authorization header.
func BearerToken(r *http.Request) string {
	header := r.Header.Get("Authorization")
	const prefix = "Bearer "
	if len(header) > len(prefix) && strings.EqualFold(header[:len(prefix)], prefix) {
		return header[len(prefix):]
	}
	return ""
}

// Middleware rejects any request without a valid session and puts the user in
// the context.
func (s *Service) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user, err := s.Authenticate(r.Context(), BearerToken(r))
		if err != nil {
			httpapi.WriteError(w, err)
			return
		}
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), userContextKey, user)))
	})
}
