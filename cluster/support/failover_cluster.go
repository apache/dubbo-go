package cluster

import (
	"context"
	"github.com/dubbo/dubbo-go/cluster"
	"github.com/dubbo/dubbo-go/common/extension"
	"github.com/dubbo/dubbo-go/protocol"
)

type FailoverCluster struct {
	context context.Context
}
const name = "failover"

func init(){
	extension.SetCluster(name,NewFailoverCluster)
}

func NewFailoverCluster(ctx context.Context) cluster.Cluster {
	return &FailoverCluster{
		context: ctx,
	}
}

func (cluster *FailoverCluster) Join(directory cluster.Directory) protocol.Invoker {
	return NewFailoverClusterInvoker(cluster.context, directory)
}
