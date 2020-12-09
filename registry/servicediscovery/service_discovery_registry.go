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
	"encoding/json"
	"strconv"
	"strings"
	"sync"
)

import (
	gxset "github.com/dubbogo/gost/container/set"
	perrors "github.com/pkg/errors"
	"go.uber.org/atomic"
)

import (
	"github.com/apache/dubbo-go/common"
	"github.com/apache/dubbo-go/common/constant"
	"github.com/apache/dubbo-go/common/extension"
	"github.com/apache/dubbo-go/common/logger"
	"github.com/apache/dubbo-go/common/observer"
	"github.com/apache/dubbo-go/config"
	"github.com/apache/dubbo-go/metadata/mapping"
	"github.com/apache/dubbo-go/metadata/service"
	"github.com/apache/dubbo-go/metadata/service/exporter/configurable"
	"github.com/apache/dubbo-go/registry"
	"github.com/apache/dubbo-go/registry/event"
	"github.com/apache/dubbo-go/registry/servicediscovery/synthesizer"
	"github.com/apache/dubbo-go/remoting"
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
	metaDataService, err := extension.GetMetadataService(config.GetApplicationConfig().MetadataType)
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
	return s.metaDataService.UnsubscribeURL(url)
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

func (s *serviceDiscoveryRegistry) GetUrl() *common.URL {
	return s.url
}

func (s *serviceDiscoveryRegistry) IsAvailable() bool {
	// TODO(whether available depends on metadata service and service discovery)
	return true
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

	err = s.metaDataService.PublishServiceDefinition(url)
	if err != nil {
		return perrors.WithMessage(err, "publish the service definition failed. ")
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
	for _, srv := range services.Values() {
		serviceName := srv.(string)
		serviceInstances := s.serviceDiscovery.GetInstances(serviceName)
		s.subscribe(url, notify, serviceName, serviceInstances)
		listener := &registry.ServiceInstancesChangedListener{
			ServiceName: serviceName,
			ChangedNotify: &InstanceChangeNotify{
				notify:                   notify,
				serviceDiscoveryRegistry: s,
			},
		}
		s.registerServiceInstancesChangedListener(url, listener)
	}
	return nil
}

func (s *serviceDiscoveryRegistry) registerServiceInstancesChangedListener(url *common.URL, listener *registry.ServiceInstancesChangedListener) {
	listenerId := listener.ServiceName + ":" + getUrlKey(url)
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

func (s *serviceDiscoveryRegistry) subscribe(url *common.URL, notify registry.NotifyListener,
	serviceName string, serviceInstances []registry.ServiceInstance) {
	if len(serviceInstances) == 0 {
		logger.Warnf("here is no instance in service[name : %s]", serviceName)
		return
	}
	var subscribedURLs []*common.URL
	subscribedURLs = append(subscribedURLs, s.getExportedUrls(url, serviceInstances)...)
	if len(subscribedURLs) == 0 {
		subscribedURLs = s.synthesizeSubscribedURLs(url, serviceInstances)
	}
	for _, url := range subscribedURLs {
		notify.Notify(&registry.ServiceEvent{
			Action:  remoting.EventTypeAdd,
			Service: url,
		})
	}

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

func (s *serviceDiscoveryRegistry) getExportedUrls(subscribedURL *common.URL, serviceInstances []registry.ServiceInstance) []*common.URL {
	var filterInstances []registry.ServiceInstance
	for _, s := range serviceInstances {
		if !s.IsEnable() || !s.IsHealthy() {
			continue
		}
		metaData := s.GetMetadata()
		_, ok1 := metaData[constant.METADATA_SERVICE_URL_PARAMS_PROPERTY_NAME]
		_, ok2 := metaData[constant.METADATA_SERVICE_URLS_PROPERTY_NAME]
		if !ok1 && !ok2 {
			continue
		}
		filterInstances = append(filterInstances, s)
	}
	if len(filterInstances) == 0 {
		return []*common.URL{}
	}
	s.prepareServiceRevisionExportedURLs(filterInstances)
	subscribedURLs := s.cloneExportedURLs(subscribedURL, filterInstances)
	return subscribedURLs
}

func (s *serviceDiscoveryRegistry) getExportedUrlsByInst(serviceInstance registry.ServiceInstance) []*common.URL {
	var urls []*common.URL
	metadataStorageType := getExportedStoreType(serviceInstance)
	proxyFactory := extension.GetMetadataServiceProxyFactory(metadataStorageType)
	if proxyFactory == nil {
		return urls
	}
	metadataService := proxyFactory.GetProxy(serviceInstance)
	if metadataService == nil {
		return urls
	}
	result, err := metadataService.GetExportedURLs(constant.ANY_VALUE, constant.ANY_VALUE, constant.ANY_VALUE, constant.ANY_VALUE)
	if err != nil {
		logger.Errorf("get exported urls catch error:%s,instance:%+v", err.Error(), serviceInstance)
		return urls
	}

	ret := make([]*common.URL, 0, len(result))
	for _, ui := range result {

		u, err := common.NewURL(ui.(string))

		if err != nil {
			logger.Errorf("could not parse the url string to URL structure: %s", ui.(string), err)
			continue
		}
		ret = append(ret, u)
	}
	return ret
}

func (s *serviceDiscoveryRegistry) prepareServiceRevisionExportedURLs(serviceInstances []registry.ServiceInstance) {
	s.lock.Lock()
	// 1. expunge stale
	s.expungeStaleRevisionExportedURLs(serviceInstances)
	// 2. Initialize
	s.initRevisionExportedURLs(serviceInstances)
	s.lock.Unlock()
}

func (s *serviceDiscoveryRegistry) expungeStaleRevisionExportedURLs(serviceInstances []registry.ServiceInstance) {
	serviceName := serviceInstances[0].GetServiceName()
	revisionExportedURLsMap, exist := s.serviceRevisionExportedURLsCache[serviceName]
	if !exist {
		return
	}
	existRevision := gxset.NewSet()
	for k := range revisionExportedURLsMap {
		existRevision.Add(k)
	}
	currentRevision := gxset.NewSet()
	for _, s := range serviceInstances {
		rv := getExportedServicesRevision(s)
		if len(rv) != 0 {
			currentRevision.Add(rv)
		}
	}
	// staleRevisions = existedRevisions(copy) - currentRevisions
	staleRevision := gxset.NewSet(existRevision.Values()...)
	staleRevision.Remove(currentRevision.Values()...)
	// remove exported URLs if staled
	for _, s := range staleRevision.Values() {
		delete(revisionExportedURLsMap, s.(string))
	}
}

func (s *serviceDiscoveryRegistry) initRevisionExportedURLs(serviceInstances []registry.ServiceInstance) {
	// initialize the revision exported URLs that the selected service instance exported
	s.initSelectedRevisionExportedURLs(serviceInstances)
	// initialize the revision exported URLs that other service instances exported
	for _, serviceInstance := range serviceInstances {
		s.initRevisionExportedURLsByInst(serviceInstance)
	}
}

func (s *serviceDiscoveryRegistry) initSelectedRevisionExportedURLs(serviceInstances []registry.ServiceInstance) {
	for range serviceInstances {
		selectServiceInstance := s.selectServiceInstance(serviceInstances)
		revisionExportedURLs := s.initRevisionExportedURLsByInst(selectServiceInstance)
		if len(revisionExportedURLs) > 0 {
			// If the result is valid,break
			break
		}
	}
}

func (s *serviceDiscoveryRegistry) selectServiceInstance(serviceInstances []registry.ServiceInstance) registry.ServiceInstance {
	size := len(serviceInstances)
	if size == 0 {
		return nil
	}
	if size == 1 {
		return serviceInstances[0]
	}
	selectorName := s.url.GetParam(constant.SERVICE_INSTANCE_SELECTOR, "random")
	selector, err := extension.GetServiceInstanceSelector(selectorName)
	if err != nil {
		logger.Errorf("get service instance selector cathe error:%s", err.Error())
		return nil
	}
	return selector.Select(s.url, serviceInstances)
}

func (s *serviceDiscoveryRegistry) initRevisionExportedURLsByInst(serviceInstance registry.ServiceInstance) []*common.URL {
	if serviceInstance == nil {
		return nil
	}
	serviceName := serviceInstance.GetServiceName()
	revision := getExportedServicesRevision(serviceInstance)
	revisionExportedURLsMap := s.serviceRevisionExportedURLsCache[serviceName]
	if revisionExportedURLsMap == nil {
		revisionExportedURLsMap = make(map[string][]*common.URL, 4)
		s.serviceRevisionExportedURLsCache[serviceName] = revisionExportedURLsMap
	}
	revisionExportedURLs := revisionExportedURLsMap[revision]
	firstGet := false
	if revisionExportedURLs == nil || len(revisionExportedURLs) == 0 {
		if len(revisionExportedURLsMap) > 0 {
			// The case is that current ServiceInstance with the different revision
			logger.Warnf("The ServiceInstance[id: %s, host : %s , port : %s] has different revision : %s"+
				", please make sure the service [name : %s] is changing or not.", serviceInstance.GetId(),
				serviceInstance.GetHost(), serviceInstance.GetPort(), revision, serviceInstance.GetServiceName())
		} else {
			firstGet = true
		}
		revisionExportedURLs = s.getExportedUrlsByInst(serviceInstance)
		if revisionExportedURLs != nil {
			revisionExportedURLsMap[revision] = revisionExportedURLs
			logger.Debugf("Get the exported URLs[size : %s, first : %s] from the target service "+
				"instance [id: %s , service : %s , host : %s , port : %s , revision : %s]",
				len(revisionExportedURLs), firstGet, serviceInstance.GetId(), serviceInstance.GetServiceName(),
				serviceInstance.GetHost(), serviceInstance.GetPort(), revision)
		}
	} else {
		// Else, The cache is hit
		logger.Debugf("Get the exported URLs[size : %s] from cache, the instance"+
			"[id: %s , service : %s , host : %s , port : %s , revision : %s]", len(revisionExportedURLs), firstGet,
			serviceInstance.GetId(), serviceInstance.GetServiceName(), serviceInstance.GetHost(),
			serviceInstance.GetPort(), revision)
	}
	return revisionExportedURLs
}

func getExportedServicesRevision(serviceInstance registry.ServiceInstance) string {
	metaData := serviceInstance.GetMetadata()
	return metaData[constant.EXPORTED_SERVICES_REVISION_PROPERTY_NAME]
}

func getExportedStoreType(serviceInstance registry.ServiceInstance) string {
	metaData := serviceInstance.GetMetadata()
	result, ok := metaData[constant.METADATA_STORAGE_TYPE_PROPERTY_NAME]
	if !ok {
		return constant.DEFAULT_METADATA_STORAGE_TYPE
	}
	return result
}

func (s *serviceDiscoveryRegistry) cloneExportedURLs(url *common.URL, serviceInsances []registry.ServiceInstance) []*common.URL {
	if len(serviceInsances) == 0 {
		return []*common.URL{}
	}
	var clonedExportedURLs []*common.URL
	removeParamSet := gxset.NewSet()
	removeParamSet.Add(constant.PID_KEY)
	removeParamSet.Add(constant.TIMESTAMP_KEY)
	for _, serviceInstance := range serviceInsances {
		templateExportURLs := s.getTemplateExportedURLs(url, serviceInstance)
		host := serviceInstance.GetHost()
		for _, u := range templateExportURLs {
			port := strconv.Itoa(getProtocolPort(serviceInstance, u.Protocol))
			if u.Location != host || u.Port != port {
				u.Port = port                  // reset port
				u.Location = host + ":" + port // reset host
			}

			cloneUrl := u.CloneExceptParams(removeParamSet)
			clonedExportedURLs = append(clonedExportedURLs, cloneUrl)
		}
	}
	return clonedExportedURLs

}

type endpoint struct {
	Port     int    `json:"port,omitempty"`
	Protocol string `json:"protocol,omitempty"`
}

func getProtocolPort(serviceInstance registry.ServiceInstance, protocol string) int {
	md := serviceInstance.GetMetadata()
	rawEndpoints := md[constant.SERVICE_INSTANCE_ENDPOINTS]
	if len(rawEndpoints) == 0 {
		return -1
	}
	var endpoints []endpoint
	err := json.Unmarshal([]byte(rawEndpoints), &endpoints)
	if err != nil {
		logger.Errorf("json umarshal rawEndpoints[%s] catch error:%s", rawEndpoints, err.Error())
		return -1
	}
	for _, e := range endpoints {
		if e.Protocol == protocol {
			return e.Port
		}
	}
	return -1
}
func (s *serviceDiscoveryRegistry) getTemplateExportedURLs(url *common.URL, serviceInstance registry.ServiceInstance) []*common.URL {
	exportedURLs := s.getRevisionExportedURLs(serviceInstance)
	if len(exportedURLs) == 0 {
		return []*common.URL{}
	}
	return filterSubscribedURLs(url, exportedURLs)
}

func (s *serviceDiscoveryRegistry) getRevisionExportedURLs(serviceInstance registry.ServiceInstance) []*common.URL {
	if serviceInstance == nil {
		return []*common.URL{}
	}
	serviceName := serviceInstance.GetServiceName()
	revision := getExportedServicesRevision(serviceInstance)
	s.lock.RLock()
	revisionExportedURLsMap, exist := s.serviceRevisionExportedURLsCache[serviceName]
	if !exist {
		return []*common.URL{}
	}
	exportedURLs, exist := revisionExportedURLsMap[revision]
	if !exist {
		return []*common.URL{}
	}
	s.lock.RUnlock()
	// Get a copy from source in order to prevent the caller trying to change the cached data
	cloneExportedURLs := make([]*common.URL, len(exportedURLs))
	copy(cloneExportedURLs, exportedURLs)
	return cloneExportedURLs
}

func filterSubscribedURLs(subscribedURL *common.URL, exportedURLs []*common.URL) []*common.URL {
	var filterExportedURLs []*common.URL
	for _, url := range exportedURLs {
		if url.GetParam(constant.INTERFACE_KEY, url.Path) != subscribedURL.GetParam(constant.INTERFACE_KEY, url.Path) {
			break
		}
		if url.GetParam(constant.VERSION_KEY, "") != subscribedURL.GetParam(constant.VERSION_KEY, "") {
			break
		}
		if url.GetParam(constant.GROUP_KEY, "") != subscribedURL.GetParam(constant.GROUP_KEY, "") {
			break
		}
		if len(subscribedURL.Protocol) != 0 {
			if subscribedURL.Protocol != url.Protocol {
				break
			}
		}
		filterExportedURLs = append(filterExportedURLs, url)
	}
	return filterExportedURLs
}

type InstanceChangeNotify struct {
	notify                   registry.NotifyListener
	serviceDiscoveryRegistry *serviceDiscoveryRegistry
}

func (icn *InstanceChangeNotify) Notify(event observer.Event) {

	if se, ok := event.(*registry.ServiceInstancesChangedEvent); ok {
		sdr := icn.serviceDiscoveryRegistry
		sdr.subscribe(sdr.url.SubURL, icn.notify, se.ServiceName, se.Instances)
	}
}

var (
	exporting = &atomic.Bool{}
)

// tryInitMetadataService will try to initialize metadata service
// TODO (move to somewhere)
func tryInitMetadataService(url *common.URL) {

	ms, err := extension.GetMetadataService(config.GetApplicationConfig().MetadataType)
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
