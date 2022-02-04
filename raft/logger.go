package libraft

import (
	dlogger "github.com/lni/dragonboat/v3/logger"
	"go.uber.org/zap"
)

type (
	Logger struct {
		*zap.SugaredLogger
		atomicLevel zap.AtomicLevel
	}
)

func (logger *Logger) SetLevel(level dlogger.LogLevel) {
	switch level {
	case dlogger.DEBUG:
		logger.atomicLevel.Enabled(zap.DebugLevel)
	case dlogger.INFO:
		logger.atomicLevel.Enabled(zap.InfoLevel)
	case dlogger.WARNING:
		logger.atomicLevel.Enabled(zap.WarnLevel)
	case dlogger.ERROR:
		logger.atomicLevel.Enabled(zap.ErrorLevel)
	case dlogger.CRITICAL:
		logger.atomicLevel.Enabled(zap.DPanicLevel)
	default:
		logger.atomicLevel.Enabled(zap.InfoLevel)
	}
}

func (logger *Logger) Warningf(format string, args ...interface{}) {
	logger.Warnf(format, args...)
}
