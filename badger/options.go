package libbadger

import (
	"context"
	"fmt"
	"os"
	"path"

	"github.com/dgraph-io/badger/v3"
	"github.com/shirou/gopsutil/v3/mem"
	"go.uber.org/fx"
)

type (
	InOptions struct {
		fx.In
		Options []Option `group:"badgerOptions"`
	}

	OutOption struct {
		fx.Out
		Option Option `group:"badgerOptions"`
	}

	Option func(badger.Options) (badger.Options, error)
)

func NewOptions(inOptions InOptions) (options badger.Options, err error) {
	options = DefaultOptions()
	for _, o := range inOptions.Options {
		if options, err = o(options); err != nil {
			return
		}
	}
	return
}

func NewBadger(lc fx.Lifecycle, options badger.Options) (db *badger.DB, err error) {
	if db, err = badger.Open(options); err != nil {
		return
	}
	ctx, cancel := context.WithCancel(context.Background())
	lc.Append(fx.Hook{
		OnStart: func(_ context.Context) error {
			go GC(ctx, db)
			return nil
		},
		OnStop: func(c context.Context) error {
			cancel()
			return db.Close()
		},
	})
	return
}

func DefaultOptions() badger.Options {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		panic(err)
	}
	memorySize := GetMemorySize()
	return badger.DefaultOptions(path.Join(homeDir, ".badger/index")).
		WithValueDir(path.Join(homeDir, ".badger/value")).
		WithBaseTableSize(1024 * 1024 * 8).
		WithMemTableSize(int64(memorySize / 32)).
		WithValueThreshold(1024 * 1).
		WithBlockCacheSize(int64(memorySize / 32)).
		WithIndexCacheSize(int64(memorySize / 32))
}

func GetMemorySize() uint64 {
	// 读取内存
	memStat, err := mem.VirtualMemory()
	if err != nil {
		panic(fmt.Errorf("get memory size", err))
	}
	return memStat.Total
}
