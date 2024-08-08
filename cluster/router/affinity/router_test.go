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

package affinity

import (
	"testing"
)

import (
	"github.com/stretchr/testify/assert"
)

import (
	"dubbo.apache.org/dubbo-go/v3/cluster/router/condition"
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/config_center"
	"dubbo.apache.org/dubbo-go/v3/protocol"
	"dubbo.apache.org/dubbo-go/v3/protocol/invocation"
	"dubbo.apache.org/dubbo-go/v3/remoting"
)

var providerUrls = []string{
	"dubbo://127.0.0.1/com.foo.BarService",
	"dubbo://127.0.0.1/com.foo.BarService",
	"dubbo://127.0.0.1/com.foo.BarService?env=normal",
	"dubbo://127.0.0.1/com.foo.BarService?env=normal",
	"dubbo://127.0.0.1/com.foo.BarService?env=normal",
	"dubbo://127.0.0.1/com.foo.BarService?region=beijing",
	"dubbo://127.0.0.1/com.foo.BarService?region=beijing",
	"dubbo://127.0.0.1/com.foo.BarService?region=beijing",
	"dubbo://127.0.0.1/com.foo.BarService?region=beijing&env=gray",
	"dubbo://127.0.0.1/com.foo.BarService?region=beijing&env=gray",
	"dubbo://127.0.0.1/com.foo.BarService?region=beijing&env=gray",
	"dubbo://127.0.0.1/com.foo.BarService?region=beijing&env=gray",
	"dubbo://127.0.0.1/com.foo.BarService?region=beijing&env=normal",
	"dubbo://127.0.0.1/com.foo.BarService?region=hangzhou",
	"dubbo://127.0.0.1/com.foo.BarService?region=hangzhou",
	"dubbo://127.0.0.1/com.foo.BarService?region=hangzhou&env=gray",
	"dubbo://127.0.0.1/com.foo.BarService?region=hangzhou&env=gray",
	"dubbo://127.0.0.1/com.foo.BarService?region=hangzhou&env=normal",
	"dubbo://127.0.0.1/com.foo.BarService?region=hangzhou&env=normal",
	"dubbo://127.0.0.1/com.foo.BarService?region=hangzhou&env=normal",
	"dubbo://dubbo.apache.org/com.foo.BarService",
	"dubbo://dubbo.apache.org/com.foo.BarService",
	"dubbo://dubbo.apache.org/com.foo.BarService?env=normal",
	"dubbo://dubbo.apache.org/com.foo.BarService?env=normal",
	"dubbo://dubbo.apache.org/com.foo.BarService?env=normal",
	"dubbo://dubbo.apache.org/com.foo.BarService?region=beijing",
	"dubbo://dubbo.apache.org/com.foo.BarService?region=beijing",
	"dubbo://dubbo.apache.org/com.foo.BarService?region=beijing",
	"dubbo://dubbo.apache.org/com.foo.BarService?region=beijing&env=gray",
	"dubbo://dubbo.apache.org/com.foo.BarService?region=beijing&env=gray",
	"dubbo://dubbo.apache.org/com.foo.BarService?region=beijing&env=gray",
	"dubbo://dubbo.apache.org/com.foo.BarService?region=beijing&env=gray",
	"dubbo://dubbo.apache.org/com.foo.BarService?region=beijing&env=normal",
	"dubbo://dubbo.apache.org/com.foo.BarService?region=hangzhou",
	"dubbo://dubbo.apache.org/com.foo.BarService?region=hangzhou",
	"dubbo://dubbo.apache.org/com.foo.BarService?region=hangzhou&env=gray",
	"dubbo://dubbo.apache.org/com.foo.BarService?region=hangzhou&env=gray",
	"dubbo://dubbo.apache.org/com.foo.BarService?region=hangzhou&env=normal",
	"dubbo://dubbo.apache.org/com.foo.BarService?region=hangzhou&env=normal",
	"dubbo://dubbo.apache.org/com.foo.BarService?region=hangzhou&env=normal",
}

func buildInvokers() []protocol.Invoker {
	res := make([]protocol.Invoker, 0, len(providerUrls))
	for _, url := range providerUrls {
		u, err := common.NewURL(url)
		if err != nil {
			panic(err)
		}
		res = append(res, protocol.NewBaseInvoker(u))
	}
	return res
}

func newUrl(s string) *common.URL {
	res, err := common.NewURL(s)
	if err != nil {
		panic(err)
	}
	return res
}

func gen_matcher(key string) *condition.FieldMatcher {
	res, err := condition.NewFieldMatcher(key)
	if err != nil {
		panic(err)
	}
	return &res
}

type InvokersFilters []condition.FieldMatcher

func NewINVOKERS_FILTERS() InvokersFilters {
	return []condition.FieldMatcher{}
}

func (INV InvokersFilters) add(rule string) InvokersFilters {
	m := gen_matcher(rule)
	return append(INV, *m)
}

func (INV InvokersFilters) filtrate(inv []protocol.Invoker, url *common.URL, invocation protocol.Invocation) []protocol.Invoker {
	for _, cond := range INV {
		tmpInv := make([]protocol.Invoker, 0)
		for _, invoker := range inv {
			if cond.MatchInvoker(url, invoker, invocation) {
				tmpInv = append(tmpInv, invoker)
			}
		}
		inv = tmpInv
	}
	return inv
}

func Test_affinityRoute_Route(t *testing.T) {
	type fields struct {
		content string
	}
	type args struct {
		invokers   []protocol.Invoker
		url        *common.URL
		invocation protocol.Invocation
	}
	tests := []struct {
		name             string
		fields           fields
		args             args
		invokers_filters InvokersFilters
		expectLen        int
	}{
		{
			name: "test base affinity router",
			fields: fields{`configVersion: v3.1
scope: service # Or application
key: service.apache.com
enabled: true
runtime: true
affinityAware:
  key: region
  ratio: 20`},
			args: args{
				invokers:   buildInvokers(),
				url:        newUrl("consumer://127.0.0.1/com.foo.BarService?env=gray&region=beijing"),
				invocation: invocation.NewRPCInvocation("getComment", nil, nil),
			},
			invokers_filters: NewINVOKERS_FILTERS().add("region=$region"),
		}, {
			name: "test bad ratio",
			fields: fields{`configVersion: v3.1
scope: service # Or application
key: service.apache.com
enabled: true
runtime: true
affinityAware:
  key: region
  ratio: 101`},
			args: args{
				invokers:   buildInvokers(),
				url:        newUrl("consumer://127.0.0.1/com.foo.BarService?env=gray&region=beijing"),
				invocation: invocation.NewRPCInvocation("getComment", nil, nil),
			},
			invokers_filters: NewINVOKERS_FILTERS(),
		}, {
			name: "test ratio false",
			fields: fields{`configVersion: v3.1
scope: service # Or application
key: service.apache.com
enabled: true
runtime: true
affinityAware:
  key: region
  ratio: 80`},
			args: args{
				invokers:   buildInvokers(),
				url:        newUrl("consumer://127.0.0.1/com.foo.BarService?env=gray&region=beijing"),
				invocation: invocation.NewRPCInvocation("getComment", nil, nil),
			},
			invokers_filters: NewINVOKERS_FILTERS(),
		}, {
			name: "test ignore affinity route",
			fields: fields{`configVersion: v3.1
scope: service # Or application
key: service.apache.com
enabled: true
runtime: true
affinityAware:
  key: bad-key
  ratio: 80`},
			args: args{
				invokers:   buildInvokers(),
				url:        newUrl("consumer://127.0.0.1/com.foo.BarService?env=gray&region=beijing"),
				invocation: invocation.NewRPCInvocation("getComment", nil, nil),
			},
			invokers_filters: NewINVOKERS_FILTERS(),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := &affinityRoute{}
			a.Process(&config_center.ConfigChangeEvent{
				Value:      tt.fields.content,
				ConfigType: remoting.EventTypeUpdate,
			})
			res := a.Route(tt.args.invokers, tt.args.url, tt.args.invocation)
			if tt.invokers_filters != nil {
				// check expect filtrate path
				ans := tt.invokers_filters.filtrate(tt.args.invokers, tt.args.url, tt.args.invocation)
				if len(ans) < int(float32(len(providerUrls))*(float32(a.ratio)/100.)) {
					assert.Equalf(t, 0, len(ans), "route(%v, %v, %v)", tt.args.invokers, tt.args.url, tt.args.invocation)
				} else {
					assert.Equalf(t, ans, res, "route(%v, %v, %v)", tt.args.invokers, tt.args.url, tt.args.invocation)
				}
			} else {
				ans := tt.invokers_filters.filtrate(tt.args.invokers, tt.args.url, tt.args.invocation)
				assert.Equalf(t, tt.expectLen, len(ans), "route(%v, %v, %v)", tt.args.invokers, tt.args.url, tt.args.invocation)
			}
		})
	}

}
