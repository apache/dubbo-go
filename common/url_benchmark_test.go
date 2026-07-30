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

package common

import (
	"fmt"
	"net/url"
	"testing"
)

import (
	gxset "github.com/dubbogo/gost/container/set"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common/constant"
)

const (
	benchmarkURLProtocol  = "dubbo"
	benchmarkURLIP        = "127.0.0.1"
	benchmarkURLPort      = "20000"
	benchmarkURLInterface = "com.test.BenchmarkService"
	benchmarkURLGroup     = "benchmark"
	benchmarkURLVersion   = "1.0.0"
	benchmarkURLTimestamp = "1234567890"
	benchmarkURLMeshID    = "benchmark-mesh"
)

var (
	benchmarkURLParamSizes = []int{1, 32, 256, 1024}

	benchmarkURLStringSink string
	benchmarkURLSink       *URL
	benchmarkURLParamsSink url.Values
	benchmarkURLMapSink    map[string]string
	benchmarkURLBoolSink   bool
)

func BenchmarkURLString(b *testing.B) {
	runBenchmarkURLByParamSize(b, func(u *URL) {
		benchmarkURLStringSink = u.String()
	})
}

func BenchmarkURLKey(b *testing.B) {
	runBenchmarkURLByParamSize(b, func(u *URL) {
		benchmarkURLStringSink = u.Key()
	})
}

func BenchmarkURLGetCacheInvokerMapKey(b *testing.B) {
	runBenchmarkURLByParamSize(b, func(u *URL) {
		benchmarkURLStringSink = u.GetCacheInvokerMapKey()
	})
}

func BenchmarkURLServiceKey(b *testing.B) {
	runBenchmarkURLByParamSize(b, func(u *URL) {
		benchmarkURLStringSink = u.ServiceKey()
	})
}

func BenchmarkURLCopyParams(b *testing.B) {
	runBenchmarkURLByParamSize(b, func(u *URL) {
		benchmarkURLParamsSink = u.CopyParams()
	})
}

func BenchmarkURLGetParams(b *testing.B) {
	runBenchmarkURLByParamSize(b, func(u *URL) {
		benchmarkURLParamsSink = u.GetParams()
	})
}

func BenchmarkURLClone(b *testing.B) {
	runBenchmarkURLByParamSize(b, func(u *URL) {
		benchmarkURLSink = u.Clone()
	})

	for _, paramCount := range benchmarkURLParamSizes {
		b.Run(fmt.Sprintf("params_%d_with_suburl", paramCount), func(b *testing.B) {
			u := makeBenchmarkURLWithSubURL(paramCount)
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				benchmarkURLSink = u.Clone()
			}
		})
	}
}

func BenchmarkURLCloneWithFilter(b *testing.B) {
	runBenchmarkURLByParamSize(b, func(u *URL) {
		benchmarkURLSink = u.CloneWithFilter(nil, nil)
	})

	for _, paramCount := range benchmarkURLParamSizes {
		b.Run(fmt.Sprintf("params_%d_exclude_20_percent", paramCount), func(b *testing.B) {
			u := makeBenchmarkURL(paramCount)
			excludeParams := makeBenchmarkExcludeSet(paramCount)
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				benchmarkURLSink = u.CloneWithFilter(excludeParams, nil)
			}
		})
	}

	for _, paramCount := range benchmarkURLParamSizes {
		b.Run(fmt.Sprintf("params_%d_exclude_20_percent_with_suburl", paramCount), func(b *testing.B) {
			u := makeBenchmarkURLWithSubURL(paramCount)
			excludeParams := makeBenchmarkExcludeSet(paramCount)
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				benchmarkURLSink = u.CloneWithFilter(excludeParams, nil)
			}
		})
	}

	for _, paramCount := range benchmarkURLParamSizes {
		b.Run(fmt.Sprintf("params_%d_reserve_20_percent", paramCount), func(b *testing.B) {
			u := makeBenchmarkURL(paramCount)
			reserveParams := makeBenchmarkReserveKeys(paramCount)
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				benchmarkURLSink = u.CloneWithFilter(nil, reserveParams)
			}
		})
	}

	for _, paramCount := range benchmarkURLParamSizes {
		b.Run(fmt.Sprintf("params_%d_reserve_20_percent_with_suburl", paramCount), func(b *testing.B) {
			u := makeBenchmarkURLWithSubURL(paramCount)
			reserveParams := makeBenchmarkReserveKeys(paramCount)
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				benchmarkURLSink = u.CloneWithFilter(nil, reserveParams)
			}
		})
	}
}

func BenchmarkURLMergeURL(b *testing.B) {
	for _, paramCount := range benchmarkURLParamSizes {
		b.Run(fmt.Sprintf("params_%d_with_method_params", paramCount), func(b *testing.B) {
			left, right := makeBenchmarkMergeHalfOverlapPair(paramCount)
			addBenchmarkMethodParams(right)
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				benchmarkURLSink = left.MergeURL(right)
			}
		})
	}
}

func BenchmarkURLToMap(b *testing.B) {
	runBenchmarkURLByParamSize(b, func(u *URL) {
		benchmarkURLMapSink = u.ToMap()
	})
}

func BenchmarkURLIsEquals(b *testing.B) {
	for _, paramCount := range benchmarkURLParamSizes {
		b.Run(fmt.Sprintf("params_%d", paramCount), func(b *testing.B) {
			left := makeBenchmarkURL(paramCount)
			right := makeBenchmarkURL(paramCount)
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				benchmarkURLBoolSink = IsEquals(left, right)
			}
		})
	}
}

func runBenchmarkURLByParamSize(b *testing.B, run func(*URL)) {
	b.Helper()
	for _, paramCount := range benchmarkURLParamSizes {
		b.Run(fmt.Sprintf("params_%d", paramCount), func(b *testing.B) {
			u := makeBenchmarkURL(paramCount)
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				run(u)
			}
		})
	}
}

func makeBenchmarkURL(paramCount int) *URL {
	return makeBenchmarkURLWithParamStart(paramCount, 0)
}

func makeBenchmarkURLWithSubURL(paramCount int) *URL {
	u := makeBenchmarkURL(paramCount)
	u.SubURL = makeBenchmarkURL(paramCount)
	return u
}

func makeBenchmarkURLWithParamStart(paramCount, genericStart int) *URL {
	u := NewURLWithOptions(
		WithProtocol(benchmarkURLProtocol),
		WithIp(benchmarkURLIP),
		WithPort(benchmarkURLPort),
		WithPath(benchmarkURLInterface),
		WithMethods(makeBenchmarkMethods()),
		WithParams(makeBenchmarkParams(paramCount, genericStart)),
	)
	u.primitiveTS = u.GetParam(constant.TimestampKey, "")
	return u
}

func makeBenchmarkMergeHalfOverlapPair(paramCount int) (*URL, *URL) {
	return makeBenchmarkURL(paramCount), makeBenchmarkURLWithParamStart(paramCount, paramCount/2)
}

func makeBenchmarkParams(paramCount, genericStart int) url.Values {
	params := make(url.Values, len(benchmarkURLIdentityParams)+paramCount)
	for _, param := range benchmarkURLIdentityParams {
		params.Set(param.key, param.value)
	}
	for i := range paramCount {
		genericIndex := genericStart + i
		params.Set(fmt.Sprintf("key%d", genericIndex), fmt.Sprintf("value%d", genericIndex))
	}
	return params
}

func makeBenchmarkParamKeys(paramCount int) []string {
	keys := make([]string, 0, len(benchmarkURLIdentityParams)+paramCount)
	for _, param := range benchmarkURLIdentityParams {
		keys = append(keys, param.key)
	}
	for i := range paramCount {
		keys = append(keys, fmt.Sprintf("key%d", i))
	}
	return keys
}

func makeBenchmarkExcludeSet(paramCount int) *gxset.HashSet {
	keys := makeBenchmarkFilterKeys(paramCount)
	values := make([]any, 0, len(keys))
	for _, key := range keys {
		values = append(values, key)
	}
	return gxset.NewSet(values...)
}

func makeBenchmarkReserveKeys(paramCount int) []string {
	return makeBenchmarkFilterKeys(paramCount)
}

func makeBenchmarkFilterKeys(paramCount int) []string {
	keys := makeBenchmarkParamKeys(paramCount)
	filterCount := min(max(len(keys)/5, 1), len(keys))
	return keys[:filterCount]
}

func makeBenchmarkMethods() []string {
	return []string{"method0", "method1", "method2"}
}

func addBenchmarkMethodParams(u *URL) {
	u.Methods = makeBenchmarkMethods()
	u.SetParam(constant.LoadbalanceKey, "random")
	u.SetParam(constant.ClusterKey, "failover")
	u.SetParam(constant.RetriesKey, "2")
	u.SetParam(constant.TimeoutKey, "3000")
	for _, method := range u.Methods {
		u.SetParam("methods."+method+"."+constant.LoadbalanceKey, "roundrobin")
		u.SetParam("methods."+method+"."+constant.ClusterKey, "failfast")
		u.SetParam("methods."+method+"."+constant.RetriesKey, "1")
		u.SetParam("methods."+method+"."+constant.TimeoutKey, "1000")
	}
}

var benchmarkURLIdentityParams = []struct {
	key   string
	value string
}{
	{constant.InterfaceKey, benchmarkURLInterface},
	{constant.GroupKey, benchmarkURLGroup},
	{constant.VersionKey, benchmarkURLVersion},
	{constant.TimestampKey, benchmarkURLTimestamp},
	{constant.MeshClusterIDKey, benchmarkURLMeshID},
}
