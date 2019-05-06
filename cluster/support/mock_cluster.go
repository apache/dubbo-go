package cluster

import (
	"github.com/dubbo/dubbo-go/cluster"
	"github.com/dubbo/dubbo-go/config"
	"github.com/dubbo/dubbo-go/protocol"
)

type MockCluster struct {
}

func NewMockCluster() cluster.Cluster {
	return &MockCluster{}
}

func (cluster *MockCluster) Join(directory cluster.Directory) protocol.Invoker {
	return protocol.NewBaseInvoker(config.URL{})
}
