package libraft

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"

	dragonboat "github.com/lni/dragonboat/v3"
	dclient "github.com/lni/dragonboat/v3/client"
	dconfig "github.com/lni/dragonboat/v3/config"
	dstatemachine "github.com/lni/dragonboat/v3/statemachine"
	libraftpb "github.com/otamoe/go-library/raft-pb"
)

// raft address 错误
var ErrRaftAddress = errors.New("raft address error")

func Cluster(nh *dragonboat.NodeHost, bootstrap map[uint64]string, rc dconfig.Config, event StateMachineEventFunc) (err error) {
	logDBFactory, ok := nh.NodeHostConfig().Expert.LogDBFactory.(*LogDBFactory)
	if !ok {
		err = errors.New("Expert.LogDBFactory badger, logger")
		return
	}
	if len(bootstrap) == 0 {
		err = errors.New("bootstrap node is nil")
		return
	}
	join := true

	if rc.NodeID == 0 {
		rc.NodeID = AddrRaftNodeIDP(nh.RaftAddress())
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

	if err = nh.StartConcurrentCluster(initialMembers, join, NewStateMachine(logDBFactory.db, logDBFactory.logger.Named(fmt.Sprintf("cluster-%s", rc.ClusterID)), event), rc); err != nil {
		return
	}
	return
}

func SyncRead(ctx context.Context, nh *dragonboat.NodeHost, clusterID uint64, lookup *libraftpb.Lookup) (data []*libraftpb.Items, err error) {
	var res interface{}
	if res, err = nh.SyncRead(ctx, clusterID, lookup); err != nil {
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

func SyncPropose(ctx context.Context, nh *dragonboat.NodeHost, session *dclient.Session, update *libraftpb.Update) (data []*libraftpb.Items, err error) {
	var cmd []byte
	if cmd, err = update.Marshal(); err != nil {
		return
	}

	var dresult dstatemachine.Result
	if dresult, err = nh.SyncPropose(ctx, session, cmd); err != nil {
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

func AddrRaftNodeID(addr string) (nodeID uint64, err error) {
	s := strings.Split(addr, ":")
	if len(s) != 2 {
		err = ErrRaftAddress
		return
	}
	bits := strings.Split(s[0], ".")
	if len(bits) != 4 {
		err = ErrRaftAddress
		return
	}

	b0, e := strconv.Atoi(bits[0])
	if e != nil {
		err = ErrRaftAddress
		return
	}
	b1, e := strconv.Atoi(bits[1])
	if e != nil {
		err = ErrRaftAddress
		return
	}
	b2, e := strconv.Atoi(bits[2])
	if e != nil {
		err = ErrRaftAddress
		return
	}
	b3, e := strconv.Atoi(bits[3])
	if e != nil {
		err = ErrRaftAddress
		return
	}

	nodeID += uint64(b0) << 24
	nodeID += uint64(b1) << 16
	nodeID += uint64(b2) << 8
	nodeID += uint64(b3)

	port, e := strconv.Atoi(s[1])
	if e != nil {
		err = ErrRaftAddress
		return
	}

	nodeID = nodeID<<16 + uint64(port)

	return
}

func AddrRaftNodeIDP(addr string) uint64 {
	nodeID, err := AddrRaftNodeID(addr)
	if err != nil {
		panic(err)
	}
	return nodeID
}
