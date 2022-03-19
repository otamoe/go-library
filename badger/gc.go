package libbadger

import (
	"context"
	"time"

	"github.com/dgraph-io/badger/v3"
)

func GC(ctx context.Context, extendedOptions *ExtendedOptions, db *badger.DB) {
	t := time.NewTicker(extendedOptions.GCInterval)
	defer t.Stop()
	var err error
	for {
		select {
		case <-t.C:
			switch db.RunValueLogGC(extendedOptions.GCDiscardRatio); err {
			case badger.ErrNoRewrite, badger.ErrRejected:
				// 没写入 被拒绝
				t.Reset(extendedOptions.GCInterval)
			case nil:
				// 无错误
				t.Reset(extendedOptions.GCSleep)
			case badger.ErrDBClosed:
				// 被关闭 返回

				return
			default:
				// 其他错误
				db.Opts().Logger.Errorf("error during a GC cycle %s", err)
				// Not much we can do on a random error but log it and continue.
				t.Reset(extendedOptions.GCInterval)
			}
		case <-ctx.Done():
			return
		}
	}
}
