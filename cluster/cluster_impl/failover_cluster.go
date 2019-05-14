package cluster

import (
	"github.com/dubbo/go-for-apache-dubbo/cluster"
	"github.com/dubbo/go-for-apache-dubbo/common/extension"
	"github.com/dubbo/go-for-apache-dubbo/protocol"
)

type failoverCluster struct {
}

const name = "failover"

func init() {
	extension.SetCluster(name, newFailoverCluster)
}

func newFailoverCluster() cluster.Cluster {
	return &failoverCluster{}
}

func (cluster *failoverCluster) Join(directory cluster.Directory) protocol.Invoker {
	return newFailoverClusterInvoker(directory)
}
