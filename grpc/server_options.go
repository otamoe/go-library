package libgrpc

import (
	"context"
	"net"
	"time"

	"go.uber.org/fx"
	"google.golang.org/grpc"
)

type (
	InServers struct {
		fx.In
		Servers []Server `group:"grpcServers"`
	}

	OutServer struct {
		fx.Out
		Server Server `group:"grpcServers"`
	}

	InServerOptions struct {
		fx.In
		Options []grpc.ServerOption `group:"grpcServerOptions"`
	}

	OutServerOption struct {
		fx.Out
		Option grpc.ServerOption `group:"grpcServerOptions"`
	}

	ListenAddress string
	Server        func(s *grpc.Server) (err error)
)

func ServerOption(o grpc.ServerOption) func() (out OutServerOption) {
	return func() (out OutServerOption) {
		out.Option = o
		return
	}
}

func RegisterServer(s Server) func() (out OutServer) {
	return func() (out OutServer) {
		out.Server = s
		return
	}
}

func NewServer(inServerOptions InServerOptions, inServers InServers, listenAddress ListenAddress, lc fx.Lifecycle) (server *grpc.Server, err error) {
	server = grpc.NewServer(inServerOptions.Options...)

	// 注册服务
	for _, s := range inServers.Servers {
		if err = s(server); err != nil {
			return
		}
	}

	// 启动停止
	lc.Append(fx.Hook{
		OnStart: func(c context.Context) (err error) {
			var lis net.Listener
			if lis, err = net.Listen("tcp", string(listenAddress)); err != nil {
				return
			}
			go server.Serve(lis)
			return
		},

		OnStop: func(c context.Context) (err error) {
			ch := make(chan struct{})
			go func() {
				server.GracefulStop()
				close(ch)
			}()
			// 10s 宽限期
			t := time.NewTimer(time.Second * 10)
			defer t.Stop()
			select {
			case <-t.C:
				server.Stop()
			case <-ch:
			}
			return
		},
	})

	return
}
