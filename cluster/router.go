package cluster

import (
	"github.com/dubbo/go-for-apache-dubbo/common"
	"github.com/dubbo/go-for-apache-dubbo/protocol"
)

// Extension - Router

type RouterFactory interface {
	Router(common.URL) Router
}

type Router interface {
	Route([]protocol.Invoker, common.URL, protocol.Invocation) []protocol.Invoker
}

type RouterChain struct {
	routers []Router
}

func NewRouterChain(url common.URL) {

}
