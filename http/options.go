package libhttp

import (
	"context"
	"crypto/tls"
	"log"
	"net"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/otamoe/go-library/http/certificate"
	"go.uber.org/fx"
)

type (
	Server struct {
		*http.Server
		StartTimeout time.Duration
		Handlers     []HandlerOption
	}

	InOptions struct {
		fx.In
		Options []Option `group:"httpOptions"`
	}

	OutOption struct {
		fx.Out
		Option Option `group:"httpOptions"`
	}

	Option func(server *Server) error

	HandlerFunc func(next http.Handler) http.Handler

	HandlerOption struct {
		Index   int
		Hosts   []string
		Handler HandlerFunc
	}
)

func NewServer(inOptions InOptions, lc fx.Lifecycle) (server *Server, err error) {
	server = DefaultServer()
	for _, option := range inOptions.Options {
		if err = option(server); err != nil {
			return
		}
	}

	if server.TLSConfig == nil && (server.Addr == ":443" || server.Addr == ":8443") {
		var cert *certificate.Certificate
		if cert, err = certificate.CreateTLSCertificate("ecdsa", 384, "localhost", []string{"localhost"}, false, nil); err != nil {
			return
		}

		if server.TLSConfig, err = certificate.TLSConfig([]*certificate.Certificate{cert}); err != nil {
			return
		}
	}

	// 控制器 未找到
	notFoundHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
	})

	// 控制器 排序
	sort.Slice(server.Handlers, func(i, j int) bool {
		return server.Handlers[i].Index < server.Handlers[j].Index
	})

	// 中间件 倒序 最后面添加的 丢到最里面去
	handlers := map[string][]HandlerFunc{}

	for _, oh := range server.Handlers {

		if len(oh.Hosts) == 0 {
			oh.Hosts = []string{"*"}
		}
		for _, host := range oh.Hosts {
			host = strings.ToLower(host)
			if host == "*" || host == "" {
				// 全局控制器
				for k, _ := range handlers {
					handlers[k] = append(handlers[k], oh.Handler)
				}

				// 没 * 写入到 *
				if _, ok := handlers[""]; !ok {
					handlers[""] = []HandlerFunc{}
					handlers[""] = append(handlers[""], oh.Handler)
				}
			} else {
				// 局部控制器
				if _, ok := handlers[host]; !ok {
					handlers[host] = []HandlerFunc{}

					// 添加 * host 的
					if _, ok := handlers[""]; ok {
						handlers[host] = append(handlers[host], handlers[""]...)
					}

				}
				handlers[host] = append(handlers[host], oh.Handler)
			}
		}
	}

	httpHandlers := map[string]http.Handler{}
	for host, value := range handlers {
		// 中间件 倒序 最后面添加的 丢到最里面去
		var httpHandler http.Handler
		httpHandler = notFoundHandler

		for i := len(value) - 1; i >= 0; i-- {
			httpHandler = value[i](httpHandler)
		}

		httpHandlers[host] = httpHandler
	}

	server.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		host := Host(r, "")
		r = r.WithContext(context.WithValue(r.Context(), "hostname", host))
		if handler, ok := httpHandlers[host]; ok {
			handler.ServeHTTP(w, r)
		} else if handler, ok := httpHandlers[""]; ok {
			handler.ServeHTTP(w, r)
		} else {
			notFoundHandler.ServeHTTP(w, r)
		}
	})

	lc.Append(fx.Hook{
		OnStart: func(c context.Context) (err error) {
			t := time.NewTimer(server.StartTimeout)
			defer t.Stop()
			errc := make(chan error, 1)
			go func() {
				var e error
				if server.TLSConfig == nil {
					e = server.ListenAndServe()
				} else {
					e = server.ListenAndServeTLS("", "")
				}
				if e == http.ErrServerClosed {
					e = nil
				}
				errc <- e
			}()
			select {
			case <-t.C:
			case err = <-errc:
			}
			return
		},
		OnStop: func(c context.Context) error {
			return server.Shutdown(c)
		},
	})

	return
}

func WithListenAddress(addr string) func() (out OutOption) {
	return func() (out OutOption) {
		out.Option = func(server *Server) error {
			server.Addr = addr
			return nil
		}
		return
	}
}
func WithErrorLog(logger *log.Logger) func() (out OutOption) {
	return func() (out OutOption) {
		out.Option = func(server *Server) error {
			server.Server.ErrorLog = logger
			return nil
		}
		return
	}
}

func WithReadTimeout(t time.Duration) func() (out OutOption) {
	return func() (out OutOption) {
		out.Option = func(server *Server) error {
			server.ReadTimeout = t
			return nil
		}
		return
	}
}

func WithReadHeaderTimeout(t time.Duration) func() (out OutOption) {
	return func() (out OutOption) {
		out.Option = func(server *Server) error {
			server.ReadHeaderTimeout = t
			return nil
		}
		return
	}
}

func WithWriteTimeout(t time.Duration) func() (out OutOption) {
	return func() (out OutOption) {
		out.Option = func(server *Server) error {
			server.WriteTimeout = t
			return nil
		}
		return
	}
}
func WithIdleTimeout(t time.Duration) func() (out OutOption) {
	return func() (out OutOption) {
		out.Option = func(server *Server) error {
			server.IdleTimeout = t
			return nil
		}
		return
	}
}

func WithMaxHeaderBytes(s int) func() (out OutOption) {
	return func() (out OutOption) {
		out.Option = func(server *Server) error {
			server.MaxHeaderBytes = s
			return nil
		}
		return
	}
}

func WithTLSConfig(s *tls.Config) func() (out OutOption) {
	return func() (out OutOption) {
		out.Option = func(server *Server) error {
			server.TLSConfig = s
			return nil
		}
		return
	}
}

func WithStartTimeout(b time.Duration) func() (out OutOption) {
	return func() (out OutOption) {
		out.Option = func(server *Server) error {
			server.StartTimeout = b
			return nil
		}
		return
	}
}
func WithBaseContext(b func(net.Listener) context.Context) func() (out OutOption) {
	return func() (out OutOption) {
		out.Option = func(server *Server) error {
			server.BaseContext = b
			return nil
		}
		return
	}
}

func WithConnContext(b func(ctx context.Context, c net.Conn) context.Context) func() (out OutOption) {
	return func() (out OutOption) {
		out.Option = func(server *Server) error {
			server.ConnContext = b
			return nil
		}
		return
	}
}

func WithRegisterOnShutdown(f func()) func() (out OutOption) {
	return func() (out OutOption) {
		out.Option = func(server *Server) error {
			server.RegisterOnShutdown(f)
			return nil
		}
		return
	}
}

func WithHandler(hosts []string, index int, handler HandlerFunc) func() (out OutOption) {
	return func() (out OutOption) {
		out.Option = func(server *Server) error {
			server.Handlers = append(server.Handlers, HandlerOption{Hosts: hosts, Index: index, Handler: handler})
			return nil
		}
		return
	}
}

func DefaultServer() *Server {
	return &Server{
		Server: &http.Server{
			Addr:              ":8080",
			ReadTimeout:       time.Second * 18000,
			ReadHeaderTimeout: time.Second * 10,
			WriteTimeout:      time.Second * 18000,
			IdleTimeout:       time.Second * 1800,
			MaxHeaderBytes:    4096,
		},
		StartTimeout: time.Second * 2,
	}
}
