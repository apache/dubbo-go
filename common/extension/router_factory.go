package extension

import (
	"github.com/apache/dubbo-go/cluster"
)

var (
	routers = make(map[string]func() cluster.RouterFactory)
)

func SetRouterFactory(name string, fun func() cluster.RouterFactory) {
	routers[name] = fun
}

func GetRouterFactory(name string) cluster.RouterFactory {
	if routers[name] == nil {
		panic("router_factory for " + name + " is not existing, make sure you have import the package.")
	}
	return routers[name]()

}
