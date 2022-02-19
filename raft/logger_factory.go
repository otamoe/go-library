package libraft

import (
	"sync"

	goLog "github.com/ipfs/go-log/v2"
	dlogger "github.com/lni/dragonboat/v3/logger"
	"go.uber.org/zap"
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
	atomicLevel := zap.NewAtomicLevel()

	loggerFactory.pkgs[pkgName] = &Logger{
		SugaredLogger: goLog.Logger("raft." + pkgName).Desugar().WithOptions(zap.IncreaseLevel(atomicLevel)).Sugar(),
		atomicLevel:   atomicLevel,
	}
	return loggerFactory.pkgs[pkgName]
}

func init() {
	loggerFactory := &LoggerFactory{
		pkgs: make(map[string]*Logger),
	}
	dlogger.SetLoggerFactory(loggerFactory.Create)
}
