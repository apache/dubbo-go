package loadBalance

import (
	"github.com/dubbo/dubbo-go/client"
	"github.com/dubbo/dubbo-go/service"
	"math/rand"
	"sync/atomic"
)



type RandomSelector struct{
}

func NewRandomSelector()Selector{
	return &RandomSelector{}
}

func (s *RandomSelector) Select(ID int64,array client.ServiceArrayIf) (*service.ServiceURL, error) {

	idx := atomic.AddInt64(array.GetIdx(), 1)

	idx = ((int64)(rand.Int()) + ID) % int64(array.GetSize())

	return array.GetService(int(idx)), nil
}
