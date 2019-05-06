package cluster

import (
	"github.com/dubbo/go-for-apache-dubbo/cluster"
	"github.com/dubbo/go-for-apache-dubbo/common/extension"
	"github.com/dubbo/go-for-apache-dubbo/protocol"
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
	return newRegistryAwareClusterInvoker(directory)
}
