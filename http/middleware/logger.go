package middleware

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"strings"
	"time"

	"go.uber.org/zap"
)

type (
	Logger struct {
		Logger    *zap.Logger
		SlowQuery time.Duration
		Forwarded bool
	}
	responseWriter struct {
		http.ResponseWriter
		status int
	}
)

func (w *responseWriter) WriteHeader(code int) {
	w.status = code
	w.ResponseWriter.WriteHeader(code)
}

func LoggerFields(ctx context.Context, fields ...zap.Field) {
	loggerFields, ok := ctx.Value("LOGGER_FIELDS").(*[]zap.Field)
	if ok {
		b := append(*loggerFields, fields...)
		*loggerFields = b
	}
}

func LoggerEnable(ctx context.Context, enable bool) {
	b, ok := ctx.Value("LOGGER_ENABLE").(*bool)
	if ok {
		*b = enable
	}
}

func (logger *Logger) Handler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		now := time.Now().UTC()
		enable := true
		fields := []zap.Field{}

		ctx := r.Context()
		ctx = context.WithValue(ctx, "LOGGER", logger.Logger)
		ctx = context.WithValue(ctx, "LOGGER_IP", ClientIP(r, logger.Forwarded))
		ctx = context.WithValue(ctx, "LOGGER_TIME", now)
		ctx = context.WithValue(ctx, "LOGGER_ENABLE", &enable)
		ctx = context.WithValue(ctx, "LOGGER_FIELDS", &fields)
		r = r.WithContext(ctx)
		sw := &responseWriter{ResponseWriter: w}
		w = sw
		defer func() {

			var err error
			if rerr := recover(); rerr != nil {

				switch val := rerr.(type) {
				case string:
					err = errors.New(val)
				case error:
					err = val
				default:
					err = errors.New(fmt.Sprintf("%+v", err))
				}
				logger.Logger.Error("recover", zap.Error(err), zap.Stack("stack"))
				http.Error(sw, err.Error(), http.StatusInternalServerError)
			}

			if err == nil {
				if !enable {
					return
				}
			}

			latency := time.Now().UTC().Sub(now)

			fields = append(fields, zap.Int("status", sw.status),
				zap.Duration("latency", latency),
				zap.String("method", r.Method),
				zap.String("referer", r.Referer()),
				zap.String("userAgent", r.UserAgent()),
				zap.String("remoteAddr", r.RemoteAddr),
				zap.String("clientIP", ClientIP(r, logger.Forwarded)))

			if err != nil {

				logger.Logger.Named("recover").With(zap.Error(err)).With(zap.Stack("debug_stack")).Error(
					r.RequestURI,
					fields...,
				)
			} else if sw.status >= 500 {
				// 状态大于 >= 500
				logger.Logger.Named("logger").Error(
					r.RequestURI,
					fields...,
				)
			} else if logger.SlowQuery != 0 && latency > logger.SlowQuery {
				// 慢查询
				logger.Logger.Named("logger").Warn(
					r.RequestURI,
					fields...,
				)
			} else {
				// 其他日志
				logger.Logger.Named("logger").Info(
					r.RequestURI,
					fields...,
				)
			}
		}()
		ctx = r.Context()
		next.ServeHTTP(w, r)
	})
}

func ClientIP(req *http.Request, forwarded bool) string {
	if forwarded {
		clientIP := req.Header.Get("X-Forwarded-For")
		clientIP = strings.TrimSpace(strings.Split(clientIP, ",")[0])
		if clientIP == "" {
			clientIP = strings.TrimSpace(req.Header.Get("X-Real-Ip"))
		}
		if clientIP != "" {
			return clientIP
		}
	}
	if ip, _, err := net.SplitHostPort(strings.TrimSpace(req.RemoteAddr)); err == nil {
		return ip
	}
	return ""
}
