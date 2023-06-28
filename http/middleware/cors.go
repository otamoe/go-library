package middleware

import (
	"net/http"
	"strconv"
	"strings"
)

type (
	Cors struct {
		Methods       []string
		Origins       []string
		Headers       []string
		ExposeHeaders []string
		MaxAge        int
	}
)

func (cors *Cors) Handler(next http.Handler) http.Handler {

	maxAge := strconv.Itoa(cors.MaxAge)
	origins := strings.Join(cors.Origins, ", ")
	methods := strings.Join(cors.Methods, ", ")
	if methods == "" {
		methods = "*"
	}
	headers := strings.Join(cors.Headers, ", ")
	if headers == "" {
		headers = "Content-Type, Authorization, User-Agent, Range, If-Match, If-Modified-Since, If-None-Match, If-Range, If-Unmodified-Since, X-Requested-With"
	}

	exposeHeaders := strings.Join(cors.ExposeHeaders, ", ")
	if exposeHeaders == "" {
		exposeHeaders = "Accept-Ranges, Content-Range, Content-Length, Content-Disposition, ETag, Date, X-Chunked-Output, X-Stream-Output"
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if origins != "" {
			w.Header().Set("Access-Control-Allow-Methods", methods)
			w.Header().Set("Access-Control-Allow-Headers", headers)
			w.Header().Set("Access-Control-Expose-Headers", exposeHeaders)
			w.Header().Set("Access-Control-Allow-Origin", origins)
			w.Header().Set("Access-Control-Max-Age", maxAge)
			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusOK)
				return
			}
		}
		next.ServeHTTP(w, r)
	})
}
