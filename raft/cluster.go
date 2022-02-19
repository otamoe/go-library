package libraft

import (
	"context"
	"errors"

	"github.com/dgraph-io/badger/v3"
	"github.com/lni/dragonboat/v3"
	dconfig "github.com/lni/dragonboat/v3/config"
	dstatemachine "github.com/lni/dragonboat/v3/statemachine"
	libraftpb "github.com/otamoe/go-library/raft/pb"
)

type (
	Cluster struct {
		nodeHost *dragonboat.NodeHost
		db       *badger.DB
	}
)

func NewCluster(db *badger.DB, nodeHost *dragonboat.NodeHost) *Cluster {
	return &Cluster{
		db:       db,
		nodeHost: nodeHost,
	}
}

func (cluster *Cluster) Start(bootstrap map[uint64]string, rc dconfig.Config, event StateMachineEventFunc) (err error) {
	if len(bootstrap) == 0 {
		err = errors.New("bootstrap node is nil")
		return
	}
	join := true

	if rc.NodeID == 0 {
		rc.NodeID = AddrRaftNodeIDP(cluster.nodeHost.RaftAddress())
	}

	// 初始节点 join 是 false
	if _, ok := bootstrap[rc.NodeID]; ok {
		join = false
	}

	initialMembers := make(map[uint64]dragonboat.Target, len(bootstrap))
	for nodeID, value := range bootstrap {
		initialMembers[nodeID] = dragonboat.Target(value)
	}

	if rc.ClusterID == 0 {
		err = errors.New("Cluster id is empty")
		return
	}

	if rc.ElectionRTT == 0 {
		rc.ElectionRTT = 10
	}
	if rc.HeartbeatRTT == 0 {
		rc.HeartbeatRTT = 1
	}
	if rc.CheckQuorum == false {
		rc.CheckQuorum = true
	}
	if rc.SnapshotEntries == 0 {
		rc.SnapshotEntries = 100000
	}
	if rc.CompactionOverhead == 0 {
		rc.CompactionOverhead = 20000
	}
	rc.DisableAutoCompactions = true

	if err = cluster.nodeHost.StartConcurrentCluster(initialMembers, join, NewStateMachine(cluster.db, event), rc); err != nil {
		return
	}
	return
}

func (cluster *Cluster) Lookup(ctx context.Context, clusterID uint64, lookupRequest *libraftpb.LookupRequest) (data []*libraftpb.ResponseData, err error) {
	var res interface{}
	if res, err = cluster.nodeHost.SyncRead(ctx, clusterID, lookupRequest); err != nil {
		// 其他错误
		if err != dragonboat.ErrClusterNotFound {
			return
		}
		// 集群未找到 在全局集群里面寻找
		if res, err = cluster.grpcLookup(ctx, clusterID, lookupRequest); err != nil {
			return
		}
	}

	response, ok := res.(*libraftpb.Response)
	if !ok {
		err = errors.New("Response error")
		return
	}

	if response.Error != "" {
		err = errors.New(response.Error)
		return
	}
	if response.Index != -2 {
		err = errors.New("Response error")
		return
	}

	data = response.Data
	return
}

func (cluster *Cluster) Update(ctx context.Context, clusterID uint64, updateRequest *libraftpb.UpdateRequest) (data []*libraftpb.ResponseData, err error) {
	var cmd []byte
	if cmd, err = updateRequest.Marshal(); err != nil {
		return
	}

	var dresult dstatemachine.Result
	if dresult, err = cluster.nodeHost.SyncPropose(ctx, cluster.nodeHost.GetNoOPSession(clusterID), cmd); err != nil {
		// 其他错误
		if err != dragonboat.ErrClusterNotFound {
			return
		}
		// 集群未找到 在全局集群里面寻找
		if dresult, err = cluster.grpcUpdate(ctx, clusterID, cmd); err != nil {
			return
		}
	}

	if len(dresult.Data) == 0 {
		err = errors.New("Response error")
		return
	}

	// 解码数据
	response := &libraftpb.Response{}
	if err = response.Unmarshal(dresult.Data); err != nil {
		return
	}

	// 有错误
	if response.Error != "" {
		err = errors.New(response.Error)
		return
	}

	// 未知错误
	if dresult.Value == StateMachineResultCodeFailure || response.Index != -2 {
		err = errors.New("Response error")
		return
	}

	data = response.Data

	return
}

func (cluster *Cluster) NodeHost() *dragonboat.NodeHost {
	return cluster.nodeHost
}

func (cluster *Cluster) grpcLookup(ctx context.Context, clusterID uint64, lookupRequest *libraftpb.LookupRequest) (data []*libraftpb.ResponseData, err error) {
	return
}

func (cluster *Cluster) grpcUpdate(ctx context.Context, clusterID uint64, cmd []byte) (result dstatemachine.Result, err error) {
	return
}
