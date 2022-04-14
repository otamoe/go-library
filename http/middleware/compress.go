package middleware

import (
	"bufio"
	"compress/gzip"
	"errors"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"strings"
	"sync"

	// "github.com/google/brotli/go/cbrotli"
	"github.com/andybalholm/brotli"
)

type (
	compressResponseWriter struct {
		ispre    bool
		encoding string
		compress *Compress
		io.Writer
		http.ResponseWriter
	}

	Compress struct {
		Types []string `json:"types,omitempty"`

		Br        bool
		BrQuality int
		BrLGWin   int

		Gzip      bool
		GzipLevel int

		gzipPool *sync.Pool
	}
)

func (w *compressResponseWriter) WriteHeader(status int) {
	w.pre()
	w.ResponseWriter.WriteHeader(status)
}

func (w *compressResponseWriter) Write(b []byte) (int, error) {
	w.pre()
	return w.Writer.Write(b)
}

func (w *compressResponseWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	h, ok := w.ResponseWriter.(http.Hijacker)
	if !ok {
		return nil, nil, errors.New("response does not implement http.Hijacker")
	}
	return h.Hijack()
}

//  预处理
func (w *compressResponseWriter) pre() {
	if w.ispre {
		return
	}
	w.ispre = true

	// 添加 缓存头规则
	vary := w.Header().Get("Vary")
	if vary == "" {
		vary = "Accept-Encoding"
	} else {
		vary += ", Accept-Encoding"
	}
	w.Header().Set("Vary", vary)

	// 类型检查
	contentType := w.Header().Get("Content-Type")
	var typeMatch bool
	for _, typ := range w.compress.Types {
		if strings.HasPrefix(contentType, typ) {
			typeMatch = true
			break
		}
	}
	if !typeMatch {
		return
	}

	w.Header().Set("Content-Encoding", w.encoding)
	w.Header().Del("Content-Length")

	// 编码
	switch w.encoding {
	case "br":
		w.Writer = brotli.NewWriterOptions(w.Writer, brotli.WriterOptions{
			LGWin:   w.compress.BrLGWin,
			Quality: w.compress.BrQuality,
		})
	case "gzip":
		gz := w.compress.gzipPool.Get().(*gzip.Writer)
		gz.Reset(w.Writer)
		w.Writer = gz
	}
}

func (w *compressResponseWriter) Close() (err error) {
	switch writer := w.Writer.(type) {
	case *gzip.Writer:
		err = writer.Close()
		w.compress.gzipPool.Put(w.Writer)
	case *brotli.Writer:
		err = writer.Close()
	}
	return
}

func (compress *Compress) Handler(next http.Handler) http.Handler {
	compress.gzipPool = &sync.Pool{
		New: func() interface{} {
			writer, err := gzip.NewWriterLevel(ioutil.Discard, compress.GzipLevel)
			if err != nil {
				panic(err)
			}
			return writer
		},
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		compressW := &compressResponseWriter{encoding: compress.getEncoding(r), compress: compress, ResponseWriter: w, Writer: w}
		defer compressW.Close()
		next.ServeHTTP(compressW, r)
	})
}

func (compress *Compress) getEncoding(req *http.Request) (encoding string) {
	if req.Method == http.MethodOptions {
		return
	}
	if req.Proto == "HTTP/1.0" {
		return
	}
	if strings.Contains(req.Header.Get("Connection"), "Upgrade") {
		return
	}

	for _, val := range strings.Split(req.Header.Get("Accept-Encoding"), ",") {
		val = strings.TrimSpace(val)
		if compress.Br && val == "br" {
			encoding = val
			break
		}
		if compress.Gzip && val == "gzip" {
			encoding = val
		}
	}
	return
}
