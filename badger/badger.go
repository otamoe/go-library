package libbadger

import (
	"go.uber.org/fx"
)

func New() fx.Option {
	return fx.Options(
		fx.Provide(NewOptions),
		fx.Provide(NewExtendedOption),
		fx.Provide(ViperLoggerLevel),

		fx.Provide(Logger),

		fx.Provide(NewBadger),
	)
}
