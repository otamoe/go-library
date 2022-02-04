package libraft

type (
	ClusterInfos struct {
		clusters map[uint64]map[uint64]bool
	}
)

const ClusterNodeInfoID = 1
