package libbadger

import (
	"github.com/dgraph-io/badger/v3"
	liblogger "github.com/otamoe/go-library/logger"
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

func Logger() (out OutOption) {
	out.Option = func(o badger.Options) (badger.Options, error) {
		return o.WithLogger(&compatLogger{liblogger.Get("badger").Sugar()}), nil
	}
	return
}
