package liblogger

import (
	"go.uber.org/fx"
)

func New() fx.Option {
	return fx.Options(
		fx.WithLogger(FxLogger),
	)
}
