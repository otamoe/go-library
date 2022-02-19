package libraft

import (
	"github.com/dgraph-io/badger/v3"
	dconfig "github.com/lni/dragonboat/v3/config"
	draftio "github.com/lni/dragonboat/v3/raftio"
)

type (
	LogDBFactory struct {
		db *badger.DB
	}
)

func (logDBFactory *LogDBFactory) Name() string {
	return "badger"
}
func (logDBFactory *LogDBFactory) Create(nhc dconfig.NodeHostConfig, logDBCallback dconfig.LogDBCallback, valueDirs []string, indexDirs []string) (logdb draftio.ILogDB, err error) {
	logdb, err = NewLogDB(logDBFactory.db)
	return
}

func NewLogDBFactory(db *badger.DB) (out OutNodeHostConfig) {
	out.Config = func(nhc dconfig.NodeHostConfig) (dconfig.NodeHostConfig, error) {
		nhc.Expert.LogDBFactory = &LogDBFactory{
			db: db,
		}
		return nhc, nil
	}
	return
}
