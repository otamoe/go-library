package libgraphql

import (
	libviper "github.com/otamoe/go-library/viper"
	"go.uber.org/fx"
)

func New(hosts []string) (fxOption fx.Option) {
	return fx.Provide(
		fx.Provide(libviper.WithSetDefault("graphql.host", "graphql.localhost", "graphql host")),
		fx.Provide(Host),
		fx.Provide(Compress),
		fx.Provide(Cors),
		fx.Provide(Logger),
		fx.Provide(LoggerDisable),
		fx.Provide(Static),
		fx.Provide(NotFound),

		fx.Provide(NewGraphql),
	)
}
