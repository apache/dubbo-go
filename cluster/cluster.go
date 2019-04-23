package cluster

import "github.com/dubbo/dubbo-go/protocol"

type Cluster interface {
	Join(Directory)protocol.Invoker
}
