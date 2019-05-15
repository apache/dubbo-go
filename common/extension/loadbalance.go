package extension

import "github.com/dubbo/go-for-apache-dubbo/cluster"

var (
	loadbalances = make(map[string]func() cluster.LoadBalance)
)

func SetLoadbalance(name string, fcn func() cluster.LoadBalance) {
	loadbalances[name] = fcn
}

func GetLoadbalance(name string) cluster.LoadBalance {
	if loadbalances[name] == nil {
		panic("loadbalance for " + name + " is not existing, you must import corresponding package.")
	}
	return loadbalances[name]()
}
