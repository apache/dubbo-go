package cluster

import (
	"github.com/dubbo/go-for-apache-dubbo/config"
	"github.com/dubbo/go-for-apache-dubbo/protocol"
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
