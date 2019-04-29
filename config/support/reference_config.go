package support

import (
	"context"
)

import (
	"github.com/dubbo/dubbo-go/common/extension"
	"github.com/dubbo/dubbo-go/common/proxy"
	"github.com/dubbo/dubbo-go/config"
	"github.com/dubbo/dubbo-go/protocol"
)

type ReferenceConfig struct {
	context    context.Context
	pxy        *proxy.Proxy
	Interface  string           `required:"true"  yaml:"interface"  json:"interface,omitempty"`
	Registries []ConfigRegistry `required:"true"  yaml:"registries"  json:"registries,omitempty"`
	Cluster    string           `default:"failover" yaml:"cluster"  json:"cluster,omitempty"`
	Methods    []method         `yaml:"methods"  json:"methods,omitempty"`
	URLs       []config.URL     `yaml:"-"`
	Async      bool             `yaml:"async"  json:"async,omitempty"`
	invoker    protocol.Invoker
}
type ConfigRegistry struct {
	string
}

type method struct {
	name        string `yaml:"name"  json:"name,omitempty"`
	retries     int64  `yaml:"retries"  json:"retries,omitempty"`
	loadbalance string `yaml:"loadbalance"  json:"loadbalance,omitempty"`
}

func NewReferenceConfig(ctx context.Context) *ReferenceConfig {
	return &ReferenceConfig{context: ctx}
}

func (refconfig *ReferenceConfig) Refer() {
	//首先是user specified URL, could be peer-to-peer address, or register center's address.

	//其次是assemble URL from register center's configuration模式
	urls := loadRegistries(refconfig.Registries, consumerConfig.Registries)

	if len(urls) == 1 {
		refconfig.invoker = extension.GetProtocolExtension("registry").Refer(urls[0])

	} else {
		//TODO:multi registries ，just wrap multi registry as registry cluster invoker including cluster invoker
	}
	//create proxy
	attachments := map[string]string{} // todo : attachments is necessary, include keys: ASYNC_KEY、
	refconfig.pxy = proxy.NewProxy(refconfig.invoker, nil, attachments)
}

// @v is service provider implemented RPCService
func (refconfig *ReferenceConfig) Implement(v interface{}) error {
	return refconfig.pxy.Implement(v)
}
