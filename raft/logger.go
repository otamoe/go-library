package libraft

import (
	dlogger "github.com/lni/dragonboat/v3/logger"
	liblogger "github.com/otamoe/go-library/logger"
	"go.uber.org/zap"
)

type (
	Logger struct {
		*zap.SugaredLogger
		name string
	}
)

func (logger *Logger) SetLevel(level dlogger.LogLevel) {
	switch level {
	case dlogger.DEBUG:
		liblogger.SetLevel("raft."+logger.name, zap.DebugLevel)
	case dlogger.INFO:
		liblogger.SetLevel("raft."+logger.name, zap.InfoLevel)
	case dlogger.WARNING:
		liblogger.SetLevel("raft."+logger.name, zap.WarnLevel)
	case dlogger.ERROR:
		liblogger.SetLevel("raft."+logger.name, zap.ErrorLevel)
	case dlogger.CRITICAL:
		liblogger.SetLevel("raft."+logger.name, zap.DPanicLevel)
	default:
		liblogger.SetLevel("raft."+logger.name, zap.InfoLevel)
	}
}

func (logger *Logger) Warningf(format string, args ...interface{}) {
	logger.Warnf(format, args...)
}
