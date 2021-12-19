package libhttp

import (
	"context"
	"crypto/tls"
	"log"
	"net"
	"net/http"
	"time"
)

type (
	Options struct {
		Addr              string
		ErrorLog          *log.Logger
		ReadTimeout       time.Duration
		ReadHeaderTimeout time.Duration
		WriteTimeout      time.Duration
		IdleTimeout       time.Duration
		RequestTimeout    time.Duration
		StartTimeout      time.Duration
		MaxHeaderBytes    int
		TLSConfig         *tls.Config
		BaseContext       func(net.Listener) context.Context
		ConnContext       func(ctx context.Context, c net.Conn) context.Context

		Handlers []HandlerOption
	}

	Option func(options *Options) error

	HandlerFunc func(next http.Handler) http.Handler

	HandlerOption struct {
		Index   int
		Hosts   []string
		Handler HandlerFunc
	}
)

func Addr(addr string) Option {
	return func(options *Options) error {
		options.Addr = addr
		return nil
	}
}
func ErrorLog(logger *log.Logger) Option {
	return func(options *Options) error {
		options.ErrorLog = logger
		return nil
	}
}

func ReadTimeout(t time.Duration) Option {
	return func(options *Options) error {
		options.ReadTimeout = t
		return nil
	}
}

func ReadHeaderTimeout(t time.Duration) Option {
	return func(options *Options) error {
		options.ReadHeaderTimeout = t
		return nil
	}
}

func WriteTimeout(t time.Duration) Option {
	return func(options *Options) error {
		options.WriteTimeout = t
		return nil
	}
}
func IdleTimeout(t time.Duration) Option {
	return func(options *Options) error {
		options.IdleTimeout = t
		return nil
	}
}

func MaxHeaderBytes(s int) Option {
	return func(options *Options) error {
		options.MaxHeaderBytes = s
		return nil
	}
}
func TLSConfig(s *tls.Config) Option {
	return func(options *Options) error {
		options.TLSConfig = s
		return nil
	}

}

func StartTimeout(b time.Duration) Option {
	return func(options *Options) error {
		options.StartTimeout = b
		return nil
	}
}
func BaseContext(b func(net.Listener) context.Context) Option {
	return func(options *Options) error {
		options.BaseContext = b
		return nil
	}
}

func ConnContext(b func(ctx context.Context, c net.Conn) context.Context) Option {
	return func(options *Options) error {
		options.ConnContext = b
		return nil
	}
}

func Handler(hosts []string, index int, handler HandlerFunc) Option {
	return func(options *Options) error {
		options.Handlers = append(options.Handlers, HandlerOption{Hosts: hosts, Index: index, Handler: handler})
		return nil
	}
}

func DefaultOptions() *Options {
	return &Options{
		Addr:              ":8080",
		ReadTimeout:       time.Second * 18000,
		ReadHeaderTimeout: time.Second * 10,
		WriteTimeout:      time.Second * 18000,
		IdleTimeout:       time.Second * 1800,
		MaxHeaderBytes:    4096,
		StartTimeout:      time.Second * 2,
	}
}
