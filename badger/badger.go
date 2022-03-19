package libbadger

import (
	"github.com/dgraph-io/badger/v3"
	"go.uber.org/fx"
)

func New(options badger.Options) fx.Option {
	return fx.Options(
		fx.Provide(NewOptions),
		fx.Provide(NewExtendedOption),
		fx.Provide(ViperLoggerLevel),

		fx.Provide(Logger),

		fx.Provide(NewBadger),
	)
}
