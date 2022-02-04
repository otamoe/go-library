package libraft

import (
	"github.com/lni/dragonboat/v3"
	libgrpc "github.com/otamoe/go-library/grpc"
	libraftpb "github.com/otamoe/go-library/raft/pb"
	"google.golang.org/grpc"
)

func NewGrpcServer(nodeHost *dragonboat.NodeHost) (out libgrpc.OutServer, grpcServer *GrpcServer) {
	grpcServer = &GrpcServer{nodeHost: nodeHost}
	out.Server = func(server *grpc.Server) (err error) {
		libraftpb.RegisterRaftServer(server, grpcServer)
		return
	}
	return
}

type (
	GrpcServer struct {
		nodeHost *dragonboat.NodeHost
	}
)

func (g *GrpcServer) Lookup(lookup libraftpb.Raft_LookupServer) (err error) {
	return nil
}

func (g *GrpcServer) Update(update libraftpb.Raft_UpdateServer) (err error) {
	return nil
}
