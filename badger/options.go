package libbadger

import (
	"context"
	"fmt"
	"os"
	"path"
	"time"

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

	ExtendedOptions struct {
		GCDiscardRatio float64
		GCInterval     time.Duration
		GCSleep        time.Duration
	}

	InExtendedOptions struct {
		fx.In
		Options []ExtendedOption `group:"badgerExtendedOptions"`
	}

	OutExtendedOption struct {
		fx.Out
		Option ExtendedOption `group:"badgerExtendedOptions"`
	}

	ExtendedOption func(extendedOptions *ExtendedOptions) (err error)
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

func NewExtendedOption(inExtendedOptions InExtendedOptions) (extendedOptions *ExtendedOptions, err error) {
	extendedOptions = &ExtendedOptions{
		GCDiscardRatio: 0.5,
		GCInterval:     time.Minute * 15,
		GCSleep:        time.Second * 15,
	}
	// 扩展选项
	for _, o := range inExtendedOptions.Options {
		if err = o(extendedOptions); err != nil {
			return
		}
	}

	return
}

func NewBadger(lc fx.Lifecycle, options badger.Options, extendedOptions *ExtendedOptions) (db *badger.DB, err error) {

	if options.Dir != "" {
		if err = os.MkdirAll(options.Dir, 0755); err != nil {
			return
		}
	}
	if options.ValueDir != "" {
		if err = os.MkdirAll(options.ValueDir, 0755); err != nil {
			return
		}
	}
	if db, err = badger.Open(options); err != nil {
		return
	}
	ctx, cancel := context.WithCancel(context.Background())
	lc.Append(fx.Hook{
		OnStart: func(_ context.Context) error {
			if extendedOptions.GCDiscardRatio != 0 && extendedOptions.GCInterval != 0 {
				go GC(ctx, extendedOptions, db)
			}
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
	memTableSize := memorySize / 32
	if memTableSize > 1024*1024*256 {
		memTableSize = 1024 * 1024 * 256
	}
	blockCacheSize := memorySize / 32
	if blockCacheSize > 1024*1024*1024 {
		blockCacheSize = 1024 * 1024 * 1024
	}
	indexCacheSize := memorySize / 32
	if indexCacheSize > 1024*1024*1024 {
		indexCacheSize = 1024 * 1024 * 1024
	}
	return badger.DefaultOptions(path.Join(homeDir, ".badger/index")).
		WithValueDir(path.Join(homeDir, ".badger/value")).
		WithBaseTableSize(1024 * 1024 * 4).
		WithMemTableSize(memTableSize).
		WithValueThreshold(1024 * 1).
		WithBlockCacheSize(blockCacheSize).
		WithIndexCacheSize(indexCacheSize)
}

func GetMemorySize() int64 {
	// 读取内存
	memStat, err := mem.VirtualMemory()
	if err != nil {
		panic(fmt.Errorf("get memory size", err))
	}
	return int64(memStat.Total)
}
