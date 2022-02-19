package libraft

import (
	libviper "github.com/otamoe/go-library/viper"
	"go.uber.org/fx"
)

func New() fx.Option {
	return fx.Options(

		fx.Provide(ViperDeploymentID),
		fx.Provide(ViperAddress),
		fx.Provide(ViperListenAddress),
		fx.Provide(ViperTLS),

		fx.Provide(NewLogDBFactory),

		fx.Provide(NewNodeHostConfig),
		fx.Provide(NewNodeGrpcConfig),
		fx.Provide(NewNodeHost),
		fx.Provide(NewCluster),
		fx.Provide(NewGrpcServer),
	)
}

func init() {
	libviper.SetDefault("raft.deploymentID", 1, "raft deployment ID")
	libviper.SetDefault("raft.address", "127.0.0.1:6501", "raft address")
	libviper.SetDefault("raft.listenAddress", "127.0.0.1:6501", "raft listen address")
	libviper.SetDefault("raft.tls.ca", "", "raft tls ca")
	libviper.SetDefault("raft.tls.cert", "", "raft tls cert")
	libviper.SetDefault("raft.tls.key", "", "raft tls key")
}
