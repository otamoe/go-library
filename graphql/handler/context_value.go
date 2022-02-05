package handler

import (
	"context"
	"net/http"
)

type (
	ContextValue struct {
		Name    string
		Factory func(ctx context.Context) interface{}
	}
)

func (contextValue *ContextValue) Handler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		value := contextValue.Factory(ctx)
		ctx = context.WithValue(ctx, contextValue.Name, value)
		r = r.WithContext(ctx)
		next.ServeHTTP(w, r)
	})
}
