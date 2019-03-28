package loadBalance

import (
	"github.com/dubbo/dubbo-go/client"
	"github.com/dubbo/dubbo-go/service"
	"sync/atomic"
)



type RoundRobinSelector struct{
}

func NewRoundRobinSelector()Selector{
	return &RoundRobinSelector{}
}

func (s *RoundRobinSelector) Select(ID int64,array client.ServiceArrayIf) (*service.ServiceURL, error) {

	idx := atomic.AddInt64(array.GetIdx(), 1)

		idx = (ID + idx) % int64(array.GetSize())
	//default: // random
	//	idx = ((int64)(rand.Int()) + ID) % int64(arrSize)
	//}

	return array.GetService(int(idx)), nil
}
