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
	context     context.Context
	Protocol    string           //multi protocol support, split by ','
	Interface   string           `required:"true"  yaml:"interface"  json:"interface,omitempty"`
	Registries  []ConfigRegistry `required:"true"  yaml:"registries"  json:"registries,omitempty"`
	Cluster     string           `default:"failover" yaml:"cluster"  json:"cluster,omitempty"`
	Loadbalance string           `default:"random" yaml:"loadbalance"  json:"loadbalance,omitempty"`
	Methods     []struct {
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
		err := jerrors.Errorf("The service %v has already unexported! ", srvconfig.Interface)
		log.Error(err.Error())
		return err
	}
	if srvconfig.exported.Load() {
		log.Warn("The service %v has already exported! ", srvconfig.Interface)
		return nil
	}

	regUrls := loadRegistries(srvconfig.Registries, providerConfig.Registries)
	urlMap := srvconfig.getUrlMap()

	for _, proto := range loadProtocol(srvconfig.Protocol, providerConfig.Protocols) {
		//registry the service reflect
		_, err := config.ServiceMap.Register(proto.name, srvconfig.rpcService)
		if err != nil {
			err := jerrors.Errorf("The service %v  export the protocol %v error! Error message is %v .", srvconfig.Interface, proto.name, err.Error())
			log.Error(err.Error())
			return err
		}
		contextPath := proto.contextPath
		if contextPath == "" {
			contextPath = providerConfig.Path
		}
		url := config.NewURLWithOptions(srvconfig.Interface,
			config.WithProtocol(proto.name),
			config.WithIp(proto.ip),
			config.WithPort(proto.port),
			config.WithPath(contextPath),
			config.WithParams(urlMap))

		for _, regUrl := range regUrls {
			regUrl.URL = *url
			invoker := protocol.NewBaseInvoker(regUrl)
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
	urlMap.Set(constant.CLUSTER_KEY, srvconfig.Cluster)
	urlMap.Set(constant.LOADBALANCE_KEY, srvconfig.Loadbalance)
	urlMap.Set(constant.WARMUP_KEY, srvconfig.warmup)
	urlMap.Set(constant.RETRIES_KEY, strconv.FormatInt(srvconfig.retries, 10))

	//application info
	urlMap.Set(constant.APPLICATION_KEY, providerConfig.ApplicationConfig.Name)
	urlMap.Set(constant.ORGANIZATION_KEY, providerConfig.ApplicationConfig.Organization)
	urlMap.Set(constant.NAME_KEY, providerConfig.ApplicationConfig.Name)
	urlMap.Set(constant.MODULE_KEY, providerConfig.ApplicationConfig.Module)
	urlMap.Set(constant.APP_VERSION_KEY, providerConfig.ApplicationConfig.Version)
	urlMap.Set(constant.OWNER_KEY, providerConfig.ApplicationConfig.Owner)
	urlMap.Set(constant.ENVIRONMENT_KEY, providerConfig.ApplicationConfig.Environment)

	for _, v := range srvconfig.Methods {
		urlMap.Set("methods."+v.name+"."+constant.LOADBALANCE_KEY, v.loadbalance)
		urlMap.Set("methods."+v.name+"."+constant.RETRIES_KEY, strconv.FormatInt(v.retries, 10))
		urlMap.Set("methods."+v.name+"."+constant.WEIGHT_KEY, strconv.FormatInt(v.weight, 10))
	}

	return urlMap

}
