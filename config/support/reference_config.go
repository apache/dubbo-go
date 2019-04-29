package support

import (
	"context"
	"net/url"
	"strconv"
	"time"
)

import (
	"github.com/dubbo/dubbo-go/common/constant"
	"github.com/dubbo/dubbo-go/common/extension"
	"github.com/dubbo/dubbo-go/common/proxy"
	"github.com/dubbo/dubbo-go/config"
	"github.com/dubbo/dubbo-go/protocol"
)

type ReferenceConfig struct {
	context     context.Context
	pxy         *proxy.Proxy
	Interface   string           `required:"true"  yaml:"interface"  json:"interface,omitempty"`
	Registries  []ConfigRegistry `required:"true"  yaml:"registries"  json:"registries,omitempty"`
	Cluster     string           `default:"failover" yaml:"cluster"  json:"cluster,omitempty"`
	Loadbalance string           `default:"random" yaml:"loadbalance"  json:"loadbalance,omitempty"`
	retries     int64            `yaml:"retries"  json:"retries,omitempty"`
	Methods     []struct {
		name        string `yaml:"name"  json:"name,omitempty"`
		retries     int64  `yaml:"retries"  json:"retries,omitempty"`
		loadbalance string `yaml:"loadbalance"  json:"loadbalance,omitempty"`
	} `yaml:"methods"  json:"methods,omitempty"`
	Async   bool `yaml:"async"  json:"async,omitempty"`
	invoker protocol.Invoker
}
type ConfigRegistry struct {
	string
}

func NewReferenceConfig(ctx context.Context) *ReferenceConfig {
	return &ReferenceConfig{context: ctx}
}

func (refconfig *ReferenceConfig) Refer() {
	//首先是user specified URL, could be peer-to-peer address, or register center's address.

	//其次是assemble URL from register center's configuration模式
	regUrls := loadRegistries(refconfig.Registries, consumerConfig.Registries)
	url := config.NewURLWithOptions(refconfig.Interface, config.WithParams(refconfig.getUrlMap()))

	//set url to regUrls
	for _, regUrl := range regUrls {
		regUrl.URL = *url
	}

	if len(regUrls) == 1 {
		refconfig.invoker = extension.GetProtocolExtension("registry").Refer(regUrls[0])

	} else {
		//TODO:multi registries ，just wrap multi registry as registry cluster invoker including cluster invoker
	}
	//create proxy
	attachments := map[string]string{} // todo : attachments is necessary, include keys: ASYNC_KEY、
	refconfig.pxy = proxy.NewProxy(refconfig.invoker, nil, attachments)
}

// @v is service provider implemented RPCService
func (refconfig *ReferenceConfig) Implement(v config.RPCService) error {
	return refconfig.pxy.Implement(v)
}

func (refconfig *ReferenceConfig) getUrlMap() url.Values {
	urlMap := url.Values{}

	urlMap.Set(constant.TIMESTAMP_KEY, strconv.FormatInt(time.Now().Unix(), 10))
	urlMap.Set(constant.CLUSTER_KEY, refconfig.Cluster)
	urlMap.Set(constant.LOADBALANCE_KEY, refconfig.Loadbalance)
	urlMap.Set(constant.RETRIES_KEY, strconv.FormatInt(refconfig.retries, 10))
	//getty invoke async or sync
	urlMap.Set(constant.ASYNC_KEY, strconv.FormatBool(refconfig.Async))

	//application info
	urlMap.Set(constant.APPLICATION_KEY, consumerConfig.ApplicationConfig.Name)
	urlMap.Set(constant.ORGANIZATION_KEY, consumerConfig.ApplicationConfig.Organization)
	urlMap.Set(constant.NAME_KEY, consumerConfig.ApplicationConfig.Name)
	urlMap.Set(constant.MODULE_KEY, consumerConfig.ApplicationConfig.Module)
	urlMap.Set(constant.APP_VERSION_KEY, consumerConfig.ApplicationConfig.Version)
	urlMap.Set(constant.OWNER_KEY, consumerConfig.ApplicationConfig.Owner)
	urlMap.Set(constant.ENVIRONMENT_KEY, consumerConfig.ApplicationConfig.Environment)

	for _, v := range refconfig.Methods {
		urlMap.Set("methods."+v.name+"."+constant.LOADBALANCE_KEY, v.loadbalance)
		urlMap.Set("methods."+v.name+"."+constant.RETRIES_KEY, strconv.FormatInt(v.retries, 10))
	}

	return urlMap

}
