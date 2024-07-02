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

package loadbalance_test

import (
	"fmt"
	"testing"
)

import (
	"dubbo.apache.org/dubbo-go/v3/cluster/loadbalance"
	_ "dubbo.apache.org/dubbo-go/v3/cluster/loadbalance/aliasmethod"
	_ "dubbo.apache.org/dubbo-go/v3/cluster/loadbalance/consistenthashing"
	_ "dubbo.apache.org/dubbo-go/v3/cluster/loadbalance/iwrr"
	_ "dubbo.apache.org/dubbo-go/v3/cluster/loadbalance/leastactive"
	_ "dubbo.apache.org/dubbo-go/v3/cluster/loadbalance/p2c"
	_ "dubbo.apache.org/dubbo-go/v3/cluster/loadbalance/random"
	_ "dubbo.apache.org/dubbo-go/v3/cluster/loadbalance/ringhash"
	_ "dubbo.apache.org/dubbo-go/v3/cluster/loadbalance/roundrobin"
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/common/extension"
	"dubbo.apache.org/dubbo-go/v3/protocol"
	"dubbo.apache.org/dubbo-go/v3/protocol/invocation"
)

func Generate() []protocol.Invoker {
	var invokers []protocol.Invoker
	for i := 1; i < 256; i++ {
		url, _ := common.NewURL(fmt.Sprintf("dubbo://192.168.1.%v:20000/org.apache.demo.HelloService", i))
		invokers = append(invokers, protocol.NewBaseInvoker(url))
	}
	return invokers
}

func Benchloadbalance(b *testing.B, lb loadbalance.LoadBalance) {
	b.Helper()
	invokers := Generate()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		lb.Select(invokers, &invocation.RPCInvocation{})
	}
}

func BenchmarkRoudrobinLoadbalance(b *testing.B) {
	Benchloadbalance(b, extension.GetLoadbalance(constant.LoadBalanceKeyRoundRobin))
}

func BenchmarkLeastativeLoadbalance(b *testing.B) {
	Benchloadbalance(b, extension.GetLoadbalance(constant.LoadBalanceKeyLeastActive))
}

func BenchmarkConsistenthashingLoadbalance(b *testing.B) {
	Benchloadbalance(b, extension.GetLoadbalance(constant.LoadBalanceKeyConsistentHashing))
}

func BenchmarkP2CLoadbalance(b *testing.B) {
	Benchloadbalance(b, extension.GetLoadbalance(constant.LoadBalanceKeyP2C))
}

func BenchmarkInterleavedWeightedRoundRobinLoadbalance(b *testing.B) {
	Benchloadbalance(b, extension.GetLoadbalance(constant.LoadBalanceKeyInterleavedWeightedRoundRobin))
}

func BenchmarkRandomLoadbalance(b *testing.B) {
	Benchloadbalance(b, extension.GetLoadbalance(constant.LoadBalanceKeyRandom))
}

func BenchmarkAliasMethodLoadbalance(b *testing.B) {
	Benchloadbalance(b, extension.GetLoadbalance(constant.LoadBalanceKeyAliasMethod))
}
