package libconfig

import (
	"github.com/spf13/viper"
	"go.uber.org/fx"
)

func NewFX(v *viper.Viper, stop bool) (fxOption fx.Option) {
	if v == nil {
		v = GetConfig()
		return
	}
	fxOption = fx.Provide(func(lc fx.Lifecycle) *viper.Viper {
		return v
	})
	return
}
