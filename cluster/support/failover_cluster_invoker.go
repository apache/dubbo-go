package cluster

import (
	"context"
	"github.com/dubbo/dubbo-go/cluster"
	"github.com/dubbo/dubbo-go/protocol"
)

type failoverClusterInvoker struct {
	baseClusterInvoker
}

func NewFailoverClusterInvoker(ctx context.Context, directory cluster.Directory) protocol.Invoker {
	return &failoverClusterInvoker{
		baseClusterInvoker: newBaseClusterInvoker(ctx, directory),
	}
}

func (invoker *failoverClusterInvoker) Invoke() {

}
