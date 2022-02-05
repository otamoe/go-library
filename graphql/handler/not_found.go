package handler

import (
	"encoding/json"
	"net/http"
)

type (
	NotFound struct {
	}
)

// 未找到控制器
func (h *NotFound) Handler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		j := map[string]interface{}{
			"errors": []map[string]string{
				{
					"message": http.StatusText(http.StatusNotFound),
				},
			},
			"data": nil,
		}
		b, _ := json.Marshal(j)
		w.Write(b)
	})
}
