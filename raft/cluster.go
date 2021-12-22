package libraft

import (
	"context"
	"errors"
	"fmt"

	"github.com/dgraph-io/badger/v3"
	"github.com/lni/dragonboat/v3"
	dclient "github.com/lni/dragonboat/v3/client"
	dconfig "github.com/lni/dragonboat/v3/config"
	dstatemachine "github.com/lni/dragonboat/v3/statemachine"
	libraftpb "github.com/otamoe/go-library/raft-pb"
	"go.uber.org/zap"
)

type (
	Cluster struct {
		nodeHost *dragonboat.NodeHost
		logger   *zap.Logger
		db       *badger.DB
	}
)

func NewCluster(db *badger.DB, logger *zap.Logger, nodeHost *dragonboat.NodeHost) *Cluster {
	return &Cluster{
		db:       db,
		logger:   logger.Named("raft"),
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

	if err = cluster.nodeHost.StartConcurrentCluster(initialMembers, join, NewStateMachine(cluster.db, cluster.logger.Named(fmt.Sprintf("cluster-%s", rc.ClusterID)), event), rc); err != nil {
		return
	}
	return
}

func (cluster *Cluster) SyncRead(ctx context.Context, clusterID uint64, lookup *libraftpb.Lookup) (data []*libraftpb.Items, err error) {
	var res interface{}
	if res, err = cluster.nodeHost.SyncRead(ctx, clusterID, lookup); err != nil {
		return
	}

	res2, ok := res.(*libraftpb.Result)
	if !ok {
		err = errors.New("Result error")
		return
	}

	if res2.Error != "" {
		err = errors.New(res2.Error)
		return
	}
	if res2.Index != -2 {
		err = errors.New("Result error")
		return
	}

	data = res2.Data
	return
}

func (cluster *Cluster) SyncPropose(ctx context.Context, session *dclient.Session, update *libraftpb.Update) (data []*libraftpb.Items, err error) {
	var cmd []byte
	if cmd, err = update.Marshal(); err != nil {
		return
	}

	var dresult dstatemachine.Result
	if dresult, err = cluster.nodeHost.SyncPropose(ctx, session, cmd); err != nil {
		return
	}

	if len(dresult.Data) == 0 {
		err = errors.New("Result error")
		return
	}

	// 解码数据
	result := &libraftpb.Result{}
	if err = result.Unmarshal(dresult.Data); err != nil {
		return
	}

	// 有错误
	if result.Error != "" {
		err = errors.New(result.Error)
		return
	}

	// 未知错误
	if dresult.Value == StateMachineResultCodeFailure || result.Index != -2 {
		err = errors.New("Result error")
		return
	}

	data = result.Data

	return
}

func (cluster *Cluster) NodeHost() *dragonboat.NodeHost {
	return cluster.nodeHost
}
