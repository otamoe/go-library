package libcontext

import (
	"context"

	"go.uber.org/fx"
)

var Context context.Context

func New(ctx context.Context) fx.Option {
	return fx.Options(
		fx.Provide(WithContext(ctx)),
	)
}
