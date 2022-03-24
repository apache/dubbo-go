package xds

import (
	"strings"
)

type Addr struct {
	HostnameOrIP string
	Port         string
}

func NewAddr(addr string) Addr {
	addrs := strings.Split(addr, ":")
	return Addr{
		HostnameOrIP: addrs[0],
		Port:         addrs[1],
	}
}

func (a *Addr) String() string {
	return a.HostnameOrIP + ":" + a.Port
}

type Cluster struct {
	Bound  string
	Addr   Addr
	Subset string
}

func NewCluster(clusterID string) Cluster {
	clusterIDs := strings.Split(clusterID, "|")
	return Cluster{
		Bound: clusterIDs[0],
		Addr: Addr{
			Port:         clusterIDs[1],
			HostnameOrIP: clusterIDs[3],
		},
		Subset: clusterIDs[2],
	}
}
