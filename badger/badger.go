package libbadger

import (
	"os"
	"path"

	"github.com/dgraph-io/badger/v3"
	libviper "github.com/otamoe/go-library/viper"
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

func init() {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		panic(err)
	}
	libviper.SetDefault("badger.indexDir", path.Join(homeDir, ".{name}/badger/index"), "badger index dir")
	libviper.SetDefault("badger.valueDir", path.Join(homeDir, ".{name}/badger/value"), "badger value dir")
}
