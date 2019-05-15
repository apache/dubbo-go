package extension

import (
	"github.com/dubbo/go-for-apache-dubbo/cluster"
)

var (
	clusters = make(map[string]func() cluster.Cluster)
)

func SetCluster(name string, fcn func() cluster.Cluster) {
	clusters[name] = fcn
}

func GetCluster(name string) cluster.Cluster {
	if clusters[name] == nil {
		panic("cluster for " + name + " is not existing, you must import corresponding package.")
	}
	return clusters[name]()
}
