package libbadger

import (
	"fmt"
	"os"
	"path"

	"github.com/dgraph-io/badger/v3"
	libconfig "github.com/otamoe/go-library/config"
	liblogger "github.com/otamoe/go-library/logger"
	"github.com/shirou/gopsutil/v3/mem"
)

func init() {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		panic(err)
	}
	libconfig.SetDefault("badger.indexDir", path.Join(homeDir, "."+libconfig.GetName(), "badger", "index"), "Badger index dir")
	libconfig.SetDefault("badger.valueDir", path.Join(homeDir, "."+libconfig.GetName(), "badger", "value"), "Badger index dir")
}

var DB *badger.DB

func GetDB() *badger.DB {
	return DB
}

func SetDB(v *badger.DB) {
	DB = v
}

func Close() error {
	return DB.Close()
}

func DefaultOptions() badger.Options {
	memorySize := GetMemorySize()
	return badger.DefaultOptions(libconfig.GetString("badger.indexDir")).
		WithValueDir(libconfig.GetString("badger.valueDir")).
		WithBaseTableSize(1024 * 1024 * 8).
		WithMemTableSize(int64(memorySize / 32)).
		WithValueThreshold(1024 * 1).
		WithBlockCacheSize(int64(memorySize / 32)).
		WithIndexCacheSize(int64(memorySize / 32)).
		WithLogger(NewLogger(liblogger.GetLogger()))
}

func GetMemorySize() uint64 {
	// 读取内存
	memStat, err := mem.VirtualMemory()
	if err != nil {
		panic(fmt.Errorf("get memory size", err))
	}
	return memStat.Total
}
