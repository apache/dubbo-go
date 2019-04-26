package support

import "github.com/dubbo/dubbo-go/config"

type ServiceConfig struct {
	Service    string `required:"true"  yaml:"service"  json:"service,omitempty"`
	URLs       []config.URL
	rpcService config.RPCService
}

func NewDefaultProviderServiceConfig() *ServiceConfig {
	return &ServiceConfig{}
}

// todo: D:\Users\yc.fang\WorkStation\goWorkStation\fangyincheng\dubbo-go\dubbo\server.go#Line102
// todo: D:\Users\yc.fang\WorkStation\goWorkStation\fangyincheng\dubbo-go\jsonrpc\server.go#Line238
func (sc *ServiceConfig) Export() {

}
