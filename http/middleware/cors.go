package middleware

import (
	"net/http"
	"strconv"
	"strings"
)

type (
	Cors struct {
		Methods []string
		Origins []string
		MaxAge  int
	}
)

func (cors *Cors) Handler(next http.Handler) http.Handler {

	maxAge := strconv.Itoa(cors.MaxAge)
	origins := strings.Join(cors.Origins, ", ")
	methods := strings.Join(cors.Methods, ", ")
	if methods == "" {
		methods = "*"
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if origins != "" {
			w.Header().Set("Access-Control-Allow-Methods", methods)
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, User-Agent, Range, If-Match, If-Modified-Since, If-None-Match, If-Range, If-Unmodified-Since, X-Requested-With")
			w.Header().Set("Access-Control-Expose-Headers", "Accept-Ranges, Content-Range, Content-Length, Content-Disposition, ETag, Date, X-Chunked-Output, X-Stream-Output")
			w.Header().Set("Access-Control-Max-Age", maxAge)
			w.Header().Set("Access-Control-Allow-Origin", origins)
			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusOK)
				return
			}
		}
		next.ServeHTTP(w, r)
	})
}
