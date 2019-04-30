package cluster

import (
	"github.com/dubbo/dubbo-go/cluster"
	"github.com/dubbo/dubbo-go/common/extension"
	"github.com/dubbo/dubbo-go/protocol"
)

type RegistryAwareCluster struct {
}

func init() {
	extension.SetCluster("registryAware", NewRegistryAwareCluster)
}

func NewRegistryAwareCluster() cluster.Cluster {
	return &RegistryAwareCluster{}
}

func (cluster *RegistryAwareCluster) Join(directory cluster.Directory) protocol.Invoker {
	return newFailoverClusterInvoker(directory)
}
