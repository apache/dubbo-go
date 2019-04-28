package cluster

import (
	"github.com/dubbo/dubbo-go/common/constant"
	"github.com/dubbo/dubbo-go/config"
	"github.com/dubbo/dubbo-go/protocol"
	"time"
)

// Extension - LoadBalance
type LoadBalance interface {
	Select([]protocol.Invoker, config.URL, protocol.Invocation) protocol.Invoker
}

func GetWeight(invoker protocol.Invoker, invocation protocol.Invocation) int64 {
	url := invoker.GetUrl().(*config.URL)
	weight := url.GetMethodParamInt(invocation.MethodName(), constant.WEIGHT_KEY, constant.DEFAULT_WEIGHT)
	if weight > 0 {
		//get service register time an do warm up time
		now := time.Now().Unix()
		timestamp := url.GetParamInt(constant.REMOTE_TIMESTAMP_KEY, now)
		if uptime := now - timestamp; uptime > 0 {
			warmup := url.GetParamInt(constant.WARMUP_KEY, constant.DEFAULT_WARMUP)
			if uptime < warmup {
				if ww := float64(uptime) / float64(warmup) / float64(weight); ww < 1 {
					weight = 1
				} else if int64(ww) <= weight {
					weight = int64(ww)
				}
			}
		}
	}
	return weight
}
