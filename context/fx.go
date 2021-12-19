package libcontext

import (
	"context"

	"go.uber.org/fx"
)

func NewFX(mctx context.Context, stop bool) (fxOption fx.Option) {
	if mctx == nil {
		mctx = GetContext()
	}
	fxOption = fx.Provide(func(lc fx.Lifecycle) context.Context {
		ctx, cancel := context.WithCancel(mctx)
		lc.Append(fx.Hook{
			OnStop: func(_ context.Context) error {
				if stop {
					cancel()
				}
				return nil
			},
		})
		return ctx
	})
	return
}
