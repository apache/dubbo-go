/*
 * Licensed to the Apache Software Foundation (ASF) under one or more
 * contributor license agreements.  See the NOTICE file distributed with
 * this work for additional information regarding copyright ownership.
 * The ASF licenses this file to You under the Apache License, Version 2.0
 * (the "License"); you may not use this file except in compliance with
 * the License.  You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package tag

import (
	"fmt"
	"testing"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/global"
	"dubbo.apache.org/dubbo-go/v3/protocol/base"
	"dubbo.apache.org/dubbo-go/v3/protocol/invocation"
)

// makeBenchInvokers creates n invokers where the first tagged ones carry tag=gray.
// All invokers get the given app as ApplicationKey.
func makeBenchInvokers(n, tagged int, app string) []base.Invoker {
	invokers := make([]base.Invoker, n)
	for i := range n {
		u, _ := common.NewURL(fmt.Sprintf("dubbo://192.168.0.%d:20000/com.xxx.xxx.UserProvider?interface=com.xxx.xxx.UserProvider&group=&version=3.1.0", i+1))
		u.SetParam(constant.ApplicationKey, app)
		if i < tagged {
			u.SetParam(constant.Tagkey, "gray")
		}
		invokers[i] = base.NewBaseInvoker(u)
	}
	return invokers
}

func newBenchConsumerURL(app string) *common.URL {
	u, _ := common.NewURL(fmt.Sprintf("consumer://127.0.0.1:20000/com.xxx.xxx.UserProvider?interface=com.xxx.xxx.UserProvider&group=&version=3.1.0&%s=%s", constant.ApplicationKey, app))
	return u
}

// BenchmarkRouteStaticTag compares cached vs no-cache paths for static tag routing.
// The tag is embedded in invoker URLs; the request carries tag=gray.
//
// cached:    pool["tag\x00"+tag] single map lookup + bitmap iteration
// no_cache:  filterInvokers linear scan with GetURL().GetParam per invoker
func BenchmarkRouteStaticTag(b *testing.B) {
	b.ReportAllocs()
	for _, n := range []int{10, 100, 1000} {
		tagged := n * 30 / 100
		invokers := makeBenchInvokers(n, tagged, "bench-static")
		url := newBenchConsumerURL("bench-static")
		invoc := invocation.NewRPCInvocation("GetUser", nil, map[string]any{constant.Tagkey: "gray"})

		b.Run(fmt.Sprintf("no_cache/N=%d", n), func(b *testing.B) {
			p, _ := NewTagPriorityRouter()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				p.Route(invokers, url, invoc)
			}
		})
	}
	for _, n := range []int{10, 100, 1000} {
		tagged := n * 30 / 100
		invokers := makeBenchInvokers(n, tagged, "bench-static")
		url := newBenchConsumerURL("bench-static")
		invoc := invocation.NewRPCInvocation("GetUser", nil, map[string]any{constant.Tagkey: "gray"})

		b.Run(fmt.Sprintf("cached/N=%d", n), func(b *testing.B) {
			p, _ := NewTagPriorityRouter()
			withCache(p, invokers)
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				p.Route(invokers, url, invoc)
			}
		})
	}
}

// BenchmarkRouteDynamicTagAddr compares cached vs no-cache paths for dynamic tag
// routing with an address list. The request carries tag=gray; the dynamic config
// specifies 3 addresses for the gray tag.
//
// cached:    requestTagMatchBM -> addressesBM map lookups + bitmap OR
// no_cache:  requestTag -> filterInvokers + getAddressPredicate O(N*A) nested loop
func BenchmarkRouteDynamicTagAddr(b *testing.B) {
	b.ReportAllocs()
	for _, n := range []int{10, 100, 1000} {
		tagged := n * 30 / 100
		app := "bench-dynaddr"
		invokers := makeBenchInvokers(n, tagged, app)
		url := newBenchConsumerURL(app)
		invoc := invocation.NewRPCInvocation("GetUser", nil, map[string]any{constant.Tagkey: "gray"})
		cfgKey := app + constant.TagRouterRuleSuffix
		cfg := global.RouterConfig{
			Key:     cfgKey,
			Force:   boolPtr(false),
			Enabled: boolPtr(true),
			Valid:   boolPtr(true),
			Tags: []global.Tag{{
				Name:      "gray",
				Addresses: []string{"192.168.0.1:20000", "192.168.0.2:20000", "192.168.0.3:20000"},
			}},
		}

		b.Run(fmt.Sprintf("no_cache/N=%d", n), func(b *testing.B) {
			p, _ := NewTagPriorityRouter()
			p.routerConfigs.Store(cfgKey, cfg)
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				p.Route(invokers, url, invoc)
			}
		})
	}
	for _, n := range []int{10, 100, 1000} {
		tagged := n * 30 / 100
		app := "bench-dynaddr"
		invokers := makeBenchInvokers(n, tagged, app)
		url := newBenchConsumerURL(app)
		invoc := invocation.NewRPCInvocation("GetUser", nil, map[string]any{constant.Tagkey: "gray"})
		cfgKey := app + constant.TagRouterRuleSuffix
		cfg := global.RouterConfig{
			Key:     cfgKey,
			Force:   boolPtr(false),
			Enabled: boolPtr(true),
			Valid:   boolPtr(true),
			Tags: []global.Tag{{
				Name:      "gray",
				Addresses: []string{"192.168.0.1:20000", "192.168.0.2:20000", "192.168.0.3:20000"},
			}},
		}

		b.Run(fmt.Sprintf("cached/N=%d", n), func(b *testing.B) {
			p, _ := NewTagPriorityRouter()
			withCache(p, invokers)
			p.routerConfigs.Store(cfgKey, cfg)
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				p.Route(invokers, url, invoc)
			}
		})
	}
}

// BenchmarkRouteCacheHitBenefit measures cache benefit at fixed N=100 with varying
// tag density (10/50/90 tagged invokers). The dynamic config specifies 3 addresses
// for the gray tag; the request carries tag=gray.
//
// At low tagged ratio the no-cache path scans all N but finds few matches (O(N));
// the cached path directly locates a small bitmap (O(result_size)).
// At high tagged ratio the result set is large and the gap narrows.
func BenchmarkRouteCacheHitBenefit(b *testing.B) {
	b.ReportAllocs()
	n := 100
	for _, tagged := range []int{10, 50, 90} {
		app := "bench-hitbenefit"
		invokers := makeBenchInvokers(n, tagged, app)
		url := newBenchConsumerURL(app)
		invoc := invocation.NewRPCInvocation("GetUser", nil, map[string]any{constant.Tagkey: "gray"})
		cfgKey := app + constant.TagRouterRuleSuffix
		cfg := global.RouterConfig{
			Key:     cfgKey,
			Force:   boolPtr(false),
			Enabled: boolPtr(true),
			Valid:   boolPtr(true),
			Tags: []global.Tag{{
				Name:      "gray",
				Addresses: []string{"192.168.0.1:20000", "192.168.0.2:20000", "192.168.0.3:20000"},
			}},
		}

		b.Run(fmt.Sprintf("no_cache/tagged=%d", tagged), func(b *testing.B) {
			p, _ := NewTagPriorityRouter()
			p.routerConfigs.Store(cfgKey, cfg)
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				p.Route(invokers, url, invoc)
			}
		})
	}
	for _, tagged := range []int{10, 50, 90} {
		app := "bench-hitbenefit"
		invokers := makeBenchInvokers(n, tagged, app)
		url := newBenchConsumerURL(app)
		invoc := invocation.NewRPCInvocation("GetUser", nil, map[string]any{constant.Tagkey: "gray"})
		cfgKey := app + constant.TagRouterRuleSuffix
		cfg := global.RouterConfig{
			Key:     cfgKey,
			Force:   boolPtr(false),
			Enabled: boolPtr(true),
			Valid:   boolPtr(true),
			Tags: []global.Tag{{
				Name:      "gray",
				Addresses: []string{"192.168.0.1:20000", "192.168.0.2:20000", "192.168.0.3:20000"},
			}},
		}

		b.Run(fmt.Sprintf("cached/tagged=%d", tagged), func(b *testing.B) {
			p, _ := NewTagPriorityRouter()
			withCache(p, invokers)
			p.routerConfigs.Store(cfgKey, cfg)
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				p.Route(invokers, url, invoc)
			}
		})
	}
}
