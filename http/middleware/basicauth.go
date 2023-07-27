package middleware

import (
	"encoding/base64"
	"net/http"
	"strings"
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
				if inputUsername, inputPassword, ok = parseBasicAuth(r.URL.Query().Get("authorization")); !ok {
					return
				}
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

func parseBasicAuth(auth string) (username, password string, ok bool) {
	const prefix = "Basic "
	// Case insensitive prefix match. See Issue 22736.
	if len(auth) < len(prefix) || !equalFold(auth[:len(prefix)], prefix) {
		return "", "", false
	}
	c, err := base64.StdEncoding.DecodeString(auth[len(prefix):])
	if err != nil {
		return "", "", false
	}
	cs := string(c)
	username, password, ok = strings.Cut(cs, ":")
	if !ok {
		return "", "", false
	}
	return username, password, true
}

// equalFold is strings.equalFold, ASCII only. It reports whether s and t
// are equal, ASCII-case-insensitively.
func equalFold(s, t string) bool {
	if len(s) != len(t) {
		return false
	}
	for i := 0; i < len(s); i++ {
		if lower(s[i]) != lower(t[i]) {
			return false
		}
	}
	return true
}

// lower returns the ASCII lowercase version of b.
func lower(b byte) byte {
	if 'A' <= b && b <= 'Z' {
		return b + ('a' - 'A')
	}
	return b
}
