package cluster

import (
	"github.com/dubbo/dubbo-go/cluster"
	"github.com/dubbo/dubbo-go/protocol"
)

type failoverClusterInvoker struct {
	baseClusterInvoker
}

func NewFailoverClusterInvoker(directory cluster.Directory) protocol.Invoker {
	return &failoverClusterInvoker{
		baseClusterInvoker: newBaseClusterInvoker(directory),
	}
}

func (invoker *failoverClusterInvoker) Invoke(invocation protocol.Invocation) protocol.Result {
	invokers := invoker.directory.List(invocation)
	invokers[0].GetUrl()
	return &protocol.RPCResult{}
}
