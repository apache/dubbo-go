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

func getRouteArgs() ([]protocol.Invoker, protocol.Invocation, context.Context) {
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
				res.invokers, res.invocation, _ = getRouteArgs()
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
				res.invokers, res.invocation, _ = getRouteArgs()
				return res
			}(),
			want: func(invokers []protocol.Invoker) bool {
				return true
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
				res.invokers, res.invocation, _ = getRouteArgs()
				return res
			}(),
			want: func(invokers []protocol.Invoker) bool {
				return true
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
