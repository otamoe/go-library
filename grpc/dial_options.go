package libgrpc

import (
	"context"

	"go.uber.org/fx"
	"google.golang.org/grpc"
)

type (
	InDialOptions struct {
		fx.In
		Options []grpc.DialOption `group:"grpcDialOptions"`
	}

	OutDialOption struct {
		fx.Out
		Option grpc.DialOption `group:"grpcDialOptions"`
	}
	ExtendedDialOptions struct {
		TargetAddress string
	}

	InExtendedDialOptions struct {
		fx.In
		Options []ExtendedDialOption `group:"grpcExtendedDialOptions"`
	}

	ExtendedDialOption func(extendedDialOptions *ExtendedDialOptions) (err error)

	InClients struct {
		fx.In
		Clients []Client `group:"grpcClients"`
	}

	OutClient struct {
		fx.Out
		Client Client `group:"grpcClients"`
	}

	Client func(ctx context.Context, clientConn *grpc.ClientConn) (err error)
)

func NewExtendedDialOptions(inExtendedDialOptions InExtendedDialOptions) (extendedDialOptions *ExtendedDialOptions, err error) {
	extendedDialOptions = &ExtendedDialOptions{
		TargetAddress: "127.0.0.1:8090",
	}
	// 扩展选项
	for _, o := range inExtendedDialOptions.Options {
		if err = o(extendedDialOptions); err != nil {
			return
		}
	}
	return
}

func DialOption(o grpc.DialOption) func() (out OutDialOption) {
	return func() (out OutDialOption) {
		out.Option = o
		return
	}
}

func RegisterClient(s Client) func() (out OutClient) {
	return func() (out OutClient) {
		out.Client = s
		return
	}
}
func NewClientConn(ctx context.Context, inDialtOptions InDialOptions, inClients InClients, extendedDialOptions *ExtendedDialOptions, lc fx.Lifecycle) (clientConn *grpc.ClientConn, err error) {
	clientConn, err = grpc.DialContext(
		ctx,
		extendedDialOptions.TargetAddress,
		inDialtOptions.Options...,
	)
	if err != nil {
		return
	}

	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) (err error) {
			for _, c := range inClients.Clients {
				if err = c(ctx, clientConn); err != nil {
					return
				}
			}
			return nil
		},
		OnStop: func(ctx context.Context) error {
			return clientConn.Close()
		},
	})
	return
}
