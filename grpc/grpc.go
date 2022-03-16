package libgrpc

import (
	"time"

	libviper "github.com/otamoe/go-library/viper"
	"go.uber.org/fx"
	"google.golang.org/grpc"
)

func New() fx.Option {
	return fx.Options(
		fx.Provide(ServerOption(grpc.ConnectionTimeout(time.Second*30))),
		fx.Provide(ServerOption(grpc.InitialConnWindowSize(1024*256))),
		fx.Provide(ServerOption(grpc.InitialWindowSize(1024*256))),
		fx.Provide(ServerOption(grpc.MaxHeaderListSize(1024*4))),
		fx.Provide(ServerOption(grpc.MaxRecvMsgSize(1024*1024*32))),
		fx.Provide(ServerOption(grpc.ReadBufferSize(1024*128))),
		fx.Provide(ServerOption(grpc.WriteBufferSize(1024*128))),
		fx.Provide(NewServer),
	)
}

func init() {
	libviper.SetDefault("grpc.listenAddress", "127.0.0.1:8090", "grpc listen address")
}
