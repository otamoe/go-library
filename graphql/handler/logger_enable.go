package handler

import (
	"net/http"

	libhttpMiddleware "github.com/otamoe/go-library/http/middleware"
)

type (
	LoggerEnable struct {
		Enable bool
	}
)

func (h *LoggerEnable) Handler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		libhttpMiddleware.LoggerEnable(r.Context(), h.Enable)
		next.ServeHTTP(w, r)
	})
}
