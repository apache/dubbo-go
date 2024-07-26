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

package script

import (
	"context"
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/config_center"
	"dubbo.apache.org/dubbo-go/v3/protocol"
	"dubbo.apache.org/dubbo-go/v3/protocol/invocation"
	"dubbo.apache.org/dubbo-go/v3/remoting"
	"github.com/stretchr/testify/assert"
	"testing"
)

var url1 = func() *common.URL {
	i, _ := common.NewURL("dubbo://127.0.0.1:20000/com.ikurento.user.UserProvider?anyhost=true&" +
		"application=BDTService&category=providers&default.timeout=10000&dubbo=dubbo-provider-golang-1.0.0&" +
		"environment=dev&interface=com.ikurento.user.UserProvider&ip=192.168.56.1&methods=GetUser%2C&" +
		"module=dubbogo+user-info+server&org=ikurento.com&owner=ZX&pid=1447&revision=0.0.1&" +
		"side=provider&timeout=3000&timestamp=1556509797245")
	return i
}
var url2 = func() *common.URL {
	u, _ := common.NewURL("dubbo://127.0.0.1:20001/com.ikurento.user.UserProvider?anyhost=true&" +
		"application=BDTService&category=providers&default.timeout=10000&dubbo=dubbo-provider-golang-1.0.0&" +
		"environment=dev&interface=com.ikurento.user.UserProvider&ip=192.168.56.1&methods=GetUser%2C&" +
		"module=dubbogo+user-info+server&org=ikurento.com&owner=ZX&pid=1448&revision=0.0.1&" +
		"side=provider&timeout=3000&timestamp=1556509797246")
	return u
}
var url3 = func() *common.URL {
	i, _ := common.NewURL("dubbo://127.0.0.1:20002/com.ikurento.user.UserProvider?anyhost=true&" +
		"application=BDTService&category=providers&default.timeout=10000&dubbo=dubbo-provider-golang-1.0.0&" +
		"environment=dev&interface=com.ikurento.user.UserProvider&ip=192.168.56.1&methods=GetUser%2C&" +
		"module=dubbogo+user-info+server&org=ikurento.com&owner=ZX&pid=1449&revision=0.0.1&" +
		"side=provider&timeout=3000&timestamp=1556509797247")
	return i
}

func getRouteCheckArgs() ([]protocol.Invoker, protocol.Invocation, context.Context) {
	return []protocol.Invoker{
			protocol.NewBaseInvoker(url1()), protocol.NewBaseInvoker(url2()), protocol.NewBaseInvoker(url3()),
		}, invocation.NewRPCInvocation("GetUser", nil, map[string]interface{}{
			"attachmentKey": []string{"attachmentValue"},
		}),
		context.TODO()
}

func TestScriptRouter_Route(t *testing.T) {
	type fields struct {
		cfgContent string
	}
	type args struct {
		invokers   []protocol.Invoker
		invocation protocol.Invocation
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   func([]protocol.Invoker) bool
	}{
		{
			name: "base test",
			fields: fields{cfgContent: `configVersion: v3.0
key: dubbo.io
type: javascript
enabled: true
script: |
  (function route(invokers,invocation,context) {
  	var result = [];
  	for (var i = 0; i < invokers.length; i++) {
      invokers[i].GetURL().Port = "20001" 
      result.push(invokers[i]);
  	}
  	return result;
  }(invokers,invocation,context));
`},
			args: func() args {
				res := args{}
				res.invokers, res.invocation, _ = getRouteCheckArgs()
				return res
			}(),
			want: func(invokers []protocol.Invoker) bool {
				for _, invoker := range invokers {
					if invoker.GetURL().Port != "20001" {
						return false
					}
				}
				return true
			},
		}, {
			name: "disable test",
			fields: fields{cfgContent: `configVersion: v3.0
key: dubbo.io
type: javascript
enabled: false
script: |
  (function route(invokers,invocation,context) {
  	var result = [];
  	for (var i = 0; i < invokers.length; i++) {
      invokers[i].GetURL().Port = "20001" 
      result.push(invokers[i]);
  	}
  	return result;
  }(invokers,invocation,context));
`},
			args: func() args {
				res := args{}
				res.invokers, res.invocation, _ = getRouteCheckArgs()
				return res
			}(),
			want: func(invokers []protocol.Invoker) bool {
				expect_invokers, _, _ := getRouteCheckArgs()
				return checkInvokersSame(invokers, expect_invokers)
			},
		}, {
			name: "bad input",
			fields: fields{cfgContent: `configVersion: v3.0
key: dubbo.io
type: javascript
enabled: true
script: |
	badInPut  
	(().Port = "20001" 
      result.push(invokers[i]);
  	}
  	}
  	}
  	return result;
  }(invokers,invocation,context));
`},
			args: func() args {
				res := args{}
				res.invokers, res.invocation, _ = getRouteCheckArgs()
				return res
			}(),
			want: func(invokers []protocol.Invoker) bool {
				expect_invokers, _, _ := getRouteCheckArgs()
				return checkInvokersSame(invokers, expect_invokers)
			},
		}, {
			name: "bad call and recover",
			fields: fields{cfgContent: `configVersion: v3.0
key: dubbo.io
type: javascript
enabled: true
script: |
  (function route(invokers, invocation, context) {
    var result = [];
    for (var i = 0; i < invokers.length; i++) {
      if ("127.0.0.1" === invokers[i].GetURL().Ip) {
        if (invokers[i].GetURLS(" Err Here ").Port !== "20000") {
          invokers[i].GetURLS().Ip = "anotherIP"
          result.push(invokers[i]);
        }
      }
    }
    return result;
  }(invokers, invocation, context));
`},
			args: func() args {
				res := args{}
				res.invokers, res.invocation, _ = getRouteCheckArgs()
				return res
			}(),
			want: func(invokers []protocol.Invoker) bool {
				expect_invokers, _, _ := getRouteCheckArgs()
				return checkInvokersSame(invokers, expect_invokers)
			},
		}, {
			name: "bad type",
			fields: fields{cfgContent: `configVersion: v3.0
key: dubbo.io
type: errorType     # <---
enabled: true
script: |
  (function route(invokers,invocation,context) {
  	var result = [];
  	for (var i = 0; i < invokers.length; i++) {
      invokers[i].GetURL().Port = "20001" 
      result.push(invokers[i]);
  	}
  	return result;
  }(invokers,invocation,context));
`},
			args: func() args {
				res := args{}
				res.invokers, res.invocation, _ = getRouteCheckArgs()
				return res
			}(),
			want: func(invokers []protocol.Invoker) bool {
				expect_invokers, _, _ := getRouteCheckArgs()
				return checkInvokersSame(invokers, expect_invokers)
			},
		},
	}
	for _, tt := range tests {
		s := &ScriptRouter{}
		t.Run(tt.name, func(t *testing.T) {
			s.Process(&config_center.ConfigChangeEvent{Key: "", Value: tt.fields.cfgContent, ConfigType: remoting.EventTypeUpdate})
			got := s.Route(tt.args.invokers, nil, tt.args.invocation)
			assert.True(t, tt.want(got))
		})
	}
}

func checkInvokersSame(invokers []protocol.Invoker, otherInvokers []protocol.Invoker) bool {
	k := map[string]struct{}{}
	for _, invoker := range otherInvokers {
		k[invoker.GetURL().String()] = struct{}{}
	}
	for _, invoker := range invokers {
		_, ok := k[invoker.GetURL().String()]
		if !ok {
			return false
		}
		delete(k, invoker.GetURL().String())
	}
	return len(k) == 0
}
