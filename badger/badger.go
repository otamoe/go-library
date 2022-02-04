package libbadger

import (
	"os"
	"path"

	"github.com/dgraph-io/badger/v3"
	libviper "github.com/otamoe/go-library/viper"
	"go.uber.org/fx"
)

func New(options badger.Options) fx.Option {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		panic(err)
	}

	return fx.Options(
		fx.Provide(libviper.WithSetDefault("badger.indexDir", path.Join(homeDir, ".{name}/badger/index"), "badger index dir")),
		fx.Provide(libviper.WithSetDefault("badger.valueDir", path.Join(homeDir, ".{name}/badger/value"), "badger value dir")),

		fx.Provide(ViperValueDir),
		fx.Provide(ViperIndexDir),

		fx.Provide(NewOptions),
		fx.Provide(ViperLoggerLevel),

		fx.Provide(Logger),

		fx.Provide(NewBadger),
	)
}
