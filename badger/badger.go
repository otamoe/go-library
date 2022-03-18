package libbadger

import (
	"github.com/dgraph-io/badger/v3"
	"go.uber.org/fx"
)

func New(options badger.Options) fx.Option {

	return fx.Options(

		fx.Provide(ViperValueDir),
		fx.Provide(ViperIndexDir),

		fx.Provide(NewOptions),
		fx.Provide(ViperLoggerLevel),

		fx.Provide(Logger),

		fx.Provide(NewBadger),
	)
}
