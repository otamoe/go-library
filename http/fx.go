package libhttp

import (
	"context"

	"go.uber.org/fx"
)

type (
	InOptions struct {
		fx.In
		Options []Option `group:"httpOptions"`
	}

	OutOption struct {
		fx.Out
		Option Option `group:"httpOptions"`
	}
)

func OptionsFX(options ...Option) fx.Option {
	fxOptions := make([]fx.Option, len(options))
	for i, opt := range options {
		func(i int, opt Option) {
			fxOptions[i] = fx.Provide(func() (out OutOption) {
				out.Option = opt
				return
			})
		}(i, opt)
	}
	return fx.Options(fxOptions...)
}

func OptionFX(option Option) fx.Option {
	return OptionsFX(option)
}

func NewFX(withOptions ...Option) fx.Option {
	return fx.Provide(func(inOptions InOptions, lc fx.Lifecycle) (server *Server, err error) {
		withOptions = append(withOptions, inOptions.Options...)
		if server, err = New(withOptions...); err != nil {
			return
		}

		lc.Append(fx.Hook{
			OnStart: func(_ context.Context) error {
				return server.Start()
			},
			OnStop: func(ctx context.Context) error {
				return server.Shutdown(ctx)
			},
		})

		return
	})
}
