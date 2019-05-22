package extension

import "github.com/dubbo/go-for-apache-dubbo/common/proxy"

var (
	proxy_factories = make(map[string]func(...proxy.Option) proxy.ProxyFactory)
)

func SetProxyFactory(name string, f func(...proxy.Option) proxy.ProxyFactory) {
	proxy_factories[name] = f
}
func GetProxyFactory(name string) proxy.ProxyFactory {
	if name == "" {
		name = "default"
	}
	if proxy_factories[name] == nil {
		panic("proxy factory for " + name + " is not existing, make sure you have import the package.")
	}
	return proxy_factories[name]()
}
