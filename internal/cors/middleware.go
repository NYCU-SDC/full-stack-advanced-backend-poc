package cors

import (
	"go.uber.org/zap"
	"net/http"
	"slices"
)

type Middleware struct {
	logger       *zap.Logger
	allowOrigins []string
}

func NewMiddleware(logger *zap.Logger, allowOrigins []string) Middleware {
	logger.Info("CORS middleware initialized", zap.Strings("allow_origins", allowOrigins))
	return Middleware{
		logger:       logger,
		allowOrigins: allowOrigins,
	}
}

func (m Middleware) HandlerFunc(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")

		if origin == "" {
			next.ServeHTTP(w, r)
			return
		}

		if slices.Contains(m.allowOrigins, "*") {
			w.Header().Set("Access-Control-Allow-Origin", "*")
		} else if slices.Contains(m.allowOrigins, origin) {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Access-Control-Allow-Credentials", "true")
		} else {
			m.logger.Warn("CORS request from disallowed origin", zap.String("origin", origin))
			http.Error(w, "CORS not allowed", http.StatusForbidden)
			return
		}

		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	}
}
