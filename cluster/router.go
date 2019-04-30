package cluster

import (
	"github.com/dubbo/dubbo-go/config"
	"github.com/dubbo/dubbo-go/protocol"
)

// Extension - Router

type RouterFactory interface {
	Router(config.URL) Router
}

type Router interface {
	Route([]protocol.Invoker, config.URL, protocol.Invocation) []protocol.Invoker
}

type RouterChain struct {
	routers []Router
}

func NewRouterChain(url config.URL) {

}
