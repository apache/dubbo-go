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

package servicediscovery

import (
	"bytes"
	"strings"
	"sync"
)

import (
	gxset "github.com/dubbogo/gost/container/set"
	perrors "github.com/pkg/errors"
	"go.uber.org/atomic"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/common/extension"
	"dubbo.apache.org/dubbo-go/v3/common/logger"
	"dubbo.apache.org/dubbo-go/v3/config"
	"dubbo.apache.org/dubbo-go/v3/metadata/mapping"
	"dubbo.apache.org/dubbo-go/v3/metadata/service"
	"dubbo.apache.org/dubbo-go/v3/metadata/service/exporter/configurable"
	"dubbo.apache.org/dubbo-go/v3/metadata/service/inmemory"
	"dubbo.apache.org/dubbo-go/v3/registry"
	"dubbo.apache.org/dubbo-go/v3/registry/event"
	"dubbo.apache.org/dubbo-go/v3/registry/servicediscovery/synthesizer"
)

const (
	protocolName = "service-discovery"
)

func init() {
	extension.SetRegistry(protocolName, newServiceDiscoveryRegistry)
}

// serviceDiscoveryRegistry is the implementation of application-level registry.
// It's completely different from other registry implementations
// This implementation is based on ServiceDiscovery abstraction and ServiceNameMapping
// In order to keep compatible with interface-level registry，
// this implementation is
type serviceDiscoveryRegistry struct {
	lock                             sync.RWMutex
	url                              *common.URL
	serviceDiscovery                 registry.ServiceDiscovery
	subscribedServices               *gxset.HashSet
	serviceNameMapping               mapping.ServiceNameMapping
	metaDataService                  service.MetadataService
	registeredListeners              *gxset.HashSet
	subscribedURLsSynthesizers       []synthesizer.SubscribedURLsSynthesizer
	serviceRevisionExportedURLsCache map[string]map[string][]*common.URL
	serviceListeners                 map[string]registry.ServiceInstancesChangedListener
}

func newServiceDiscoveryRegistry(url *common.URL) (registry.Registry, error) {

	tryInitMetadataService(url)

	serviceDiscovery, err := creatServiceDiscovery(url)
	if err != nil {
		return nil, err
	}
	subscribedServices := parseServices(url.GetParam(constant.SUBSCRIBED_SERVICE_NAMES_KEY, ""))
	subscribedURLsSynthesizers := synthesizer.GetAllSynthesizer()
	serviceNameMapping := extension.GetGlobalServiceNameMapping()
	metaDataService, err := inmemory.GetInMemoryMetadataService()
	if err != nil {
		return nil, perrors.WithMessage(err, "could not init metadata service")
	}
	return &serviceDiscoveryRegistry{
		url:                              url,
		serviceDiscovery:                 serviceDiscovery,
		subscribedServices:               subscribedServices,
		subscribedURLsSynthesizers:       subscribedURLsSynthesizers,
		registeredListeners:              gxset.NewSet(),
		serviceRevisionExportedURLsCache: make(map[string]map[string][]*common.URL, 8),
		serviceNameMapping:               serviceNameMapping,
		metaDataService:                  metaDataService,
		serviceListeners:                 make(map[string]registry.ServiceInstancesChangedListener),
	}, nil
}

func (s *serviceDiscoveryRegistry) UnRegister(url *common.URL) error {
	if !shouldRegister(url) {
		return nil
	}
	return s.metaDataService.UnexportURL(url)
}

func (s *serviceDiscoveryRegistry) UnSubscribe(url *common.URL, listener registry.NotifyListener) error {
	if !shouldSubscribe(url) {
		return nil
	}
	err := s.metaDataService.UnsubscribeURL(url)
	if err != nil {
		return err
	}
	services := s.getServices(url)
	if services == nil {
		return nil
	}
	// FIXME ServiceNames.String() is not good
	serviceNamesKey := services.String()
	l := s.serviceListeners[serviceNamesKey]
	l.RemoveListener(url.ServiceKey())
	return nil
}

func creatServiceDiscovery(url *common.URL) (registry.ServiceDiscovery, error) {
	sdcName := url.GetParam(constant.SERVICE_DISCOVERY_KEY, "")
	sdc, ok := config.GetBaseConfig().GetServiceDiscoveries(sdcName)
	if !ok {
		return nil, perrors.Errorf("The service discovery with name: %s is not found", sdcName)
	}
	originServiceDiscovery, err := extension.GetServiceDiscovery(sdc.Protocol, sdcName)
	if err != nil {
		return nil, perrors.WithMessage(err, "Create service discovery fialed")
	}
	return event.NewEventPublishingServiceDiscovery(originServiceDiscovery), nil
}

func parseServices(literalServices string) *gxset.HashSet {
	set := gxset.NewSet()
	if len(literalServices) == 0 {
		return set
	}
	var splitServices = strings.Split(literalServices, ",")
	for _, s := range splitServices {
		if len(s) != 0 {
			set.Add(s)
		}
	}
	return set
}

func (s *serviceDiscoveryRegistry) GetServiceDiscovery() registry.ServiceDiscovery {
	return s.serviceDiscovery
}

func (s *serviceDiscoveryRegistry) GetURL() *common.URL {
	return s.url
}

func (s *serviceDiscoveryRegistry) IsAvailable() bool {
	if s.serviceDiscovery.GetServices() == nil {
		return false
	}
	return len(s.serviceDiscovery.GetServices().Values()) > 0
}

func (s *serviceDiscoveryRegistry) Destroy() {
	err := s.serviceDiscovery.Destroy()
	if err != nil {
		logger.Errorf("destroy serviceDiscovery catch error:%s", err.Error())
	}
}

func (s *serviceDiscoveryRegistry) Register(url *common.URL) error {
	if !shouldRegister(url) {
		return nil
	}
	ok, err := s.metaDataService.ExportURL(url)

	if err != nil {
		logger.Errorf("The URL[%s] registry catch error:%s!", url.String(), err.Error())
		return err
	}
	if !ok {
		logger.Warnf("The URL[%s] has been registry!", url.String())
	}

	return s.serviceNameMapping.Map(url.GetParam(constant.INTERFACE_KEY, ""),
		url.GetParam(constant.GROUP_KEY, ""),
		url.GetParam(constant.Version, ""),
		url.Protocol)
}

func shouldRegister(url *common.URL) bool {
	side := url.GetParam(constant.SIDE_KEY, "")
	if side == constant.PROVIDER_PROTOCOL {
		return true
	}
	logger.Debugf("The URL should not be register.", url.String())
	return false
}

func (s *serviceDiscoveryRegistry) Subscribe(url *common.URL, notify registry.NotifyListener) error {
	if !shouldSubscribe(url) {
		return nil
	}
	_, err := s.metaDataService.SubscribeURL(url)
	if err != nil {
		return perrors.WithMessage(err, "subscribe url error: "+url.String())
	}
	services := s.getServices(url)
	if services.Empty() {
		return perrors.Errorf("Should has at least one way to know which services this interface belongs to, "+
			"subscription url:%s", url.String())
	}
	// FIXME ServiceNames.String() is not good
	serviceNamesKey := services.String()
	protocolServiceKey := url.ServiceKey() + ":" + url.Protocol
	listener := s.serviceListeners[serviceNamesKey]
	if listener == nil {
		listener = event.NewServiceInstancesChangedListener(services)
		for _, serviceNameTmp := range services.Values() {
			serviceName := serviceNameTmp.(string)
			instances := s.serviceDiscovery.GetInstances(serviceName)
			err = listener.OnEvent(&registry.ServiceInstancesChangedEvent{
				ServiceName: serviceName,
				Instances:   instances,
			})
			if err != nil {
				logger.Warnf("[ServiceDiscoveryRegistry] ServiceInstancesChangedListenerImpl handle error:%v", err)
			}
		}
	}
	s.serviceListeners[serviceNamesKey] = listener
	listener.AddListenerAndNotify(protocolServiceKey, notify)
	s.registerServiceInstancesChangedListener(url, listener)
	return nil
}

func (s *serviceDiscoveryRegistry) registerServiceInstancesChangedListener(url *common.URL, listener registry.ServiceInstancesChangedListener) {
	// FIXME ServiceNames.String() is not good
	listenerId := listener.GetServiceNames().String() + ":" + getUrlKey(url)
	if !s.subscribedServices.Contains(listenerId) {
		err := s.serviceDiscovery.AddListener(listener)
		if err != nil {
			logger.Errorf("add listener[%s] catch error,url:%s err:%s", listenerId, url.String(), err.Error())
		}
	}

}

func getUrlKey(url *common.URL) string {
	var bf bytes.Buffer
	if len(url.Protocol) != 0 {
		bf.WriteString(url.Protocol)
		bf.WriteString("://")
	}
	if len(url.Location) != 0 {
		bf.WriteString(url.Location)
		bf.WriteString(":")
		bf.WriteString(url.Port)
	}
	if len(url.Path) != 0 {
		bf.WriteString("/")
		bf.WriteString(url.Path)
	}
	bf.WriteString("?")
	appendParam(bf, constant.VERSION_KEY, url)
	appendParam(bf, constant.GROUP_KEY, url)
	appendParam(bf, constant.NACOS_PROTOCOL_KEY, url)
	return bf.String()
}

func appendParam(buffer bytes.Buffer, paramKey string, url *common.URL) {
	buffer.WriteString(paramKey)
	buffer.WriteString("=")
	buffer.WriteString(url.GetParam(paramKey, ""))
}

func (s *serviceDiscoveryRegistry) synthesizeSubscribedURLs(subscribedURL *common.URL, serviceInstances []registry.ServiceInstance) []*common.URL {
	var urls []*common.URL
	for _, syn := range s.subscribedURLsSynthesizers {
		if syn.Support(subscribedURL) {
			urls = append(urls, syn.Synthesize(subscribedURL, serviceInstances)...)
		}
	}
	return urls
}

func shouldSubscribe(url *common.URL) bool {
	return !shouldRegister(url)
}

func (s *serviceDiscoveryRegistry) getServices(url *common.URL) *gxset.HashSet {
	services := gxset.NewSet()
	serviceNames := url.GetParam(constant.PROVIDER_BY, "")
	if len(serviceNames) > 0 {
		services = parseServices(serviceNames)
	}
	if services.Empty() {
		services = s.findMappedServices(url)
		if services.Empty() {
			return s.subscribedServices
		}
	}
	return services
}

func (s *serviceDiscoveryRegistry) findMappedServices(url *common.URL) *gxset.HashSet {
	serviceInterface := url.GetParam(constant.INTERFACE_KEY, url.Path)
	group := url.GetParam(constant.GROUP_KEY, "")
	version := url.GetParam(constant.VERSION_KEY, "")
	protocol := url.Protocol
	serviceNames, err := s.serviceNameMapping.Get(serviceInterface, group, version, protocol)
	if err != nil {
		logger.Errorf("get serviceInterface:[%s] group:[%s] version:[%s] protocol:[%s] from "+
			"serviceNameMap error:%s", err.Error())
		return gxset.NewSet()
	}
	return serviceNames
}

var (
	exporting = &atomic.Bool{}
)

// tryInitMetadataService will try to initialize metadata service
// TODO (move to somewhere)
func tryInitMetadataService(url *common.URL) {

	ms, err := inmemory.GetInMemoryMetadataService()
	if err != nil {
		logger.Errorf("could not init metadata service", err)
	}

	if !config.IsProvider() || exporting.Load() {
		return
	}

	// In theory, we can use sync.Once
	// But sync.Once is not reentrant.
	// Now the invocation chain is createRegistry -> tryInitMetadataService -> metadataServiceExporter.export
	// -> createRegistry -> initMetadataService...
	// So using sync.Once will result in dead lock
	exporting.Store(true)

	expt := configurable.NewMetadataServiceExporter(ms)

	err = expt.Export(url)
	if err != nil {
		logger.Errorf("could not export the metadata service", err)
	}
	extension.GetGlobalDispatcher().Dispatch(event.NewServiceConfigExportedEvent(expt.(*configurable.MetadataServiceExporter).ServiceConfig))
}
