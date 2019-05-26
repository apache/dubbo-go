package loadbalance

import (
    "github.com/dubbo/go-for-apache-dubbo/cluster"
    "github.com/dubbo/go-for-apache-dubbo/common/extension"
    "github.com/dubbo/go-for-apache-dubbo/protocol"
    "math"
    "sync"
    "sync/atomic"
    "time"
)

const (
    roundRobin = "roundrobin"

    complete = 0
    updating = 1
)

var (
    methodWeightMap sync.Map         // [string]invokers
    state           int32 = complete // update lock acquired ?
)

func init() {
    extension.SetLoadbalance(roundRobin, NewRandomLoadBalance)
}

type roundRobinLoadBalance struct {
}

func NewRoundRobinLoadBalance() cluster.LoadBalance {
    return &roundRobinLoadBalance{}
}

func (lb *roundRobinLoadBalance) Select(invokers []protocol.Invoker, invocation protocol.Invocation) protocol.Invoker {

    count := len(invokers)
    if invokers == nil || count == 0 {
        return nil
    }
    if count == 1 {
        return invokers[0]
    }

    key := invokers[0].GetUrl().Path + "." + invocation.MethodName()
    cache, _ := methodWeightMap.LoadOrStore(key, cachedInvokers{})
    cachedInvokers := cache.(cachedInvokers)

    clean := false
    totalWeight := int64(0)
    maxCurrentWeight := int64(math.MinInt64)
    var selectedInvoker protocol.Invoker
    var selectedWeightRobin weightedRoundRobin
    now := time.Now()

    for _, invoker := range invokers {

        var weight = GetWeight(invoker, invocation)
        if weight < 0 {
            weight = 0
        }

        identify := invoker.GetUrl().Key()
        loaded, found := cachedInvokers.LoadOrStore(identify, weightedRoundRobin{weight: weight})
        weightRobin := loaded.(weightedRoundRobin)
        if !found {
            clean = true
        }

        if weightRobin.Weight() != weight {
            weightRobin.setWeight(weight)
        }

        currentWeight := weightRobin.increaseCurrent()
        weightRobin.lastUpdate = &now

        if currentWeight > maxCurrentWeight {
            maxCurrentWeight = currentWeight
            selectedInvoker = invoker
            selectedWeightRobin = weightRobin
        }
        totalWeight += weight
    }

    cleanIfRequired(clean, cachedInvokers)

    if selectedInvoker != nil {
        selectedWeightRobin.Current(totalWeight)
        return selectedInvoker
    }

    return invokers[0]
}

func cleanIfRequired(clean bool, invokers cachedInvokers) {
    if atomic.LoadInt32(&state) < updating && clean && atomic.CompareAndSwapInt32(&state, complete, updating) {
        defer atomic.CompareAndSwapInt32(&state, updating, complete)
        invokers.Range(func(identify, weightedRoundRobin interface{}) bool {
            
            return true
        })
    }
}

// Record the weight of the invoker
type weightedRoundRobin struct {
    weight     int64
    current    int64
    lastUpdate *time.Time
}

func (robin *weightedRoundRobin) Weight() int64 {
    return atomic.LoadInt64(&robin.weight)
}

func (robin *weightedRoundRobin) setWeight(weight int64) {
    robin.weight = weight
    robin.current = 0
}

func (robin *weightedRoundRobin) increaseCurrent() int64 {
    return atomic.AddInt64(&robin.current, robin.weight)
}

func (robin *weightedRoundRobin) Current(delta int64) {
    atomic.AddInt64(&robin.current, -1*delta)
}

//type methodWeightCache struct {
//    mu           sync.RWMutex
//    methodWeight map[string]cachedInvokers
//}
//
//type invokerCache struct {
//    mu           sync.RWMutex
//    invokers map[string]weightedRoundRobin
//}

type cachedInvokers struct {
    sync.Map /*[string]weightedRoundRobin*/
}
