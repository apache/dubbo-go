package loadbalance

import (
	"github.com/dubbo/dubbo-go/cluster"
	"github.com/dubbo/dubbo-go/common/extension"
	"github.com/dubbo/dubbo-go/config"
	"github.com/dubbo/dubbo-go/protocol"
	"math/rand"
)

const name = "random"

func init() {
	extension.SetLoadbalance(name, NewRandomLoadBalance)
}

type RandomLoadBalance struct {
}

func NewRandomLoadBalance() cluster.LoadBalance {
	return &RandomLoadBalance{}
}

func (lb *RandomLoadBalance) Select(invokers []protocol.Invoker, url config.URL, invocation protocol.Invocation) protocol.Invoker {
	var length int
	if length = len(invokers); length == 1 {
		return invokers[0]
	}
	sameWeight := true
	weights := make([]int64, length)

	firstWeight := cluster.GetWeight(invokers[0], invocation)
	totalWeight := firstWeight
	weights[0] = firstWeight

	for i := 1; i < length; i++ {
		weight := cluster.GetWeight(invokers[i], invocation)
		weights[i] = weight

		totalWeight += weight
		if sameWeight && weight != firstWeight {
			sameWeight = false
		}
	}

	if totalWeight > 0 && !sameWeight {
		// If (not every invoker has the same weight & at least one invoker's weight>0), select randomly based on totalWeight.
		offset := rand.Int63n(totalWeight)

		for i := 0; i < length; i++ {
			offset -= weights[i]
			if offset < 0 {
				return invokers[i]
			}
		}
	}
	// If all invokers have the same weight value or totalWeight=0, return evenly.
	return invokers[rand.Intn(length)]
}
