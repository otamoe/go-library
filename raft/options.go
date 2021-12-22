package libraft

import (
	"context"
	"errors"

	"github.com/lni/dragonboat/v3"
	dconfig "github.com/lni/dragonboat/v3/config"
	"go.uber.org/fx"
)

type (
	InNodeHostConfigs struct {
		fx.In
		Configs []NodeHostConfig `group:"raftNodeHostConfig"`
	}
	OutNodeHostConfig struct {
		fx.Out
		Config NodeHostConfig `group:"raftNodeHostConfig"`
	}
	NodeHostConfig func(dconfig.NodeHostConfig) (dconfig.NodeHostConfig, error)
)

var ErrRaftAddress = errors.New("raft address error")

func NewNodeHostConfig(inNodeHostConfigs InNodeHostConfigs) (nhc dconfig.NodeHostConfig, err error) {

	nhc = DefaultNodeHostConfig()

	// 配置文件
	for _, c := range inNodeHostConfigs.Configs {
		if nhc, err = c(nhc); err != nil {
			return
		}
	}
	return
}

func NewNodeHost(nhc dconfig.NodeHostConfig, lc fx.Lifecycle) (nh *dragonboat.NodeHost, err error) {
	if nh, err = dragonboat.NewNodeHost(nhc); err != nil {
		return
	}
	lc.Append(fx.Hook{
		OnStop: func(c context.Context) error {
			nh.Stop()
			return nil
		},
	})
	return
}

func DefaultNodeHostConfig() dconfig.NodeHostConfig {
	return dconfig.NodeHostConfig{
		DeploymentID:        1,
		RaftAddress:         "127.0.0.1:6501",
		RTTMillisecond:      2000,
		EnableMetrics:       true,
		MaxSendQueueSize:    1024 * 1024 * 256,
		MaxReceiveQueueSize: 1024 * 1024 * 256,
	}
}
