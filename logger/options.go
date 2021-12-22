package liblogger

import (
	"context"
	"os"

	"github.com/spf13/viper"
	"go.uber.org/fx"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type (
	InOptions struct {
		fx.In
		Options []zap.Option `group:"zapOptions"`
	}

	OutOption struct {
		fx.Out
		Option zap.Option `group:"zapOptions"`
	}
)

func WithCore(core zapcore.Core) func(v *viper.Viper) zapcore.Core {
	return func(v *viper.Viper) zapcore.Core {
		if core == nil {
			encoderCfg := zapcore.EncoderConfig{
				MessageKey:     "msg",
				LevelKey:       "level",
				NameKey:        "logger",
				EncodeLevel:    zapcore.LowercaseLevelEncoder,
				EncodeTime:     zapcore.RFC3339TimeEncoder,
				EncodeDuration: zapcore.StringDurationEncoder,
			}

			if v.GetString("env") == "development" {
				core = zapcore.NewCore(zapcore.NewConsoleEncoder(encoderCfg), os.Stdout, zap.DebugLevel)
			} else {
				core = zapcore.NewCore(zapcore.NewJSONEncoder(encoderCfg), os.Stdout, zap.InfoLevel)
			}
		}
		return core
	}
}

func Logger(core zapcore.Core, out InOptions, lc fx.Lifecycle) (logger *zap.Logger) {
	logger = zap.New(core, out.Options...)
	lc.Append(fx.Hook{
		OnStop: func(c context.Context) error {
			return logger.Sync()
		},
	})
	return
}

func WithOption(o zap.Option) func() (out OutOption) {
	return func() (out OutOption) {
		out.Option = out
		return
	}
}

func WithLogger(logger *zap.Logger) func() *zap.Logger {
	return func() *zap.Logger {
		return logger
	}
}
