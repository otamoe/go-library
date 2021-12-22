package liblogger

import (
	"go.uber.org/fx"
	"go.uber.org/zap/zapcore"
)

func New(core zapcore.Core) fx.Option {
	return fx.Options(
		fx.Provide(WithCore(core)),
		fx.Provide(Logger),
		fx.WithLogger(FxLogger),
	)
}
