package libbadger

import (
	"context"
	"errors"

	"github.com/dgraph-io/badger/v3"
	"go.uber.org/fx"
)

func NewFX(db *badger.DB, stop bool) (fxOption fx.Option) {
	if db == nil {
		db = GetDB()
	}
	if db == nil {
		return fx.Error(errors.New("badger db is nil"))
	}
	return fx.Provide(func(lc fx.Lifecycle) *badger.DB {
		ctx, cancel := context.WithCancel(context.Background())
		lc.Append(fx.Hook{
			OnStart: func(_ context.Context) error {
				go GC(ctx, db)
				return nil
			},
			OnStop: func(c context.Context) error {
				cancel()
				if stop {
					return db.Close()
				}
				return nil
			},
		})
		return db
	})
}
