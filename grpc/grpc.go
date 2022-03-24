package libgrpc

import (
	"time"

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
		fx.Provide(NewExtendedServerOptions),
	)
}

func NewClient() fx.Option {
	return fx.Options(
		fx.Provide(DialOption(grpc.WithMaxHeaderListSize(1024*4))),
		fx.Provide(NewClientConn),
		fx.Provide(NewExtendedDialOptions),
	)
}
