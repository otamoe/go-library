package libviper

import (
	"go.uber.org/fx"
)

func New(name string) fx.Option {
	return fx.Options(
		fx.Provide(WithSetDefault("env", "production", "Environment type  production, development, test")),
		fx.Provide(WithEnvPrefix(name)),
		fx.Provide(PFlag),
		fx.Provide(Viper),
	)
}
