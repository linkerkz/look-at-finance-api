package middleware

import (
	"net/http"
	"runtime/debug"

	"github.com/lukivan8/look-at-finance-api/internal/shared/logger"
	"github.com/lukivan8/look-at-finance-api/internal/shared/response"
)

func Recovery(log *logger.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if err := recover(); err != nil {
					log.Errorw("panic recovered",
						"error", err,
						"stack", string(debug.Stack()),
						"path", r.URL.Path,
						"method", r.Method,
					)
					response.InternalServerError(w, "an unexpected error occurred")
				}
			}()
			next.ServeHTTP(w, r)
		})
	}
}

