package selector

import (
	"math/rand"
	"sync/atomic"
)

import (
	"github.com/dubbo/go-for-apache-dubbo/client"
	"github.com/dubbo/go-for-apache-dubbo/registry"
)

type RandomSelector struct{}

func NewRandomSelector() Selector {
	return &RandomSelector{}
}

func (s *RandomSelector) Select(ID int64, array client.ServiceArrayIf) (registry.ServiceURL, error) {
	if array.GetSize() == 0 {
		return nil, ServiceArrayEmpty
	}

	idx := atomic.AddInt64(array.GetIdx(), 1)
	idx = ((int64)(rand.Int()) + ID) % array.GetSize()
	return array.GetService(idx), nil
}
