package p2c

import (
	"dubbo.apache.org/dubbo-go/v3/cluster/loadbalance"
	"dubbo.apache.org/dubbo-go/v3/cluster/metrics"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/common/extension"
	"dubbo.apache.org/dubbo-go/v3/protocol"
)

func init() {
	extension.SetLoadbalance(constant.LoadBalanceKeyP2C, newLoadBalance)
}

type loadBalance struct {
	
}

func newLoadBalance() loadbalance.LoadBalance {
	return &loadBalance{}
}

func (l *loadBalance) Select(invokers []protocol.Invoker, invocation protocol.Invocation) protocol.Invoker {
	// m is the Metrics, which saves the metrics of instance, invokers and methods
	// The local metrics is available only for the earlier version.
	m := metrics.LocalMetrics
	panic("implement me")
}

