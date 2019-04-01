package selector

import (
	"sync/atomic"
)

import (
	"github.com/dubbo/dubbo-go/client"
	"github.com/dubbo/dubbo-go/registry"
)

type RoundRobinSelector struct{}

func NewRoundRobinSelector() Selector {
	return &RoundRobinSelector{}
}

func (s *RoundRobinSelector) Select(ID int64, array client.ServiceArrayIf) (*registry.ServiceURL, error) {
	if array.GetSize() == 0 {
		return nil, ServiceArrayEmpty
	}

	idx := atomic.AddInt64(array.GetIdx(), 1)
	idx = (ID + idx) % array.GetSize()
	return array.GetService(idx), nil
}
