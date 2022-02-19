package libraft

import (
	"sync"

	dlogger "github.com/lni/dragonboat/v3/logger"
	liblogger "github.com/otamoe/go-library/logger"
)

type (
	LoggerFactory struct {
		mux  sync.Mutex
		pkgs map[string]*Logger
	}
)

func (loggerFactory *LoggerFactory) Create(pkgName string) dlogger.ILogger {
	loggerFactory.mux.Lock()
	defer loggerFactory.mux.Unlock()
	if val, ok := loggerFactory.pkgs[pkgName]; ok {
		return val
	}

	loggerFactory.pkgs[pkgName] = &Logger{
		SugaredLogger: liblogger.Get("raft." + pkgName).Sugar(),
		name:          pkgName,
	}
	return loggerFactory.pkgs[pkgName]
}

func init() {
	loggerFactory := &LoggerFactory{
		pkgs: make(map[string]*Logger),
	}
	dlogger.SetLoggerFactory(loggerFactory.Create)
}
