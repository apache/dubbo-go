package cluster

import (
	"github.com/dubbo/dubbo-go/cluster"
	"github.com/dubbo/dubbo-go/common/constant"
	"github.com/dubbo/dubbo-go/protocol"
)

type registryAwareClusterInvoker struct {
	baseClusterInvoker
}

func newRegistryAwareClusterInvoker(directory cluster.Directory) protocol.Invoker {
	return &registryAwareClusterInvoker{
		baseClusterInvoker: newBaseClusterInvoker(directory),
	}
}

func (invoker *registryAwareClusterInvoker) Invoke(invocation protocol.Invocation) protocol.Result {
	invokers := invoker.directory.List(invocation)
	//First, pick the invoker (XXXClusterInvoker) that comes from the local registry, distinguish by a 'default' key.
	for _, invoker := range invokers {
		if invoker.IsAvailable() && invoker.GetUrl().GetParam(constant.REGISTRY_DEFAULT_KEY, "false") == "true" {
			return invoker.Invoke(invocation)
		}
	}

	//If none of the invokers has a local signal, pick the first one available.
	for _, invoker := range invokers {
		if invoker.IsAvailable() {
			return invoker.Invoke(invocation)
		}
	}
	return nil
}
