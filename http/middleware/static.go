package middleware

import (
	"net/http"
	"os"
	"path"
	"strconv"
	"strings"
	"time"
)

type (
	Static struct {
		MaxAge   int
		Prefix   string
		FS       http.FileSystem
		FSPath   string
		ModTime  time.Time
		Redirect string
		Logger   bool
	}
)

func (static *Static) Handler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		upath := r.URL.Path
		if !strings.HasPrefix(upath, "/") {
			upath = "/" + upath
		}
		if static.Prefix != "" && !strings.HasPrefix(upath, static.Prefix) {
			// 没匹配到
			next.ServeHTTP(w, r)
			return
		}

		upath = path.Clean(upath)
		if !strings.HasPrefix(upath, "/") {
			upath = "/" + upath
		}

		f, err := static.FS.Open(path.Join(static.FSPath, upath))

		// 重定向链接
		if err != nil && static.Redirect != "" && os.IsNotExist(err) {
			f, err = static.FS.Open(path.Join(static.FSPath, static.Redirect))
		}

		if err != nil {
			// 文件没找到 跳到下一个
			if os.IsNotExist(err) {
				next.ServeHTTP(w, r)
				return
			}

			// 权限不正确
			if os.IsPermission(err) {
				http.Error(w, "403 Forbidden", http.StatusForbidden)
				return
			}

			// 其他错误
			http.Error(w, "500 Internal Server Error", http.StatusInternalServerError)
			return
		}
		defer f.Close()

		d, err := f.Stat()
		if err != nil {
			// 文件没找到 跳到下一个
			if os.IsNotExist(err) {
				next.ServeHTTP(w, r)
				return
			}

			// 权限不正确
			if os.IsPermission(err) {
				http.Error(w, "403 Forbidden", http.StatusForbidden)
				return
			}

			// 其他错误
			http.Error(w, "500 Internal Server Error", http.StatusInternalServerError)
			return
		}

		LoggerEnable(r.Context(), static.Logger)

		if d.IsDir() {
			if len(upath) > 2 && len(r.URL.Path) > 2 && r.URL.Path[0] == '/' && r.URL.Path[len(r.URL.Path)-1] != '/' {
				//  是目录 并且重定向
				newPath := upath + "/"
				if q := r.URL.RawQuery; q != "" {
					newPath += "?" + q
				}
				http.Redirect(w, r, newPath, http.StatusMovedPermanently)
				return
			}
			f.Close()
			f, err = static.FS.Open(path.Join(static.FSPath, upath, "/index.html"))
			if err != nil {
				// 文件没找到 或 权限不正确
				if os.IsNotExist(err) || os.IsPermission(err) {
					http.Error(w, "403 Forbidden", http.StatusForbidden)
					return
				}

				// 其他错误
				http.Error(w, "500 Internal Server Error", http.StatusInternalServerError)
				return
			}
			defer f.Close()
			d, err = f.Stat()
			if err != nil {
				// 文件没找到 或 权限不正确
				if os.IsNotExist(err) || os.IsPermission(err) {
					http.Error(w, "403 Forbidden", http.StatusForbidden)
					return
				}

				// 其他错误
				http.Error(w, "500 Internal Server Error", http.StatusInternalServerError)
				return
			}

			// 还是目录
			if d.IsDir() {
				if os.IsNotExist(err) || os.IsPermission(err) {
					http.Error(w, "403 Forbidden", http.StatusForbidden)
					return
				}
			}
		} else {
			if len(upath) > 2 && len(r.URL.Path) > 2 && r.URL.Path[0] == '/' && r.URL.Path[len(r.URL.Path)-1] == '/' {
				//  是文件 并且重定向
				newPath := upath
				if q := r.URL.RawQuery; q != "" {
					newPath += "?" + q
				}
				http.Redirect(w, r, newPath, http.StatusMovedPermanently)
				return
			}
		}

		modTime := static.ModTime
		if modTime.IsZero() {
			modTime = d.ModTime()
		}

		responseEtag := `"` + strconv.FormatInt(d.Size(), 36) + strconv.FormatInt(modTime.Unix(), 36) + `"`

		// 必须匹配 否则返回 412
		if ifMatch := r.Header.Get("If-Match"); ifMatch != "" && ifMatch != "*" && ifMatch != responseEtag {
			w.WriteHeader(http.StatusPreconditionFailed)
			return
		}

		// 必须不匹配 否则返回 304
		if ifNoneMatch := r.Header.Get("If-None-Match"); ifNoneMatch == responseEtag || ifNoneMatch == `W/`+responseEtag {
			w.WriteHeader(http.StatusNotModified)
			return
		}

		// Last-Modified
		w.Header().Set("Cache-Control", "public, max-age="+strconv.Itoa(static.MaxAge))
		w.Header().Set("Etag", responseEtag)
		w.Header().Set("Expires", time.Now().UTC().Add(time.Second*time.Duration(static.MaxAge)).Format(http.TimeFormat))

		http.ServeContent(w, r, d.Name(), modTime, f)
	})
}
