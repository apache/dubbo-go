package selector

import (
	"math/rand"
	"sync/atomic"
)

import (
	"github.com/dubbo/dubbo-go/client"
	"github.com/dubbo/dubbo-go/registry"
)

type RandomSelector struct{}

func NewRandomSelector() Selector {
	return &RandomSelector{}
}

func (s *RandomSelector) Select(ID int64, array client.ServiceArrayIf) (*registry.DefaultServiceURL, error) {
	if array.GetSize() == 0 {
		return nil, ServiceArrayEmpty
	}

	idx := atomic.AddInt64(array.GetIdx(), 1)
	idx = ((int64)(rand.Int()) + ID) % array.GetSize()
	return array.GetService(idx), nil
}
