package config

import (
	"context"
	"net/url"
	"strconv"
	"time"
)

import (
	"github.com/dubbo/go-for-apache-dubbo/cluster/directory"
	"github.com/dubbo/go-for-apache-dubbo/common"
	"github.com/dubbo/go-for-apache-dubbo/common/constant"
	"github.com/dubbo/go-for-apache-dubbo/common/extension"
	"github.com/dubbo/go-for-apache-dubbo/common/proxy"
	"github.com/dubbo/go-for-apache-dubbo/protocol"
)

type ReferenceConfig struct {
	context       context.Context
	pxy           *proxy.Proxy
	InterfaceName string           `required:"true"  yaml:"interface"  json:"interface,omitempty"`
	Protocol      string           `yaml:"protocol"  json:"protocol,omitempty"`
	Registries    []ConfigRegistry `required:"true"  yaml:"registries"  json:"registries,omitempty"`
	Cluster       string           `yaml:"cluster"  json:"cluster,omitempty"`
	Loadbalance   string           `yaml:"loadbalance"  json:"loadbalance,omitempty"`
	Retries       int64            `yaml:"retries"  json:"retries,omitempty"`
	Group         string           `yaml:"group"  json:"group,omitempty"`
	Version       string           `yaml:"version"  json:"version,omitempty"`
	Methods       []struct {
		Name        string `yaml:"name"  json:"name,omitempty"`
		Retries     int64  `yaml:"retries"  json:"retries,omitempty"`
		Loadbalance string `yaml:"loadbalance"  json:"loadbalance,omitempty"`
	} `yaml:"methods"  json:"methods,omitempty"`
	async   bool `yaml:"async"  json:"async,omitempty"`
	invoker protocol.Invoker
}

type ConfigRegistry string

func NewReferenceConfig(ctx context.Context) *ReferenceConfig {
	return &ReferenceConfig{context: ctx}
}

func (refconfig *ReferenceConfig) Refer() {
	//首先是user specified SubURL, could be peer-to-peer address, or register center's address.

	//其次是assemble SubURL from register center's configuration模式
	regUrls := loadRegistries(refconfig.Registries, consumerConfig.Registries, common.CONSUMER)
	url := common.NewURLWithOptions(refconfig.InterfaceName, common.WithProtocol(refconfig.Protocol), common.WithParams(refconfig.getUrlMap()))

	//set url to regUrls
	for _, regUrl := range regUrls {
		regUrl.SubURL = url
	}

	if len(regUrls) == 1 {
		refconfig.invoker = extension.GetProtocolExtension("registry").Refer(*regUrls[0])

	} else {
		invokers := []protocol.Invoker{}
		for _, regUrl := range regUrls {
			invokers = append(invokers, extension.GetProtocolExtension("registry").Refer(*regUrl))
		}
		cluster := extension.GetCluster("registryAware")
		refconfig.invoker = cluster.Join(directory.NewStaticDirectory(invokers))
	}
	//create proxy
	attachments := map[string]string{}
	attachments[constant.ASYNC_KEY] = url.GetParam(constant.ASYNC_KEY, "false")
	refconfig.pxy = proxy.NewProxy(refconfig.invoker, nil, attachments)
}

// @v is service provider implemented RPCService
func (refconfig *ReferenceConfig) Implement(v common.RPCService) {
	refconfig.pxy.Implement(v)
}

func (refconfig *ReferenceConfig) GetRPCService() common.RPCService {
	return refconfig.pxy.Get()
}

func (refconfig *ReferenceConfig) getUrlMap() url.Values {
	urlMap := url.Values{}
	urlMap.Set(constant.INTERFACE_KEY, refconfig.InterfaceName)
	urlMap.Set(constant.TIMESTAMP_KEY, strconv.FormatInt(time.Now().Unix(), 10))
	urlMap.Set(constant.CLUSTER_KEY, refconfig.Cluster)
	urlMap.Set(constant.LOADBALANCE_KEY, refconfig.Loadbalance)
	urlMap.Set(constant.RETRIES_KEY, strconv.FormatInt(refconfig.Retries, 10))
	urlMap.Set(constant.GROUP_KEY, refconfig.Group)
	urlMap.Set(constant.VERSION_KEY, refconfig.Version)
	//getty invoke async or sync
	urlMap.Set(constant.ASYNC_KEY, strconv.FormatBool(refconfig.async))

	//application info
	urlMap.Set(constant.APPLICATION_KEY, consumerConfig.ApplicationConfig.Name)
	urlMap.Set(constant.ORGANIZATION_KEY, consumerConfig.ApplicationConfig.Organization)
	urlMap.Set(constant.NAME_KEY, consumerConfig.ApplicationConfig.Name)
	urlMap.Set(constant.MODULE_KEY, consumerConfig.ApplicationConfig.Module)
	urlMap.Set(constant.APP_VERSION_KEY, consumerConfig.ApplicationConfig.Version)
	urlMap.Set(constant.OWNER_KEY, consumerConfig.ApplicationConfig.Owner)
	urlMap.Set(constant.ENVIRONMENT_KEY, consumerConfig.ApplicationConfig.Environment)

	for _, v := range refconfig.Methods {
		urlMap.Set("methods."+v.Name+"."+constant.LOADBALANCE_KEY, v.Loadbalance)
		urlMap.Set("methods."+v.Name+"."+constant.RETRIES_KEY, strconv.FormatInt(v.Retries, 10))
	}

	return urlMap

}
