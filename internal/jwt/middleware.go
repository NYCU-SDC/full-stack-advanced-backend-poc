package jwt

import (
	"context"
	"go.uber.org/zap"
	"net/http"
)

const UserContextKey = "user"

type Verifier interface {
	Parse(ctx context.Context, tokenString string) (User, error)
}

type Middleware struct {
	logger   *zap.Logger
	verifier Verifier
}

func NewMiddleware(logger *zap.Logger, verifier Verifier) Middleware {
	return Middleware{
		logger:   logger,
		verifier: verifier,
	}
}

func (m Middleware) HandlerFunc(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		token := r.Header.Get("Authorization")
		if token == "" {
			m.logger.Warn("Authorization header required")
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		jwtUser, err := m.verifier.Parse(ctx, token)
		if err != nil {
			m.logger.Warn("Authorization header invalid", zap.Error(err))
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		m.logger.Debug("Authorization header valid", zap.String("user_id", jwtUser.ID.String()))
		r = r.WithContext(context.WithValue(ctx, UserContextKey, jwtUser.ID))
		next.ServeHTTP(w, r)
	}
}
