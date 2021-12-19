package libhttp

import (
	"context"
	"net/http"
	"net/url"
	"sort"
	"strings"

	"github.com/otamoe/go-library/http/certificate"
)

func New(withOptions ...Option) (server *Server, err error) {
	options := DefaultOptions()
	for _, withOption := range withOptions {
		if err = withOption(options); err != nil {
			return
		}
	}
	server = &Server{
		Server: &http.Server{
			ErrorLog:          options.ErrorLog,
			ReadTimeout:       options.ReadTimeout,
			ReadHeaderTimeout: options.ReadHeaderTimeout,
			WriteTimeout:      options.WriteTimeout,
			IdleTimeout:       options.IdleTimeout,
			MaxHeaderBytes:    options.MaxHeaderBytes,
			TLSConfig:         options.TLSConfig,
			BaseContext:       options.BaseContext,
			ConnContext:       options.ConnContext,
			Addr:              options.Addr,
		},
		options: options,
	}
	if server.Addr == "" {
		if server.TLSConfig == nil {
			server.Addr = ":8080"
		} else {
			server.Addr = ":8443"
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
	sort.Slice(options.Handlers, func(i, j int) bool {
		return options.Handlers[i].Index < options.Handlers[j].Index
	})

	// 中间件 倒序 最后面添加的 丢到最里面去
	handlers := map[string][]HandlerFunc{}

	for _, oh := range options.Handlers {

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
	return
}

func Host(r *http.Request, defaultValue string) (host string) {
	if host = r.Header.Get("X-Forwarded-Host"); host != "" {

	} else if host = r.Host; host != "" {

	} else if host = r.Header.Get("X-Host"); host != "" {

	} else if host = r.URL.Host; host != "" {

	} else {
		host = defaultValue
	}

	if u, err := url.Parse("http://" + host); err == nil {
		return u.Hostname()
	}

	return defaultValue
}
