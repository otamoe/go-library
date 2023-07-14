package middleware

import (
	"net/http"
)

type BasicAuth struct {
	Auths    map[string]string
	Header   bool
	Verified func(r *http.Request) bool
}

func (basicAuth *BasicAuth) Handler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var ok bool
		defer func() {
			if !ok {
				if basicAuth.Header {
					w.Header().Set("WWW-Authenticate", `Basic realm="Login", charset="UTF-8"`)
				}
				w.WriteHeader(http.StatusUnauthorized)
				return
			}
		}()

		// 已认证的
		if basicAuth.Verified != nil {
			ok = basicAuth.Verified(r)
		}

		if !ok {
			// 没有认证
			var inputUsername string
			var inputPassword string
			if inputUsername, inputPassword, ok = r.BasicAuth(); !ok {
				return
			}

			// 用户名不存在
			var password string
			if password, ok = basicAuth.Auths[inputUsername]; !ok {
				return
			}

			// 密码不匹配
			if password != inputPassword {
				ok = false
				return
			}
			ok = true
		}

		next.ServeHTTP(w, r)
	})
}
