package libbadger

import (
	"context"
	"time"

	"github.com/dgraph-io/badger/v3"
)

var GCDiscardRatio = 0.5
var GCInterval = 15 * time.Minute
var GCSleep = 10 * time.Second

func GC(ctx context.Context, db *badger.DB) {
	gcTimeout := time.NewTimer(GCInterval)
	defer gcTimeout.Stop()
	var err error
	for {
		select {
		case <-gcTimeout.C:
			switch db.RunValueLogGC(GCDiscardRatio); err {
			case badger.ErrNoRewrite, badger.ErrRejected:
				// 没写入 被拒绝  15 分钟
				gcTimeout.Reset(GCInterval)
			case nil:
				// 无错误 间隔 10秒
				gcTimeout.Reset(GCSleep)
			case badger.ErrDBClosed:
				// 被关闭 返回
				return
			default:
				// 其他错误
				db.Opts().Logger.Errorf("error during a GC cycle %s", err)
				// Not much we can do on a random error but log it and continue.
				gcTimeout.Reset(GCInterval)
			}
		case <-ctx.Done():
			return
		}
	}
}
