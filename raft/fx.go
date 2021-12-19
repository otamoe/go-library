package libraft

import (
	"context"
	"errors"

	"github.com/lni/dragonboat/v3"
	"go.uber.org/fx"
)

func NewFX(nh *dragonboat.NodeHost, stop bool) (fxOption fx.Option) {
	if nh == nil {
		nh = GetNodeHost()
	}
	if nh == nil {
		return fx.Error(errors.New("dragonboat node host is nil"))
	}
	return fx.Provide(func(lc fx.Lifecycle) *dragonboat.NodeHost {
		lc.Append(fx.Hook{
			OnStop: func(c context.Context) error {
				if stop {
					nh.Stop()
				}
				return nil
			},
		})
		return nh
	})
}
