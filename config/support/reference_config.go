package support

import (
	"context"
	"github.com/dubbo/dubbo-go/cluster/directory"
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
	context       context.Context
	pxy           *proxy.Proxy
	interfaceName string           `required:"true"  yaml:"interface"  json:"interface,omitempty"`
	protocol      string           `yaml:"protocol"  json:"protocol,omitempty"`
	registries    []ConfigRegistry `required:"true"  yaml:"registries"  json:"registries,omitempty"`
	cluster       string           `yaml:"cluster"  json:"cluster,omitempty"`
	loadbalance   string           `yaml:"loadbalance"  json:"loadbalance,omitempty"`
	retries       int64            `yaml:"retries"  json:"retries,omitempty"`
	group         string           `yaml:"group"  json:"group,omitempty"`
	version       string           `yaml:"version"  json:"version,omitempty"`
	methods       []struct {
		name        string `yaml:"name"  json:"name,omitempty"`
		retries     int64  `yaml:"retries"  json:"retries,omitempty"`
		loadbalance string `yaml:"loadbalance"  json:"loadbalance,omitempty"`
	} `yaml:"methods"  json:"methods,omitempty"`
	async   bool `yaml:"async"  json:"async,omitempty"`
	invoker protocol.Invoker
}
type ConfigRegistry struct {
	string
}

func NewReferenceConfig(ctx context.Context) *ReferenceConfig {
	return &ReferenceConfig{context: ctx}
}

func (refconfig *ReferenceConfig) Refer() {
	//首先是user specified SubURL, could be peer-to-peer address, or register center's address.

	//其次是assemble SubURL from register center's configuration模式
	regUrls := loadRegistries(refconfig.registries, consumerConfig.Registries, config.CONSUMER)
	url := config.NewURLWithOptions(refconfig.interfaceName, config.WithParams(refconfig.getUrlMap()))

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
func (refconfig *ReferenceConfig) Implement(v config.RPCService) error {
	return refconfig.pxy.Implement(v)
}

func (refconfig *ReferenceConfig) getUrlMap() url.Values {
	urlMap := url.Values{}

	urlMap.Set(constant.TIMESTAMP_KEY, strconv.FormatInt(time.Now().Unix(), 10))
	urlMap.Set(constant.CLUSTER_KEY, refconfig.cluster)
	urlMap.Set(constant.LOADBALANCE_KEY, refconfig.loadbalance)
	urlMap.Set(constant.RETRIES_KEY, strconv.FormatInt(refconfig.retries, 10))
	urlMap.Set(constant.GROUP_KEY, refconfig.group)
	urlMap.Set(constant.VERSION_KEY, refconfig.version)
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

	for _, v := range refconfig.methods {
		urlMap.Set("methods."+v.name+"."+constant.LOADBALANCE_KEY, v.loadbalance)
		urlMap.Set("methods."+v.name+"."+constant.RETRIES_KEY, strconv.FormatInt(v.retries, 10))
	}

	return urlMap

}
