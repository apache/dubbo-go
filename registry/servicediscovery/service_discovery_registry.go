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
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/common/extension"
	"dubbo.apache.org/dubbo-go/v3/metadata/info"
	"dubbo.apache.org/dubbo-go/v3/metadata/mapping"
	reportInstance "dubbo.apache.org/dubbo-go/v3/metadata/report/instance"
	"dubbo.apache.org/dubbo-go/v3/metrics"
	metricsMetadata "dubbo.apache.org/dubbo-go/v3/metrics/metadata"
	metricsRegistry "dubbo.apache.org/dubbo-go/v3/metrics/registry"
	"dubbo.apache.org/dubbo-go/v3/registry"
	_ "dubbo.apache.org/dubbo-go/v3/registry/servicediscovery/customizer"
	"dubbo.apache.org/dubbo-go/v3/registry/servicediscovery/synthesizer"
)

func init() {
	extension.SetRegistry(constant.ServiceRegistryProtocol, newServiceDiscoveryRegistry)
}

// serviceDiscoveryRegistry is the implementation of application-level registry.
// It's completely different from other registry implementations
// This implementation is based on ServiceDiscovery abstraction and ServiceNameMapping and metadata
// In order to keep compatible with interface-level registry，
// serviceDiscoveryRegistry = ServiceDiscovery + metadata
type serviceDiscoveryRegistry struct {
	lock                    sync.RWMutex
	url                     *common.URL
	serviceDiscovery        registry.ServiceDiscovery
	metadataInfo            *info.MetadataInfo
	instance                registry.ServiceInstance
	serviceNameMapping      mapping.ServiceNameMapping
	serviceListeners        map[string]registry.ServiceInstancesChangedListener
	serviceMappingListeners map[string]mapping.MappingListener
}

func newServiceDiscoveryRegistry(url *common.URL) (registry.Registry, error) {
	serviceDiscovery, err := extension.GetServiceDiscovery(url)
	if err != nil {
		return nil, perrors.WithMessage(err, "Create service discovery failed")
	}
	return &serviceDiscoveryRegistry{
		url:                url,
		serviceDiscovery:   serviceDiscovery,
		metadataInfo:       info.NewMetadataInfo(),
		serviceNameMapping: extension.GetGlobalServiceNameMapping(),
		serviceListeners:   make(map[string]registry.ServiceInstancesChangedListener),
		// cache for mapping listener
		serviceMappingListeners: make(map[string]mapping.MappingListener),
	}, nil
}

func (s *serviceDiscoveryRegistry) RegisterService() error {
	s.instance = createInstance(s.metadataInfo)
	// consumer has no host and port, so it will not register service
	if s.instance.GetHost() != "" && s.instance.GetPort() != 0 {
		meta := s.metadataInfo
		meta.CalAndGetRevision()
		metadataReport := reportInstance.GetMetadataReport()
		if metadataReport != nil {
			err := metadataReport.PublishAppMetadata(meta.App, meta.Revision, meta)
			if err != nil {
				return err
			}
		}
		return s.serviceDiscovery.Register(s.instance)
	}
	return nil
}

func createInstance(meta *info.MetadataInfo) registry.ServiceInstance {
	metadata := make(map[string]string, 8)
	metadata[constant.MetadataStorageTypePropertyName] = reportInstance.GetMetadataType()
	instance := &registry.DefaultServiceInstance{
		ServiceName:     meta.App,
		Enable:          true,
		Healthy:         true,
		Metadata:        metadata,
		ServiceMetadata: meta,
	}

	for _, cus := range extension.GetCustomizers() {
		cus.Customize(instance)
	}
	return instance
}

func (s *serviceDiscoveryRegistry) UnRegisterService() error {
	return s.serviceDiscovery.Unregister(s.instance)
}

func (s *serviceDiscoveryRegistry) GetLocalMetadata() *info.MetadataInfo {
	return s.metadataInfo
}

func (s *serviceDiscoveryRegistry) UnRegister(url *common.URL) error {
	if !shouldRegister(url) {
		return nil
	}
	s.GetLocalMetadata().RemoveService(url)
	return nil
}

func (s *serviceDiscoveryRegistry) UnSubscribe(url *common.URL, listener registry.NotifyListener) error {
	if !shouldSubscribe(url) {
		return nil
	}
	s.GetLocalMetadata().RemoveSubscribeURL(url)
	services := s.getServices(url, nil)
	if services == nil {
		return nil
	}
	// FIXME ServiceNames.String() is not good
	serviceNamesKey := services.String()
	l := s.serviceListeners[serviceNamesKey]
	l.RemoveListener(url.ServiceKey())
	s.stopListen(url)
	err := s.serviceNameMapping.Remove(url) // TODO check is right
	if err != nil {
		return err
	}
	return nil
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
	common.HandleRegisterIPAndPort(url)
	s.GetLocalMetadata().AddService(url)
	metrics.Publish(metricsRegistry.NewServerRegisterEvent(true, time.Now()))
	return s.serviceNameMapping.Map(url)
}

func shouldRegister(url *common.URL) bool {
	side := url.GetParam(constant.SideKey, "")
	if side == constant.SideProvider {
		return true
	}
	logger.Debugf("The URL should not be register.", url.String())
	return false
}

func (s *serviceDiscoveryRegistry) Subscribe(url *common.URL, notify registry.NotifyListener) error {
	if !shouldSubscribe(url) {
		logger.Infof("Service %s is set to not subscribe to instances.", url.ServiceKey())
		return nil
	}
	s.GetLocalMetadata().AddSubscribeURL(url)
	mappingListener := NewMappingListener(s.url, url, parseServices(url.GetParam(constant.ProvidedBy, "")), notify)
	services := s.getServices(url, mappingListener)
	if services.Empty() {
		return perrors.Errorf("Should has at least one way to know which services this interface belongs to,"+
			" either specify 'provided-by' for reference or enable metadata-report center subscription url:%s", url.String())
	}
	logger.Infof("Find initial mapping applications %q for service %s.", services, url.ServiceKey())
	// first notify
	mappingListener.OnEvent(registry.NewServiceMappingChangedEvent(url.ServiceKey(), services))
	return nil
}

func (s *serviceDiscoveryRegistry) SubscribeURL(url *common.URL, notify registry.NotifyListener, services *gxset.HashSet) {
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
				logger.Warnf("[serviceDiscoveryRegistry] ServiceInstancesChangedListenerImpl handle error:%v", err)
			}
		}
	}
	s.serviceListeners[serviceNamesKey] = listener
	listener.AddListenerAndNotify(protocolServiceKey, notify)
	event := metricsMetadata.NewMetadataMetricTimeEvent(metricsMetadata.SubscribeServiceRt)

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
func (s *serviceDiscoveryRegistry) LoadSubscribeInstances(url *common.URL, notify registry.NotifyListener) error {
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

func (s *serviceDiscoveryRegistry) synthesizeSubscribedURLs(subscribedURL *common.URL, serviceInstances []registry.ServiceInstance) []*common.URL {
	var urls []*common.URL
	for _, syn := range synthesizer.GetAllSynthesizer() {
		if syn.Support(subscribedURL) {
			urls = append(urls, syn.Synthesize(subscribedURL, serviceInstances)...)
		}
	}
	return urls
}

func shouldSubscribe(url *common.URL) bool {
	return !shouldRegister(url)
}

func (s *serviceDiscoveryRegistry) getServices(url *common.URL, listener mapping.MappingListener) *gxset.HashSet {
	services := gxset.NewSet()
	serviceNames := url.GetParam(constant.ProvidedBy, "")
	if len(serviceNames) > 0 {
		services = parseServices(serviceNames)
	}
	if services.Empty() {
		services = s.findMappedServices(url, listener)
	}
	return services
}

func (s *serviceDiscoveryRegistry) findMappedServices(url *common.URL, listener mapping.MappingListener) *gxset.HashSet {
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

func (s *serviceDiscoveryRegistry) stopListen(url *common.URL) {
	protocolServiceKey := url.ServiceKey() + ":" + url.Protocol
	listener := s.serviceMappingListeners[protocolServiceKey]
	if listener != nil {
		delete(s.serviceMappingListeners, protocolServiceKey)
		listener.Stop()
	}
}
