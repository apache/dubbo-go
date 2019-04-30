package support

import (
	"context"
	"net/url"
	"strconv"
	"time"
)
import (
	log "github.com/AlexStocks/log4go"
	jerrors "github.com/juju/errors"
	"go.uber.org/atomic"
)
import (
	"github.com/dubbo/dubbo-go/common/constant"
	"github.com/dubbo/dubbo-go/common/extension"
	"github.com/dubbo/dubbo-go/config"
	"github.com/dubbo/dubbo-go/protocol"
)

type ServiceConfig struct {
	context       context.Context
	protocol      string           //multi protocol support, split by ','
	interfaceName string           `required:"true"  yaml:"interface"  json:"interface,omitempty"`
	registries    []ConfigRegistry `required:"true"  yaml:"registries"  json:"registries,omitempty"`
	cluster       string           `default:"failover" yaml:"cluster"  json:"cluster,omitempty"`
	loadbalance   string           `default:"random" yaml:"loadbalance"  json:"loadbalance,omitempty"`
	group         string           `yaml:"group"  json:"group,omitempty"`
	version       string           `yaml:"version"  json:"version,omitempty"`
	methods       []struct {
		name        string `yaml:"name"  json:"name,omitempty"`
		retries     int64  `yaml:"retries"  json:"retries,omitempty"`
		loadbalance string `yaml:"loadbalance"  json:"loadbalance,omitempty"`
		weight      int64  `yaml:"weight"  json:"weight,omitempty"`
	} `yaml:"methods"  json:"methods,omitempty"`
	warmup     string `yaml:"warmup"  json:"warmup,omitempty"`
	retries    int64  `yaml:"retries"  json:"retries,omitempty"`
	unexported *atomic.Bool
	exported   *atomic.Bool
	rpcService config.RPCService
	exporters  []protocol.Exporter
}

func NewServiceConfig() *ServiceConfig {
	return &ServiceConfig{
		unexported: atomic.NewBool(false),
		exported:   atomic.NewBool(false),
	}

}

func (srvconfig *ServiceConfig) Export() error {
	//TODO: config center start here

	//TODO:delay export
	if srvconfig.unexported.Load() {
		err := jerrors.Errorf("The service %v has already unexported! ", srvconfig.interfaceName)
		log.Error(err.Error())
		return err
	}
	if srvconfig.exported.Load() {
		log.Warn("The service %v has already exported! ", srvconfig.interfaceName)
		return nil
	}

	regUrls := loadRegistries(srvconfig.registries, providerConfig.Registries, config.PROVIDER)
	urlMap := srvconfig.getUrlMap()

	for _, proto := range loadProtocol(srvconfig.protocol, providerConfig.Protocols) {
		//registry the service reflect
		_, err := config.ServiceMap.Register(proto.name, srvconfig.rpcService)
		if err != nil {
			err := jerrors.Errorf("The service %v  export the protocol %v error! Error message is %v .", srvconfig.interfaceName, proto.name, err.Error())
			log.Error(err.Error())
			return err
		}
		contextPath := proto.contextPath
		if contextPath == "" {
			contextPath = providerConfig.Path
		}
		url := config.NewURLWithOptions(srvconfig.interfaceName,
			config.WithProtocol(proto.name),
			config.WithIp(proto.ip),
			config.WithPort(proto.port),
			config.WithPath(contextPath),
			config.WithParams(urlMap))

		for _, regUrl := range regUrls {
			regUrl.SubURL = url
			invoker := protocol.NewBaseInvoker(*regUrl)
			exporter := extension.GetProtocolExtension("registry").Export(invoker)
			srvconfig.exporters = append(srvconfig.exporters, exporter)
		}
	}
	return nil

}

func (srvconfig *ServiceConfig) Implement(s config.RPCService) {
	srvconfig.rpcService = s
}

func (srvconfig *ServiceConfig) getUrlMap() url.Values {
	urlMap := url.Values{}

	urlMap.Set(constant.TIMESTAMP_KEY, strconv.FormatInt(time.Now().Unix(), 10))
	urlMap.Set(constant.CLUSTER_KEY, srvconfig.cluster)
	urlMap.Set(constant.LOADBALANCE_KEY, srvconfig.loadbalance)
	urlMap.Set(constant.WARMUP_KEY, srvconfig.warmup)
	urlMap.Set(constant.RETRIES_KEY, strconv.FormatInt(srvconfig.retries, 10))
	urlMap.Set(constant.GROUP_KEY, srvconfig.group)
	urlMap.Set(constant.VERSION_KEY, srvconfig.version)
	//application info
	urlMap.Set(constant.APPLICATION_KEY, providerConfig.ApplicationConfig.Name)
	urlMap.Set(constant.ORGANIZATION_KEY, providerConfig.ApplicationConfig.Organization)
	urlMap.Set(constant.NAME_KEY, providerConfig.ApplicationConfig.Name)
	urlMap.Set(constant.MODULE_KEY, providerConfig.ApplicationConfig.Module)
	urlMap.Set(constant.APP_VERSION_KEY, providerConfig.ApplicationConfig.Version)
	urlMap.Set(constant.OWNER_KEY, providerConfig.ApplicationConfig.Owner)
	urlMap.Set(constant.ENVIRONMENT_KEY, providerConfig.ApplicationConfig.Environment)

	for _, v := range srvconfig.methods {
		urlMap.Set("methods."+v.name+"."+constant.LOADBALANCE_KEY, v.loadbalance)
		urlMap.Set("methods."+v.name+"."+constant.RETRIES_KEY, strconv.FormatInt(v.retries, 10))
		urlMap.Set("methods."+v.name+"."+constant.WEIGHT_KEY, strconv.FormatInt(v.weight, 10))
	}

	return urlMap

}
