package libraft

import (
	"github.com/dgraph-io/badger/v3"
	dconfig "github.com/lni/dragonboat/v3/config"
	draftio "github.com/lni/dragonboat/v3/raftio"
	"go.uber.org/zap"
)

type (
	LogDBFactory struct {
		db     *badger.DB
		logger *zap.Logger
	}
)

func (logDBFactory *LogDBFactory) Name() string {
	return "badger"
}
func (logDBFactory *LogDBFactory) Create(nhc dconfig.NodeHostConfig, logDBCallback dconfig.LogDBCallback, valueDirs []string, indexDirs []string) (logdb draftio.ILogDB, err error) {
	logdb, err = NewLogDB(logDBFactory.db, logDBFactory.logger)
	return
}

func NewLogDBFactory(db *badger.DB, logger *zap.Logger) *LogDBFactory {
	return &LogDBFactory{db: db, logger: logger}
}
