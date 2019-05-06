package cluster

import (
	"github.com/dubbo/go-for-apache-dubbo/cluster"
	"github.com/dubbo/go-for-apache-dubbo/config"
	"github.com/dubbo/go-for-apache-dubbo/protocol"
)

type MockCluster struct {
}

func NewMockCluster() cluster.Cluster {
	return &MockCluster{}
}

func (cluster *MockCluster) Join(directory cluster.Directory) protocol.Invoker {
	return protocol.NewBaseInvoker(config.URL{})
}
