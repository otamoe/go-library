package libraft

import (
	"strconv"
	"strings"
)

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
