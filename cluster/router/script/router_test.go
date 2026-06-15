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
	"testing"
)

import (
	"github.com/stretchr/testify/assert"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/config_center"
	"dubbo.apache.org/dubbo-go/v3/global"
	"dubbo.apache.org/dubbo-go/v3/protocol/base"
	"dubbo.apache.org/dubbo-go/v3/protocol/invocation"
	"dubbo.apache.org/dubbo-go/v3/remoting"
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

func getRouteCheckArgs() ([]base.Invoker, base.Invocation, context.Context) {
	return []base.Invoker{
			base.NewBaseInvoker(url1()), base.NewBaseInvoker(url2()), base.NewBaseInvoker(url3()),
		}, invocation.NewRPCInvocation("GetUser", nil, map[string]any{
			"attachmentKey": []string{"attachmentValue"},
		}),
		context.TODO()
}

func TestScriptRouter_Route(t *testing.T) {
	type fields struct {
		cfgContent string
	}
	type args struct {
		invokers   []base.Invoker
		invocation base.Invocation
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   func([]base.Invoker) bool
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
			want: func(invokers []base.Invoker) bool {
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
			want: func(invokers []base.Invoker) bool {
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
			want: func(invokers []base.Invoker) bool {
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
			want: func(invokers []base.Invoker) bool {
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
			want: func(invokers []base.Invoker) bool {
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

func TestScriptRouterProcessSkipsNonStringConfig(t *testing.T) {
	s := &ScriptRouter{
		enabled:    true,
		scriptType: "javascript",
		rawScript:  "old script",
	}

	assert.NotPanics(t, func() {
		s.Process(&config_center.ConfigChangeEvent{Key: "", Value: 123, ConfigType: remoting.EventTypeUpdate})
	})
	assert.True(t, s.enabled)
	assert.Equal(t, "javascript", s.scriptType)
	assert.Equal(t, "old script", s.rawScript)
}

func TestScriptRouterProcessDelSkipsConfigBody(t *testing.T) {
	s := &ScriptRouter{
		enabled:    true,
		scriptType: "javascript",
		rawScript:  "old script",
	}

	assert.NotPanics(t, func() {
		s.Process(&config_center.ConfigChangeEvent{Key: "", Value: nil, ConfigType: remoting.EventTypeDel})
	})
	assert.False(t, s.enabled)
	assert.Empty(t, s.scriptType)
	assert.Empty(t, s.rawScript)
}

func TestScriptRouterSetStaticConfig(t *testing.T) {
	staticScriptForPort := func(port string) string {
		return `(function route(invokers, invocation, context) {
		var result = [];
		for (var i = 0; i < invokers.length; i++) {
			if (invokers[i].GetURL().Port === "` + port + `") {
				result.push(invokers[i]);
			}
		}
		return result;
	}(invokers, invocation, context));`
	}

	t.Run("apply static script config", func(t *testing.T) {
		invokers, inv, _ := getRouteCheckArgs()
		s := NewScriptRouter()
		s.SetStaticConfig(&global.RouterConfig{
			Scope:      constant.RouterScopeApplication,
			Key:        "dubbo.io",
			ScriptType: "javascript",
			Script:     staticScriptForPort("20001"),
		})

		got := s.Route(invokers, nil, inv)
		assert.Len(t, got, 1)
		assert.Equal(t, "20001", got[0].GetURL().Port)
	})

	t.Run("ignore disabled static script config", func(t *testing.T) {
		invokers, inv, _ := getRouteCheckArgs()
		enabled := false
		s := NewScriptRouter()
		s.SetStaticConfig(&global.RouterConfig{
			Scope:      constant.RouterScopeApplication,
			Key:        "dubbo.io",
			Enabled:    &enabled,
			ScriptType: "javascript",
			Script:     staticScriptForPort("20001"),
		})

		got := s.Route(invokers, nil, inv)
		assert.True(t, checkInvokersSame(got, invokers))
	})

	t.Run("ignore config without script", func(t *testing.T) {
		invokers, inv, _ := getRouteCheckArgs()
		s := NewScriptRouter()
		s.SetStaticConfig(&global.RouterConfig{
			Scope: constant.RouterScopeApplication,
			Key:   "dubbo.io",
		})

		got := s.Route(invokers, nil, inv)
		assert.True(t, checkInvokersSame(got, invokers))
	})

	t.Run("replace previous static script config", func(t *testing.T) {
		invokers, inv, _ := getRouteCheckArgs()
		s := NewScriptRouter()
		s.SetStaticConfig(&global.RouterConfig{
			Scope:      constant.RouterScopeApplication,
			Key:        "dubbo.io",
			ScriptType: "javascript",
			Script:     staticScriptForPort("20001"),
		})
		s.SetStaticConfig(&global.RouterConfig{
			Scope:      constant.RouterScopeApplication,
			Key:        "dubbo.io",
			ScriptType: "javascript",
			Script:     staticScriptForPort("20002"),
		})

		got := s.Route(invokers, nil, inv)
		assert.Len(t, got, 1)
		assert.Equal(t, "20002", got[0].GetURL().Port)
	})

	t.Run("replace stale config with unsupported script type", func(t *testing.T) {
		invokers, inv, _ := getRouteCheckArgs()
		s := &ScriptRouter{
			enabled:    true,
			scriptType: "unsupported",
			rawScript:  "old script",
		}
		s.SetStaticConfig(&global.RouterConfig{
			Scope:      constant.RouterScopeApplication,
			Key:        "dubbo.io",
			ScriptType: "javascript",
			Script:     staticScriptForPort("20001"),
		})

		got := s.Route(invokers, nil, inv)
		assert.Len(t, got, 1)
		assert.Equal(t, "20001", got[0].GetURL().Port)
	})

	t.Run("disable unsupported script type", func(t *testing.T) {
		invokers, inv, _ := getRouteCheckArgs()
		s := NewScriptRouter()
		s.SetStaticConfig(&global.RouterConfig{
			Scope:      constant.RouterScopeApplication,
			Key:        "dubbo.io",
			ScriptType: "unsupported",
			Script:     staticScriptForPort("20001"),
		})

		got := s.Route(invokers, nil, inv)
		assert.True(t, checkInvokersSame(got, invokers))
	})

	t.Run("disable invalid static script", func(t *testing.T) {
		invokers, inv, _ := getRouteCheckArgs()
		s := NewScriptRouter()
		s.SetStaticConfig(&global.RouterConfig{
			Scope:      constant.RouterScopeApplication,
			Key:        "dubbo.io",
			ScriptType: "javascript",
			Script:     "bad input",
		})

		got := s.Route(invokers, nil, inv)
		assert.True(t, checkInvokersSame(got, invokers))
	})
}

func checkInvokersSame(invokers []base.Invoker, otherInvokers []base.Invoker) bool {
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
