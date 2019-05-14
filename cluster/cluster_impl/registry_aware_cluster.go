package cluster

import (
	"github.com/dubbo/go-for-apache-dubbo/cluster"
	"github.com/dubbo/go-for-apache-dubbo/common/extension"
	"github.com/dubbo/go-for-apache-dubbo/protocol"
)

type registryAwareCluster struct {
}

func init() {
	extension.SetCluster("registryAware", newRegistryAwareCluster)
}

func newRegistryAwareCluster() cluster.Cluster {
	return &registryAwareCluster{}
}

func (cluster *registryAwareCluster) Join(directory cluster.Directory) protocol.Invoker {
	return newRegistryAwareClusterInvoker(directory)
}
