package config

import (
	"context"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"
)
import (
	log "github.com/AlexStocks/log4go"
	jerrors "github.com/juju/errors"
	"go.uber.org/atomic"
)
import (
	"github.com/dubbo/go-for-apache-dubbo/common"
	"github.com/dubbo/go-for-apache-dubbo/common/constant"
	"github.com/dubbo/go-for-apache-dubbo/common/extension"
	"github.com/dubbo/go-for-apache-dubbo/protocol"
)

type ServiceConfig struct {
	context       context.Context
	Protocol      string           `required:"true"  yaml:"protocol"  json:"protocol,omitempty"` //multi protocol support, split by ','
	InterfaceName string           `required:"true"  yaml:"interface"  json:"interface,omitempty"`
	Registries    []ConfigRegistry `required:"true"  yaml:"registries"  json:"registries,omitempty"`
	Cluster       string           `default:"failover" yaml:"cluster"  json:"cluster,omitempty"`
	Loadbalance   string           `default:"random" yaml:"loadbalance"  json:"loadbalance,omitempty"`
	Group         string           `yaml:"group"  json:"group,omitempty"`
	Version       string           `yaml:"version"  json:"version,omitempty"`
	Methods       []struct {
		Name        string `yaml:"name"  json:"name,omitempty"`
		Retries     int64  `yaml:"retries"  json:"retries,omitempty"`
		Loadbalance string `yaml:"loadbalance"  json:"loadbalance,omitempty"`
		Weight      int64  `yaml:"weight"  json:"weight,omitempty"`
	} `yaml:"methods"  json:"methods,omitempty"`
	Warmup        string `yaml:"warmup"  json:"warmup,omitempty"`
	Retries       int64  `yaml:"retries"  json:"retries,omitempty"`
	unexported    *atomic.Bool
	exported      *atomic.Bool
	rpcService    common.RPCService
	exporters     []protocol.Exporter
	cacheProtocol protocol.Protocol
	cacheMutex    sync.Mutex
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
	if srvconfig.unexported != nil && srvconfig.unexported.Load() {
		err := jerrors.Errorf("The service %v has already unexported! ", srvconfig.InterfaceName)
		log.Error(err.Error())
		return err
	}
	if srvconfig.unexported != nil && srvconfig.exported.Load() {
		log.Warn("The service %v has already exported! ", srvconfig.InterfaceName)
		return nil
	}

	regUrls := loadRegistries(srvconfig.Registries, providerConfig.Registries, common.PROVIDER)
	urlMap := srvconfig.getUrlMap()

	for _, proto := range loadProtocol(srvconfig.Protocol, providerConfig.Protocols) {
		//registry the service reflect
		methods, err := common.ServiceMap.Register(proto.Name, srvconfig.rpcService)
		if err != nil {
			err := jerrors.Errorf("The service %v  export the protocol %v error! Error message is %v .", srvconfig.InterfaceName, proto.Name, err.Error())
			log.Error(err.Error())
			return err
		}
		//contextPath := proto.ContextPath
		//if contextPath == "" {
		//	contextPath = providerConfig.Path
		//}
		url := common.NewURLWithOptions(srvconfig.InterfaceName,
			common.WithProtocol(proto.Name),
			common.WithIp(proto.Ip),
			common.WithPort(proto.Port),
			common.WithParams(urlMap),
			common.WithMethods(strings.Split(methods, ",")))

		for _, regUrl := range regUrls {
			regUrl.SubURL = url
			invoker := protocol.NewBaseInvoker(*regUrl)
			srvconfig.cacheMutex.Lock()
			if srvconfig.cacheProtocol == nil {
				log.Info("First load the registry protocol!")
				srvconfig.cacheProtocol = extension.GetProtocolExtension("registry")
			}
			srvconfig.cacheMutex.Unlock()
			exporter := srvconfig.cacheProtocol.Export(invoker)
			if exporter == nil {
				panic(jerrors.New("New exporter error"))
			}
			srvconfig.exporters = append(srvconfig.exporters, exporter)
		}
	}
	return nil

}

func (srvconfig *ServiceConfig) Implement(s common.RPCService) {
	srvconfig.rpcService = s
}

func (srvconfig *ServiceConfig) getUrlMap() url.Values {
	urlMap := url.Values{}
	urlMap.Set(constant.INTERFACE_KEY, srvconfig.InterfaceName)
	urlMap.Set(constant.TIMESTAMP_KEY, strconv.FormatInt(time.Now().Unix(), 10))
	urlMap.Set(constant.CLUSTER_KEY, srvconfig.Cluster)
	urlMap.Set(constant.LOADBALANCE_KEY, srvconfig.Loadbalance)
	urlMap.Set(constant.WARMUP_KEY, srvconfig.Warmup)
	urlMap.Set(constant.RETRIES_KEY, strconv.FormatInt(srvconfig.Retries, 10))
	urlMap.Set(constant.GROUP_KEY, srvconfig.Group)
	urlMap.Set(constant.VERSION_KEY, srvconfig.Version)
	//application info
	urlMap.Set(constant.APPLICATION_KEY, providerConfig.ApplicationConfig.Name)
	urlMap.Set(constant.ORGANIZATION_KEY, providerConfig.ApplicationConfig.Organization)
	urlMap.Set(constant.NAME_KEY, providerConfig.ApplicationConfig.Name)
	urlMap.Set(constant.MODULE_KEY, providerConfig.ApplicationConfig.Module)
	urlMap.Set(constant.APP_VERSION_KEY, providerConfig.ApplicationConfig.Version)
	urlMap.Set(constant.OWNER_KEY, providerConfig.ApplicationConfig.Owner)
	urlMap.Set(constant.ENVIRONMENT_KEY, providerConfig.ApplicationConfig.Environment)

	for _, v := range srvconfig.Methods {
		urlMap.Set("methods."+v.Name+"."+constant.LOADBALANCE_KEY, v.Loadbalance)
		urlMap.Set("methods."+v.Name+"."+constant.RETRIES_KEY, strconv.FormatInt(v.Retries, 10))
		urlMap.Set("methods."+v.Name+"."+constant.WEIGHT_KEY, strconv.FormatInt(v.Weight, 10))
	}

	return urlMap

}
