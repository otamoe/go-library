package libbadger

import (
	"github.com/dgraph-io/badger/v3"
	"go.uber.org/zap"
)

type compatLogger struct {
	*zap.SugaredLogger
}

func (logger *compatLogger) Warning(args ...interface{}) {
	logger.SugaredLogger.Warn(args...)
}

func (logger *compatLogger) Warningf(format string, args ...interface{}) {
	logger.SugaredLogger.Warnf(format, args...)
}

func NewLogger(logger *zap.Logger) badger.Logger {
	return &compatLogger{
		SugaredLogger: logger.Sugar(),
	}
}

func Logger(logger *zap.Logger) (out OutOption) {
	out.Option = func(o badger.Options) (badger.Options, error) {
		return o.WithLogger(NewLogger(logger.Named("badger"))), nil
	}
	return
}
