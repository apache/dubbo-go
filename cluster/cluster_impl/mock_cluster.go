package cluster

import (
	"github.com/dubbo/go-for-apache-dubbo/cluster"
	"github.com/dubbo/go-for-apache-dubbo/protocol"
)

type mockCluster struct {
}

func NewMockCluster() cluster.Cluster {
	return &mockCluster{}
}

func (cluster *mockCluster) Join(directory cluster.Directory) protocol.Invoker {
	return protocol.NewBaseInvoker(directory.GetUrl())
}
