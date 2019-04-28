package cluster

import (
	"context"
)

import (
	"github.com/dubbo/dubbo-go/cluster"
	"github.com/dubbo/dubbo-go/common/extension"
	"github.com/dubbo/dubbo-go/protocol"
)

type FailoverCluster struct {
	context context.Context
}

const name = "failover"

func init() {
	extension.SetCluster(name, NewFailoverCluster)
}

func NewFailoverCluster() cluster.Cluster {
	return &FailoverCluster{}
}

func (cluster *FailoverCluster) Join(directory cluster.Directory) protocol.Invoker {
	return NewFailoverClusterInvoker(directory)
}
