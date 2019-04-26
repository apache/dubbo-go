package extension

import (
	"github.com/dubbo/dubbo-go/cluster"
)

var (
	clusters = make(map[string]func() cluster.Cluster)
)

func SetCluster(name string, fcn func() cluster.Cluster) {
	clusters[name] = fcn
}

func GetCluster(name string) cluster.Cluster {
	return clusters[name]()
}
