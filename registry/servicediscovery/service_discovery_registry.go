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
	"time"
)

import (
	gxset "github.com/dubbogo/gost/container/set"
	"github.com/dubbogo/gost/log/logger"

	perrors "github.com/pkg/errors"

	"go.uber.org/atomic"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/common/extension"
	"dubbo.apache.org/dubbo-go/v3/metadata/mapping"
	"dubbo.apache.org/dubbo-go/v3/metadata/service"
	"dubbo.apache.org/dubbo-go/v3/metadata/service/local"
	"dubbo.apache.org/dubbo-go/v3/metrics"
	metricMetadata "dubbo.apache.org/dubbo-go/v3/metrics/metadata"
	metricsRegistry "dubbo.apache.org/dubbo-go/v3/metrics/registry"
	"dubbo.apache.org/dubbo-go/v3/registry"
	_ "dubbo.apache.org/dubbo-go/v3/registry/event"
	"dubbo.apache.org/dubbo-go/v3/registry/servicediscovery/synthesizer"
)

func init() {
	extension.SetRegistry(constant.ServiceRegistryProtocol, newServiceDiscoveryRegistry)
}

// ServiceDiscoveryRegistry is the implementation of application-level registry.
// It's completely different from other registry implementations
// This implementation is based on ServiceDiscovery abstraction and ServiceNameMapping
// In order to keep compatible with interface-level registry，
// this implementation is
type ServiceDiscoveryRegistry struct {
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
	serviceMappingListeners          map[string]registry.MappingListener
}

func newServiceDiscoveryRegistry(url *common.URL) (registry.Registry, error) {
	serviceDiscovery, err := creatServiceDiscovery(url)
	if err != nil {
		return nil, err
	}
	subscribedServices := parseServices(url.GetParam(constant.SubscribedServiceNamesKey, ""))
	subscribedURLsSynthesizers := synthesizer.GetAllSynthesizer()
	serviceNameMapping := extension.GetGlobalServiceNameMapping()
	metaDataService, err := local.GetLocalMetadataService()
	if err != nil {
		return nil, perrors.WithMessage(err, "could not init metadata service")
	}
	return &ServiceDiscoveryRegistry{
		url:                              url,
		serviceDiscovery:                 serviceDiscovery,
		subscribedServices:               subscribedServices,
		subscribedURLsSynthesizers:       subscribedURLsSynthesizers,
		registeredListeners:              gxset.NewSet(),
		serviceRevisionExportedURLsCache: make(map[string]map[string][]*common.URL, 8),
		serviceNameMapping:               serviceNameMapping,
		metaDataService:                  metaDataService,
		serviceListeners:                 make(map[string]registry.ServiceInstancesChangedListener),
		// cache for mapping listener
		serviceMappingListeners: make(map[string]registry.MappingListener),
	}, nil
}

func (s *ServiceDiscoveryRegistry) UnRegister(url *common.URL) error {
	if !shouldRegister(url) {
		return nil
	}
	return s.metaDataService.UnexportURL(url)
}

func (s *ServiceDiscoveryRegistry) UnSubscribe(url *common.URL, listener registry.NotifyListener) error {
	if !shouldSubscribe(url) {
		return nil
	}
	err := s.metaDataService.UnsubscribeURL(url)
	if err != nil {
		return err
	}
	services := s.getServices(url, nil)
	if services == nil {
		return nil
	}
	// FIXME ServiceNames.String() is not good
	serviceNamesKey := services.String()
	l := s.serviceListeners[serviceNamesKey]
	l.RemoveListener(url.ServiceKey())
	s.stopListen(url)
	err = s.serviceNameMapping.Remove(url)
	if err != nil {
		return err
	}
	return nil
}

func creatServiceDiscovery(url *common.URL) (registry.ServiceDiscovery, error) {
	originServiceDiscovery, err := extension.GetServiceDiscovery(url)
	if err != nil {
		return nil, perrors.WithMessage(err, "Create service discovery fialed")
	}
	return originServiceDiscovery, nil
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

func (s *ServiceDiscoveryRegistry) GetServiceDiscovery() registry.ServiceDiscovery {
	return s.serviceDiscovery
}

func (s *ServiceDiscoveryRegistry) GetURL() *common.URL {
	return s.url
}

func (s *ServiceDiscoveryRegistry) IsAvailable() bool {
	if s.serviceDiscovery.GetServices() == nil {
		return false
	}
	return len(s.serviceDiscovery.GetServices().Values()) > 0
}

func (s *ServiceDiscoveryRegistry) Destroy() {
	err := s.serviceDiscovery.Destroy()
	if err != nil {
		logger.Errorf("destroy serviceDiscovery catch error:%s", err.Error())
	}
}

func (s *ServiceDiscoveryRegistry) Register(url *common.URL) error {
	if !shouldRegister(url) {
		return nil
	}
	common.HandleRegisterIPAndPort(url)

	start := time.Now()
	ok, err := s.metaDataService.ExportURL(url)
	metrics.Publish(metricsRegistry.NewServerRegisterEvent(ok && err == nil, start))

	if err != nil {
		logger.Errorf("The URL[%s] registry catch error:%s!", url.String(), err.Error())
		return err
	}
	if !ok {
		logger.Warnf("The URL[%s] has been registry!", url.String())
	}

	return s.serviceNameMapping.Map(url)
}

func shouldRegister(url *common.URL) bool {
	side := url.GetParam(constant.SideKey, "")
	if side == constant.ProviderProtocol {
		return true
	}
	logger.Debugf("The URL should not be register.", url.String())
	return false
}

func (s *ServiceDiscoveryRegistry) Subscribe(url *common.URL, notify registry.NotifyListener) error {
	if !shouldSubscribe(url) {
		logger.Infof("Service %s is set to not subscribe to instances.", url.ServiceKey())
		return nil
	}
	_, err := s.metaDataService.SubscribeURL(url)
	if err != nil {
		return perrors.WithMessage(err, "subscribe url error: "+url.String())
	}

	mappingListener := NewMappingListener(s.url, url, s.subscribedServices, notify)
	services := s.getServices(url, mappingListener)
	if services.Empty() {
		return perrors.Errorf("Should has at least one way to know which services this interface belongs to,"+
			" either specify 'provided-by' for reference or enable metadata-report center subscription url:%s", url.String())
	}
	logger.Infof("Find initial mapping applications %q for service %s.", services, url.ServiceKey())
	// first notify
	err = mappingListener.OnEvent(registry.NewServiceMappingChangedEvent(url.ServiceKey(), services))
	if err != nil {
		logger.Errorf("[ServiceDiscoveryRegistry] ServiceInstancesChangedListenerImpl handle error:%v", err)
	}
	return nil
}

func (s *ServiceDiscoveryRegistry) SubscribeURL(url *common.URL, notify registry.NotifyListener, services *gxset.HashSet) {
	// FIXME ServiceNames.String() is not good
	var err error
	serviceNamesKey := services.String()
	protocol := "tri" // consume "tri" protocol by default, other protocols need to be specified on reference/consumer explicitly
	if url.Protocol != "" {
		protocol = url.Protocol
	}
	protocolServiceKey := url.ServiceKey() + ":" + protocol
	listener := s.serviceListeners[serviceNamesKey]
	if listener == nil {
		listener = NewServiceInstancesChangedListener(url.GetParam(constant.ApplicationKey, ""), services)
		for _, serviceNameTmp := range services.Values() {
			serviceName := serviceNameTmp.(string)
			instances := s.serviceDiscovery.GetInstances(serviceName)
			logger.Infof("Synchronized instance notification on application %s subscription, instance list size %s", serviceName, len(instances))
			err = listener.OnEvent(&registry.ServiceInstancesChangedEvent{
				ServiceName: serviceName,
				Instances:   instances,
			})
			if err != nil {
				logger.Errorf("[ServiceDiscoveryRegistry] ServiceInstancesChangedListenerImpl handle error:%v", err)
			}
		}
	}
	s.serviceListeners[serviceNamesKey] = listener
	listener.AddListenerAndNotify(protocolServiceKey, notify)
	event := metricMetadata.NewMetadataMetricTimeEvent(metricMetadata.SubscribeServiceRt)

	logger.Infof("Start subscribing to registry for applications :%s with a new go routine.", serviceNamesKey)
	go func() {
		err = s.serviceDiscovery.AddListener(listener)
		event.Succ = err != nil
		event.End = time.Now()
		event.Attachment[constant.InterfaceKey] = url.Interface()
		metrics.Publish(event)
		metrics.Publish(metricsRegistry.NewServerSubscribeEvent(err == nil))
		if err != nil {
			logger.Errorf("add instance listener catch error,url:%s err:%s", url.String(), err.Error())
		}
	}()
}

// LoadSubscribeInstances load subscribe instance
func (s *ServiceDiscoveryRegistry) LoadSubscribeInstances(url *common.URL, notify registry.NotifyListener) error {
	return nil
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
	appendParam(bf, constant.VersionKey, url)
	appendParam(bf, constant.GroupKey, url)
	appendParam(bf, constant.NacosProtocolKey, url)
	return bf.String()
}

func appendParam(buffer bytes.Buffer, paramKey string, url *common.URL) {
	buffer.WriteString(paramKey)
	buffer.WriteString("=")
	buffer.WriteString(url.GetParam(paramKey, ""))
}

func (s *ServiceDiscoveryRegistry) synthesizeSubscribedURLs(subscribedURL *common.URL, serviceInstances []registry.ServiceInstance) []*common.URL {
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

func (s *ServiceDiscoveryRegistry) getServices(url *common.URL, listener registry.MappingListener) *gxset.HashSet {
	services := gxset.NewSet()
	serviceNames := url.GetParam(constant.ProvidedBy, "")
	if len(serviceNames) > 0 {
		services = parseServices(serviceNames)
	}
	if services.Empty() {
		services = s.findMappedServices(url, listener)
		if services.Empty() {
			return s.subscribedServices
		}
	}
	return services
}

func (s *ServiceDiscoveryRegistry) findMappedServices(url *common.URL, listener registry.MappingListener) *gxset.HashSet {
	serviceNames, err := s.serviceNameMapping.Get(url, listener)
	if err != nil {
		logger.Errorf("get service names catch error, url:%s, err:%s ", url.String(), err.Error())
		return gxset.NewSet()
	}
	if listener != nil {
		protocolServiceKey := url.ServiceKey() + ":" + url.Protocol
		s.serviceMappingListeners[protocolServiceKey] = listener
	}
	return serviceNames
}

var (
	exporting = &atomic.Bool{}
)

func (s *ServiceDiscoveryRegistry) stopListen(url *common.URL) {
	protocolServiceKey := url.ServiceKey() + ":" + url.Protocol
	listener := s.serviceMappingListeners[protocolServiceKey]
	if listener != nil {
		delete(s.serviceMappingListeners, protocolServiceKey)
		listener.Stop()
	}
}
