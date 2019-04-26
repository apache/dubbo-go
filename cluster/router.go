package cluster

import (
	"github.com/dubbo/dubbo-go/config"
	"github.com/dubbo/dubbo-go/protocol"
)

// Extension - Router

type RouterFactory interface {
	Router(config.IURL) Router
}

type Router interface {
	Route([]protocol.Invoker, config.IURL, protocol.Invocation) []protocol.Invoker
}

type RouterChain struct {
	routers []Router
}

func NewRouterChain(url config.URL) {

}
