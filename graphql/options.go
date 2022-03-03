package libgraphql

import (
	"compress/gzip"
	"context"
	"embed"
	"errors"
	"fmt"
	"net/http"
	"sort"
	"time"

	"github.com/99designs/gqlgen/graphql"
	ghandler "github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/playground"
	"github.com/otamoe/go-library/graphql/handler"
	libhttp "github.com/otamoe/go-library/http"
	libhttpMiddleware "github.com/otamoe/go-library/http/middleware"
	liblogger "github.com/otamoe/go-library/logger"
	"github.com/spf13/viper"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type (
	Graphql struct {
		Host     string
		Handlers []Handler
	}
	InOptions struct {
		fx.In
		Options []Option `group:"graphqlOptions"`
	}

	OutOption struct {
		fx.Out
		Option Option `group:"graphqlOptions"`
	}

	Option func(graphql *Graphql) error

	Handler struct {
		Handler libhttp.HandlerFunc
		Index   int
		Name    string
	}
)

func Host() (out OutOption) {
	out.Option = func(graphql *Graphql) error {
		graphql.Host = viper.GetString("graphql.host")
		return nil
	}
	return
}

func Compress() (out OutOption) {
	out.Option = func(graphql *Graphql) error {
		compress := &libhttpMiddleware.Compress{
			Types:     []string{"text/", "application/json", "application/javascript", "application/atom+xml", "application/rss+xml", "application/xml"},
			Gzip:      true,
			GzipLevel: gzip.DefaultCompression,
			BrLGWin:   24,
			BrQuality: 11,
		}
		graphql.Handlers = append(graphql.Handlers, Handler{
			Handler: compress.Handler,
			Index:   500,
			Name:    "compress",
		})
		return nil
	}
	return
}
func Cors() (out OutOption) {
	out.Option = func(graphql *Graphql) error {
		cors := &libhttpMiddleware.Cors{
			Methods: []string{"GET", "HEAD", "OPTIONS", "POST", "PUT", "DELETE"},
			Origins: []string{"*"},
			MaxAge:  86400 * 31,
		}
		graphql.Handlers = append(graphql.Handlers, Handler{
			Handler: cors.Handler,
			Index:   600,
			Name:    "cors",
		})
		return nil
	}
	return
}

func Logger() (out OutOption) {
	out.Option = func(graphql *Graphql) error {
		logger := &libhttpMiddleware.Logger{
			Logger:    liblogger.Get("http.graphql"),
			SlowQuery: time.Second * 30,
			Forwarded: true,
		}
		graphql.Handlers = append(graphql.Handlers, Handler{
			Handler: logger.Handler,
			Index:   700,
			Name:    "logger",
		})
		return nil
	}
	return
}

func LoggerDisable() (out OutOption) {
	out.Option = func(graphql *Graphql) error {
		loggerDisable := &handler.LoggerEnable{
			Enable: false,
		}
		graphql.Handlers = append(graphql.Handlers, Handler{
			Handler: loggerDisable.Handler,
			Index:   1900,
			Name:    "loggerDisable",
		})
		return nil
	}
	return
}

//go:embed public
var staticFS embed.FS

func Static() (out OutOption) {
	// static 中间件
	static := &libhttpMiddleware.Static{
		FSPath:  "public",
		MaxAge:  86400 * 31,
		FS:      staticFS,
		ModTime: time.Date(2010, time.January, 1, 1, 0, 0, 0, time.UTC),
	}

	out.Option = func(graphql *Graphql) error {
		graphql.Handlers = append(graphql.Handlers, Handler{
			Handler: static.Handler,
			Index:   2000,
			Name:    "static",
		})
		return nil
	}
	return
}

func NotFound() (out OutOption) {
	notFound := &handler.NotFound{}
	out.Option = func(graphql *Graphql) error {
		graphql.Handlers = append(graphql.Handlers, Handler{
			Handler: notFound.Handler,
			Index:   9900,
			Name:    "notFound",
		})
		return nil
	}
	return
}

func ContextValue(name string, factory func(ctx context.Context) interface{}, index int) func() (out OutOption) {
	return func() (out OutOption) {
		out.Option = func(graphql *Graphql) error {
			contextValue := &handler.ContextValue{
				Name:    name,
				Factory: factory,
			}
			graphql.Handlers = append(graphql.Handlers, Handler{
				Handler: contextValue.Handler,
				Index:   index,
				Name:    name,
			})
			return nil
		}
		return
	}
}

func Server(server *ghandler.Server, path string) func() (out OutOption) {
	return func() (out OutOption) {
		if path == "" {
			path = "/"
		}
		playgroundHandler := playground.Handler("GraphQL playground", "/")
		out.Option = func(graphql *Graphql) error {
			graphql.Handlers = append(graphql.Handlers, Handler{
				Handler: func(next http.Handler) http.Handler {
					return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
						switch r.URL.Path {
						case path:
							server.ServeHTTP(w, r)
						case "/playground":
							playgroundHandler.ServeHTTP(w, r)
						default:
							next.ServeHTTP(w, r)
						}
					})
				},
				Name:  "server",
				Index: 1000,
			})
			return nil
		}
		return
	}
}

func Recover() func(ctx context.Context, err interface{}) error {
	logger := liblogger.Get("http.graphql.recover")
	return func(ctx context.Context, err interface{}) (res error) {
		switch val := err.(type) {
		case string:
			res = errors.New(val)
		case error:
			res = val
		default:
			res = errors.New(fmt.Sprintf("%+v", err))
		}

		defer func() {
			r := recover()
			if r != nil {
				logger.Error(
					"recover",
					zap.Stack("stack"),
					zap.Error(errors.New(fmt.Sprintf("%+v", err))),
				)
			}
		}()

		rctx := graphql.GetRequestContext(ctx)
		var rawQuery string
		if rctx != nil {
			rawQuery = rctx.RawQuery
		}
		logger.Error(
			res.Error(),
			zap.Stack("stack"),
			zap.Error(res),
			zap.String("rawQuery", rawQuery),
		)
		return
	}
}

func NewGraphql(inOptions InOptions) (graphql *Graphql, httpOutOption libhttp.OutOption) {
	graphql = &Graphql{}
	for _, o := range inOptions.Options {
		o(graphql)
	}
	// 控制器 排序
	sort.Slice(graphql.Handlers, func(i, j int) bool {
		return graphql.Handlers[i].Index < graphql.Handlers[j].Index
	})
	handlers := []Handler{}
	for _, handler := range graphql.Handlers {
		handlers = append(handlers, handler)
	}
	httpOutOption.Option = func(server *libhttp.Server) error {
		existsHandler := map[string]bool{}
		for _, handler := range handlers {
			if handler.Name != "" {
				if val, _ := existsHandler[handler.Name]; val {
					continue
				}
				existsHandler[handler.Name] = true
			}
			server.Handlers = append(server.Handlers, libhttp.HandlerOption{
				Hosts:   []string{graphql.Host},
				Handler: handler.Handler,
				Index:   handler.Index,
			})
		}
		return nil
	}
	return
}
