package router_chain

import (
	"sort"
)
import (
	"github.com/apache/dubbo-go/cluster"
	"github.com/apache/dubbo-go/common"
	"github.com/apache/dubbo-go/common/extension"
	"github.com/apache/dubbo-go/protocol"
)

type RouterChain struct {
	routers        []cluster.Router
	builtinRouters []cluster.Router
	Invokers       []protocol.Invoker
}

func NewRouterChain(url *common.URL) *RouterChain {
	var builtinRouters []cluster.Router
	factories := extension.GetRouterFactories()
	for _, factory := range factories {
		router, _ := factory.Router(url)
		builtinRouters = append(builtinRouters, router)
	}
	var routers []cluster.Router
	copy(routers, builtinRouters)
	sort.SliceStable(routers, func(i, j int) bool {
		return routers[i].Priority() < routers[j].Priority()
	})
	return &RouterChain{
		builtinRouters: builtinRouters,
		routers:        routers,
	}
}

func (r RouterChain) AddRouters(routers []cluster.Router) {
	r.routers = append(r.routers, routers...)
	sort.SliceStable(r.routers, func(i, j int) bool {
		return routers[i].Priority() < routers[j].Priority()
	})
}

func (r RouterChain) SetInvokers(invokers []protocol.Invoker) {
	r.Invokers = invokers
	/*for _, _ := range r.routers {
		//router.Notify(r.Invokers)
	}*/
}
