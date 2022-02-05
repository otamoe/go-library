package handler

import (
	"net/http"

	libHttpMiddleware "github.com/otamoe/go-library/http/middleware"
)

type (
	LoggerEnable struct {
		Enable bool
	}
)

func (h *LoggerEnable) Handler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		libHttpMiddleware.LoggerEnable(r.Context(), h.Enable)
		next.ServeHTTP(w, r)
	})
}
