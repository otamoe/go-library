package libcontext

import (
	"context"

	"go.uber.org/fx"
)

func WithContext(mctx context.Context) func(lc fx.Lifecycle) context.Context {
	if mctx == nil {
		mctx = context.Background()
	}
	return func(lc fx.Lifecycle) context.Context {
		ctx, cancel := context.WithCancel(mctx)
		lc.Append(fx.Hook{
			OnStop: func(_ context.Context) error {
				cancel()
				return nil
			},
		})
		return ctx
	}
}
