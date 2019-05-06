package cluster

import "github.com/dubbo/go-for-apache-dubbo/protocol"

type Cluster interface {
	Join(Directory) protocol.Invoker
}
