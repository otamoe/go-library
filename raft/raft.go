package libraft

import (
	libviper "github.com/otamoe/go-library/viper"
	"go.uber.org/fx"
)

func New() fx.Option {
	return fx.Options(

		fx.Provide(libviper.WithSetDefault("raft.deploymentID", 1, "raft deployment ID")),
		fx.Provide(libviper.WithSetDefault("raft.address", "127.0.0.1:6501", "raft address")),
		fx.Provide(libviper.WithSetDefault("raft.tls.ca", "", "raft tls ca")),
		fx.Provide(libviper.WithSetDefault("raft.tls.cert", "", "raft tls cert")),
		fx.Provide(libviper.WithSetDefault("raft.tls.key", "", "raft tls key")),
		fx.Provide(ViperDeploymentID),
		fx.Provide(ViperAddress),
		fx.Provide(ViperTLS),

		fx.Provide(NewLogDBFactory),
		fx.Provide(NewLoggerFactory),

		fx.Provide(NewNodeHostConfig),
		fx.Provide(NewNodeHost),
		fx.Provide(NewCluster),
	)
}
