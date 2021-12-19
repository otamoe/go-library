package liblogger

import (
	"os"

	libconfig "github.com/otamoe/go-library/config"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var Logger *zap.Logger

func New(core zapcore.Core) *zap.Logger {
	if core == nil {
		core = Core(nil)
	}
	return zap.New(core)
}

func SetLogger(logger *zap.Logger) {
	Logger = logger
}

func GetLogger() *zap.Logger {
	return Logger
}

func Sync() error {
	return Logger.Sync()
}

func Core(core zapcore.Core) zapcore.Core {
	if core == nil {
		encoderCfg := zapcore.EncoderConfig{
			MessageKey:     "msg",
			LevelKey:       "level",
			NameKey:        "logger",
			EncodeLevel:    zapcore.LowercaseLevelEncoder,
			EncodeTime:     zapcore.RFC3339TimeEncoder,
			EncodeDuration: zapcore.StringDurationEncoder,
		}

		if libconfig.GetString("env") == "development" {
			core = zapcore.NewCore(zapcore.NewConsoleEncoder(encoderCfg), os.Stdout, zap.DebugLevel)
		} else {
			core = zapcore.NewCore(zapcore.NewJSONEncoder(encoderCfg), os.Stdout, zap.InfoLevel)
		}
	}
	return core
}
