package cluster

import (
	"github.com/dubbo/go-for-apache-dubbo/cluster"
	"github.com/dubbo/go-for-apache-dubbo/common/extension"
	"github.com/dubbo/go-for-apache-dubbo/protocol"
)

type FailoverCluster struct {
}

const name = "failover"

func init() {
	extension.SetCluster(name, NewFailoverCluster)
}

func NewFailoverCluster() cluster.Cluster {
	return &FailoverCluster{}
}

func (cluster *FailoverCluster) Join(directory cluster.Directory) protocol.Invoker {
	return newFailoverClusterInvoker(directory)
}
