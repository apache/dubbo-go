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
	"context"
	"errors"
	"fmt"
	"math/rand/v2"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

import (
	gxset "github.com/dubbogo/gost/container/set"
	"github.com/dubbogo/gost/log/logger"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/common/extension"
	"dubbo.apache.org/dubbo-go/v3/metadata"
	"dubbo.apache.org/dubbo-go/v3/metadata/info"
	"dubbo.apache.org/dubbo-go/v3/metadata/mapping"
	"dubbo.apache.org/dubbo-go/v3/metadata/report"
	"dubbo.apache.org/dubbo-go/v3/metrics"
	metricsMetadata "dubbo.apache.org/dubbo-go/v3/metrics/metadata"
	metricsRegistry "dubbo.apache.org/dubbo-go/v3/metrics/registry"
	"dubbo.apache.org/dubbo-go/v3/registry"
	_ "dubbo.apache.org/dubbo-go/v3/registry/servicediscovery/customizer"
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
	ctx                     context.Context
	cancel                  context.CancelFunc
	lock                    sync.RWMutex
	url                     *common.URL
	serviceDiscovery        registry.ServiceDiscovery
	instances               []registry.ServiceInstance
	instanceURLs            map[registry.ServiceInstance]*common.URL
	serviceNameMapping      mapping.ServiceNameMapping
	metadataReport          report.MetadataReport
	serviceListeners        map[string]registry.ServiceInstancesChangedListener
	serviceMappingListeners map[string]mapping.MappingListener
	// subscribeRetries holds at most one pending AddListener retry per
	// serviceNamesKey. Guarded by lock.
	subscribeRetries map[string]*subscribeRetry
	// definitionRetries holds at most one pending service-definition publish
	// retry per service key. Guarded by lock.
	definitionRetries map[string]*definitionRetry
	// Full definition publishes are coalesced behind one worker per registry.
	// The state is guarded by lock; definitionPublishMu serializes the backend
	// call with per-service retries as well.
	definitionPublishRunning bool
	definitionPublishPending []*common.URL
	definitionPublishMu      sync.Mutex
	renewAppMetadataTimer    *time.Timer
	// destroyed is set by Destroy. SubscribeURL re-checks it under lock right
	// before installing a listener, so a subscribe whose initial GetInstances /
	// metadata phase outlives Destroy is discarded instead of being installed
	// into a dead registry, and late AddListener failures cannot arm new retry
	// timers. Guarded by lock.
	destroyed bool
}

func newServiceDiscoveryRegistry(url *common.URL) (registry.Registry, error) {
	serviceDiscovery, err := extension.GetServiceDiscovery(url)
	if err != nil {
		return nil, fmt.Errorf("create service discovery failed: %w", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	return &serviceDiscoveryRegistry{
		ctx:                ctx,
		cancel:             cancel,
		url:                url,
		serviceDiscovery:   serviceDiscovery,
		instanceURLs:       make(map[registry.ServiceInstance]*common.URL),
		serviceNameMapping: extension.GetGlobalServiceNameMapping(),
		metadataReport:     metadata.GetMetadataReportByRegistry(url.GetParam(constant.RegistryIdKey, "")),
		serviceListeners:   make(map[string]registry.ServiceInstancesChangedListener),
		subscribeRetries:   make(map[string]*subscribeRetry),
		definitionRetries:  make(map[string]*definitionRetry),
		// cache for mapping listener
		serviceMappingListeners: make(map[string]mapping.MappingListener),
	}, nil
}

func isPublishableRevision(revision string) bool {
	return len(revision) > 0 && revision != "0" && revision != "N/A"
}

// startMetadataTimers starts the daily renew timer when there is anything for
// it to refresh. GC runs after each renew cycle inside doRenewAppMetadata.
//
// Two independent things want the timer. Remote application metadata needs
// re-publishing, and so do interface-level service definitions — which are
// published regardless of metadataType, so the timer must start for them even
// when application metadata lives locally. Each half guards itself inside
// doRenewAppMetadata.
func (s *serviceDiscoveryRegistry) startMetadataTimers() {
	if s.metadataReport == nil {
		return
	}
	if !renewsAppMetadata() && !metadata.ServiceDefinitionsEnabled(s.metadataReport) {
		return
	}
	metaInfo := metadata.GetMetadataInfo(s.url.GetParam(constant.RegistryIdKey, ""))
	if metaInfo == nil || !isPublishableRevision(metaInfo.Revision) {
		return
	}
	s.startRenewAppMetadataTimer()
}

// renewsAppMetadata reports whether application metadata lives in the metadata
// center, and therefore needs the daily re-publish.
func renewsAppMetadata() bool {
	return metadata.GetMetadataType() == constant.RemoteMetadataStorageType
}

func (s *serviceDiscoveryRegistry) RegisterService() error {
	registryId := s.url.GetParam(constant.RegistryIdKey, constant.DefaultKey)
	metaInfo := metadata.GetMetadataInfo(registryId)
	if metaInfo == nil {
		panic("no metada info found of registry id " + registryId)
	}
	urls := metaInfo.GetExportedServiceURLs()
	if len(urls) == 0 {
		return nil
	}

	instances := make([]registry.ServiceInstance, 0, len(urls))
	instanceURLs := make(map[registry.ServiceInstance]*common.URL)
	for _, url := range urls {
		instance := createInstance(metaInfo, url, registryId)
		metaInfo.Revision = instance.GetMetadata()[constant.ExportedServicesRevisionPropertyName]
		instances = append(instances, instance)
		instanceURLs[instance] = url
	}

	if metadata.GetMetadataType() == constant.RemoteMetadataStorageType {
		if s.metadataReport == nil {
			return fmt.Errorf("metadata_report failed: operation=publish app=%s revision=%s registry_id=%s storage_type=%s: no metadata report instance found",
				metaInfo.App, metaInfo.Revision, registryId, constant.RemoteMetadataStorageType)
		}
		if err := s.metadataReport.PublishAppMetadata(metaInfo.App, metaInfo.Revision, metaInfo); err != nil {
			return err
		}
		logger.Infof("[Metadata][Publish] published app metadata, app=%s revision=%s urls=%d",
			metaInfo.App, metaInfo.Revision, len(urls))
	}

	for _, instance := range instances {
		err := s.serviceDiscovery.Register(instance)
		if err != nil {
			return fmt.Errorf("register service failed: %w", err)
		}
		s.lock.Lock()
		s.instances = append(s.instances, instance)
		s.instanceURLs[instance] = instanceURLs[instance]
		s.lock.Unlock()
	}

	// Interface-level service definitions are a separate data path from the
	// application-level metadata above: they describe the RPC contract rather
	// than what this instance currently exports, and Admin discovers them by
	// their own key. They are therefore published regardless of metadataType,
	// which only governs where application metadata lives.
	//
	// Scheduled rather than run inline, and after registration rather than
	// before, so the metadata center is never in the path of an instance
	// entering traffic. Failures retry with backoff; the daily cycle report is
	// the backstop past that.
	s.scheduleServiceDefinitionPublish(urls)

	s.lock.Lock()
	if s.renewAppMetadataTimer == nil {
		s.startMetadataTimers()
	}
	s.lock.Unlock()

	return nil
}

func createInstance(meta *info.MetadataInfo, url *common.URL, registryId string) registry.ServiceInstance {
	params := make(map[string]string, 8)
	params[constant.MetadataStorageTypePropertyName] = metadata.GetMetadataType()
	// Expose the registry this instance belongs to so that customizers (e.g. revision
	// calculators) can scope their work to the correct per-registry service set.
	params[constant.RegistryIdKey] = registryId
	// Keep routing attributes visible on the registered instance as well as in service metadata.
	if environment := url.GetParam(constant.EnvironmentKey, ""); len(environment) > 0 {
		params[constant.EnvironmentKey] = environment
	}
	port, err := strconv.Atoi(url.Port)
	if err != nil {
		logger.Warnf("[Registry][ServiceDiscovery] parse port %s failed, err=%v", url.Port, err)
	}
	instance := &registry.DefaultServiceInstance{
		ID:              url.Address(),
		Host:            url.Ip,
		Port:            port,
		ServiceName:     meta.App,
		Enable:          true,
		Healthy:         true,
		Metadata:        params,
		ServiceMetadata: meta,
		Tag:             meta.Tag,
	}
	for _, cus := range extension.GetCustomizers() {
		cus.Customize(instance)
	}
	return instance
}

func (s *serviceDiscoveryRegistry) UnRegisterService() error {
	return s.unregisterService(nil)
}

func (s *serviceDiscoveryRegistry) unregisterService(targetURL *common.URL) error {
	s.lock.Lock()
	keep := s.instances[:0]
	origin := s.instances[:]
	s.lock.Unlock()

	var errs []error
	keepInstanceURLs := make(map[registry.ServiceInstance]*common.URL)

	for _, v := range origin {
		if err := s.serviceDiscovery.Unregister(v); err != nil {
			// fail to unregister
			keep = append(keep, v)
			errs = append(errs, err)
			s.lock.RLock()
			if sourceURL, ok := s.instanceURLs[v]; ok {
				keepInstanceURLs[v] = sourceURL
			}
			s.lock.RUnlock()
		}
	}

	s.lock.Lock()
	s.instances = keep
	s.instanceURLs = keepInstanceURLs
	s.lock.Unlock()
	if err := s.syncExportedMetadataAfterUnregister(targetURL, origin, keep); err != nil {
		errs = append(errs, err)
	}
	return errors.Join(errs...)
}

func (s *serviceDiscoveryRegistry) UnRegister(url *common.URL) error {
	if !shouldRegister(url) {
		return nil
	}
	return s.unregisterService(url)
}

func (s *serviceDiscoveryRegistry) UnSubscribe(url *common.URL, listener registry.NotifyListener) error {
	if !shouldSubscribe(url) {
		return nil
	}
	services := s.getServices(url, nil)
	if services == nil {
		return nil
	}
	serviceNamesKey := sortServices(services)
	if l := s.getServiceListener(serviceNamesKey); l != nil {
		// Must match the key SubscribeURL used for AddListenerAndNotify,
		// otherwise the entry leaks and subscriber tracking breaks.
		l.RemoveListener(protocolSubscribeKey(url))
		if !listenerHasSubscribers(l) {
			// Last subscriber left: stop retrying AddListener for this key.
			s.cancelSubscribeRetry(serviceNamesKey)
		}
	}
	s.stopListen(url)
	err := s.serviceNameMapping.Remove(url)
	if err != nil {
		return err
	}
	if id, exist := s.url.GetNonDefaultParam(constant.RegistryIdKey); exist {
		metadata.RemoveSubscribeURL(id, url)
	}
	return nil
}

func (s *serviceDiscoveryRegistry) syncExportedMetadataAfterUnregister(targetURL *common.URL, origin []registry.ServiceInstance, keep []registry.ServiceInstance) error {
	registryId, exist := s.url.GetNonDefaultParam(constant.RegistryIdKey)
	if !exist {
		return nil
	}
	metadataInfo := metadata.GetMetadataInfo(registryId)
	if metadataInfo == nil {
		return nil
	}

	keepURLs := s.getInstanceURLs(keep)
	if len(origin) > 0 {
		metadataInfo.ReplaceExportedServices(keepURLs)
	} else if targetURL != nil {
		metadata.RemoveService(registryId, targetURL)
	}
	remainingURLs := metadataInfo.GetExportedServiceURLs()
	if len(remainingURLs) == 0 {
		metadataInfo.Revision = "0"
		return nil
	}
	instance := createInstance(metadataInfo, remainingURLs[0], registryId)
	revision := instance.GetMetadata()[constant.ExportedServicesRevisionPropertyName]
	metadataInfo.Revision = revision
	if len(keepURLs) == 0 {
		return nil
	}
	if metadata.GetMetadataType() == constant.RemoteMetadataStorageType {
		if s.metadataReport == nil {
			return fmt.Errorf("metadata_report failed: operation=publish app=%s revision=%s registry_id=%s storage_type=%s: no metadata report instance found",
				metadataInfo.App, revision, registryId, constant.RemoteMetadataStorageType)
		}
		if err := s.metadataReport.PublishAppMetadata(metadataInfo.App, revision, metadataInfo); err != nil {
			return err
		}
	}
	for _, keepInstance := range keep {
		keepInstance.SetServiceMetadata(metadataInfo)
		keepInstance.GetMetadata()[constant.ExportedServicesRevisionPropertyName] = revision
		if err := s.serviceDiscovery.Update(keepInstance); err != nil {
			return fmt.Errorf("update service failed: %w", err)
		}
	}
	return nil
}

func (s *serviceDiscoveryRegistry) getInstanceURLs(instances []registry.ServiceInstance) []*common.URL {
	urls := make([]*common.URL, 0, len(instances))
	s.lock.RLock()
	defer s.lock.RUnlock()
	for _, instance := range instances {
		if sourceURL, ok := s.instanceURLs[instance]; ok {
			urls = append(urls, sourceURL)
		}
	}
	return urls
}

func parseServices(literalServices string) *gxset.HashSet {
	set := gxset.NewSet()
	if len(literalServices) == 0 {
		return set
	}
	splitServices := strings.SplitSeq(literalServices, ",")
	for s := range splitServices {
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
	if s.cancel != nil {
		s.cancel()
	}
	s.stopMetadataTimers()
	s.lock.Lock()
	s.destroyed = true
	for key := range s.subscribeRetries {
		// Cancel pending AddListener retries so their timers cannot leak.
		s.cancelSubscribeRetryLocked(key)
	}
	// Same for pending definition publish retries and queued full publishes.
	s.cancelDefinitionRetriesLocked()
	s.cancelDefinitionPublishLocked()
	for _, l := range s.serviceListeners {
		// Destroy drops listeners without RemoveListener; cancel any pending
		// metadata retry so its timer cannot leak.
		if impl, ok := l.(*ServiceInstancesChangedListenerImpl); ok {
			impl.stopMetadataRetry()
		}
	}
	s.lock.Unlock()
	err := s.serviceDiscovery.Destroy()
	if err != nil {
		logger.Errorf("[Registry][ServiceDiscovery] destroy serviceDiscovery catch error, err=%s", err.Error())
	}
}

func (s *serviceDiscoveryRegistry) stopMetadataTimers() {
	s.lock.Lock()
	defer s.lock.Unlock()
	if s.renewAppMetadataTimer != nil {
		s.renewAppMetadataTimer.Stop()
		s.renewAppMetadataTimer = nil
	}
}

// ========== service definitions: publish with bounded retry ==========

var (
	// definitionRetryInitialDelay is the first backoff delay before retrying a
	// failed definition publish. Package-level so tests can shrink it.
	definitionRetryInitialDelay = time.Second
	// definitionRetryMaxDelay caps the backoff.
	definitionRetryMaxDelay = 30 * time.Second
	// definitionRetryMaxAttempts bounds the retries, unlike the deliberately
	// unlimited subscribe retry next door. The two differ because their
	// backstops do: a subscription that never establishes leaves a permanently
	// stale consumer and nothing else will fix it, whereas a definition that
	// never publishes is re-attempted by the daily cycle report. Retrying past
	// the point where the metadata center is plainly not coming back would only
	// add log noise to a failure the daily pass already owns.
	//
	// With the delays above this spans roughly four minutes, which covers the
	// case this exists for: the metadata center bouncing while a provider
	// happens to start.
	definitionRetryMaxAttempts = 10

	// publishDefinitions indirects the publish call so tests can drive the
	// retry loop without a live metadata center.
	publishDefinitions = metadata.PublishServiceDefinitions
)

// definitionRetry is a pending or in-flight definition publish retry for one
// service key. Mirrors subscribeRetry: the entry stays in the map while an
// attempt is in flight so a concurrent failure cannot reset the backoff.
type definitionRetry struct {
	url      *common.URL
	timer    *time.Timer
	attempts int
	inFlight bool
	canceled bool
}

// publishServiceDefinitions publishes definitions for urls and arms a retry for
// whatever failed to reach the metadata center.
//
// Synchronous. The full-publish worker uses this after removing the work from
// the registration path; callers on that path must use
// scheduleServiceDefinitionPublish instead.
func (s *serviceDiscoveryRegistry) publishServiceDefinitions(urls []*common.URL) {
	for _, u := range s.publishServiceDefinitionsNow(urls) {
		s.scheduleDefinitionRetry(u)
	}
}

// publishServiceDefinitionsNow serializes every backend definition write for
// this registry, including full publishes and per-service retries.
//
// Destroy deliberately does not wait for a call that has already entered the
// backend: the Nacos API has no context/cancellation parameter. The call may
// finish within the backend timeout, but no retry or queued full publish will
// follow it.
func (s *serviceDiscoveryRegistry) publishServiceDefinitionsNow(urls []*common.URL) []*common.URL {
	s.definitionPublishMu.Lock()
	defer s.definitionPublishMu.Unlock()

	s.lock.RLock()
	destroyed := s.destroyed
	reportInstance := s.metadataReport
	s.lock.RUnlock()
	if destroyed {
		return nil
	}
	return publishDefinitions(reportInstance, urls)
}

// scheduleServiceDefinitionPublish hands a full publish to the registry's
// single worker instead of running it inline.
//
// Publishing reaches the metadata center synchronously, once per service. On the
// registration path that cost lands between "the process is ready" and "the
// instance is discoverable": a slow or unreachable Nacos delays every service in
// turn, holding an otherwise healthy provider out of traffic for as long as the
// writes take. Definitions are console metadata, so nothing about them justifies
// that.
//
// Repeated schedules are coalesced to the latest full URL snapshot. Destroy
// discards a snapshot that has not started, while an already-running backend
// call is allowed to finish without blocking shutdown. Failures land in the
// same per-service retry as any other publish.
func (s *serviceDiscoveryRegistry) scheduleServiceDefinitionPublish(urls []*common.URL) {
	s.lock.Lock()
	if s.destroyed {
		s.lock.Unlock()
		return
	}
	s.definitionPublishPending = append([]*common.URL(nil), urls...)
	if s.definitionPublishRunning {
		s.lock.Unlock()
		return
	}
	s.definitionPublishRunning = true
	s.lock.Unlock()

	go s.runDefinitionPublishWorker()
}

func (s *serviceDiscoveryRegistry) runDefinitionPublishWorker() {
	for {
		s.lock.Lock()
		if s.destroyed || len(s.definitionPublishPending) == 0 {
			s.definitionPublishPending = nil
			s.definitionPublishRunning = false
			s.lock.Unlock()
			return
		}
		urls := s.definitionPublishPending
		s.definitionPublishPending = nil
		s.lock.Unlock()

		s.publishServiceDefinitions(urls)
	}
}

// cancelDefinitionPublishLocked drops a full snapshot that has not started.
// Caller must hold s.lock. An in-flight backend call cannot be interrupted, but
// the worker observes destroyed and exits without consuming another snapshot.
func (s *serviceDiscoveryRegistry) cancelDefinitionPublishLocked() {
	s.definitionPublishPending = nil
}

// scheduleDefinitionRetry arms the retry timer for one service. One pending
// retry per service key: repeated failures share the same state rather than
// stacking new ones, so the backoff is not reset by a later attempt.
func (s *serviceDiscoveryRegistry) scheduleDefinitionRetry(u *common.URL) {
	key := u.ServiceKey()

	s.lock.Lock()
	defer s.lock.Unlock()
	if s.destroyed {
		return
	}
	if _, pending := s.definitionRetries[key]; pending {
		return
	}
	if s.definitionRetries == nil {
		// A registry built as a struct literal rather than through
		// newServiceDiscoveryRegistry still has to be safe to publish from.
		s.definitionRetries = make(map[string]*definitionRetry)
	}
	state := &definitionRetry{url: u}
	s.definitionRetries[key] = state
	s.armDefinitionRetryLocked(key, state)
}

// armDefinitionRetryLocked computes the next backoff delay and arms the timer;
// caller must hold s.lock and the state must be in the map.
func (s *serviceDiscoveryRegistry) armDefinitionRetryLocked(key string, state *definitionRetry) {
	if state.attempts >= definitionRetryMaxAttempts {
		logger.Warnf("[Metadata][Definition] giving up on publishing %s after %d attempts; "+
			"the daily cycle report will try again", key, state.attempts)
		delete(s.definitionRetries, key)
		return
	}

	delay := definitionRetryDelay(state.attempts)
	state.attempts++
	state.timer = time.AfterFunc(delay, func() {
		s.retryPublishDefinition(key)
	})
	logger.Debugf("[Metadata][Definition] definition for %s not published, retry in %s", key, delay)
}

func (s *serviceDiscoveryRegistry) retryPublishDefinition(key string) {
	s.lock.Lock()
	state, ok := s.definitionRetries[key]
	if !ok {
		s.lock.Unlock()
		return
	}
	state.timer = nil
	state.inFlight = true
	active := !s.destroyed && !state.canceled
	s.lock.Unlock()

	if !active {
		s.lock.Lock()
		if s.definitionRetries[key] == state {
			delete(s.definitionRetries, key)
		}
		s.lock.Unlock()
		return
	}

	failed := s.publishServiceDefinitionsNow([]*common.URL{state.url})

	s.lock.Lock()
	defer s.lock.Unlock()
	state.inFlight = false
	if s.definitionRetries[key] != state {
		// Superseded or canceled while in flight.
		return
	}
	if len(failed) == 0 || s.destroyed || state.canceled {
		delete(s.definitionRetries, key)
		return
	}
	// Re-arm in place: the entry never left the map, so the backoff continues
	// from where it was instead of restarting.
	s.armDefinitionRetryLocked(key, state)
}

// cancelDefinitionRetriesLocked stops every pending definition retry; caller
// must hold s.lock. An in-flight attempt is not interrupted but will not
// reschedule.
func (s *serviceDiscoveryRegistry) cancelDefinitionRetriesLocked() {
	for key, state := range s.definitionRetries {
		state.canceled = true
		if state.timer != nil {
			state.timer.Stop()
			state.timer = nil
		}
		if !state.inFlight {
			delete(s.definitionRetries, key)
		}
	}
}

// definitionRetryDelay grows exponentially from definitionRetryInitialDelay,
// capped at definitionRetryMaxDelay, plus up to 25% jitter so providers
// restarting together do not retry in lockstep.
func definitionRetryDelay(attempt int) time.Duration {
	delay := definitionRetryMaxDelay
	if attempt >= 0 && attempt < 30 { // guard against shift overflow
		delay = definitionRetryInitialDelay << attempt
		if delay <= 0 || delay > definitionRetryMaxDelay {
			delay = definitionRetryMaxDelay
		}
	}
	return delay + time.Duration(rand.Int64N(int64(delay/4)+1))
}

// ========== renewAppMetadata: daily app-level metadata re-publish ==========

// metadataReportURL returns the URL from the metadata report instance.
func (s *serviceDiscoveryRegistry) metadataReportURL() *common.URL {
	if s.metadataReport == nil {
		return nil
	}
	return s.metadataReport.URL()
}

func (s *serviceDiscoveryRegistry) startRenewAppMetadataTimer() {
	reportURL := s.metadataReportURL()
	if reportURL == nil || !reportURL.GetParamBool(constant.CycleReportKey, true) {
		return
	}

	// Run immediately on start
	if reportURL.GetParamBool(constant.MetadataRenewOnStartupKey, true) {
		go s.doRenewAppMetadata()
	}

	delay := s.calculateRenewAppMetadataDelay()
	s.renewAppMetadataTimer = time.AfterFunc(delay, func() {
		s.doRenewAppMetadata()
		// Reschedule for next day
		s.lock.Lock()
		if s.renewAppMetadataTimer != nil {
			s.renewAppMetadataTimer.Reset(24 * time.Hour)
		}
		s.lock.Unlock()
	})
}

func (s *serviceDiscoveryRegistry) doRenewAppMetadata() {
	registryID := s.url.GetParam(constant.RegistryIdKey, "")
	metaInfo := metadata.GetMetadataInfo(registryID)
	if metaInfo == nil || !isPublishableRevision(metaInfo.Revision) {
		return
	}

	if renewsAppMetadata() {
		// Copy snapshot to avoid data race
		snapshot := metaInfo.Snapshot()
		snapshot.LastUpdatedTime = time.Now().UnixMilli()
		if err := s.metadataReport.PublishAppMetadata(snapshot.App, snapshot.Revision, &snapshot); err != nil {
			logger.Errorf("[Metadata][renewAppMetadata] failed to re-publish metadata for app=%s revision=%s: %v", snapshot.App, snapshot.Revision, err)
		} else {
			logger.Infof("[Metadata][renewAppMetadata] refreshed metadata for app=%s revision=%s", snapshot.App, snapshot.Revision)
		}

		// Run garbage collection if enabled, after each renew cycle. It reasons
		// about application metadata revisions, so it only applies here.
		reportURL := s.metadataReportURL()
		if reportURL != nil && reportURL.GetParamBool(constant.MetadataGCEnabledKey, true) {
			s.doGarbageCollect()
		}
	}

	// Re-publish interface-level definitions.
	//
	// Nothing ever deletes a definition — its key holds no instance and no
	// revision, and a provider killed with SIGKILL gets no cleanup hook — so
	// operators need some way to tell a live contract from an abandoned one.
	// This pass is that way: every service a running provider still exports
	// gets its timestamp refreshed daily, while a service that was removed from
	// the code is never republished and its timestamp freezes. "Last updated
	// more than about two days ago and no live instance" then becomes a safe
	// death test.
	//
	// Without the refresh, never-deleting would be an unbounded leak with no
	// way to distinguish the garbage. Java relies on exactly this property, via
	// AbstractMetadataReport's daily publishAll over the same 02:00–06:00
	// window.
	s.scheduleServiceDefinitionPublish(metaInfo.GetExportedServiceURLs())
}

func (s *serviceDiscoveryRegistry) calculateRenewAppMetadataDelay() time.Duration {
	now := time.Now()
	// Next day 2:00 AM
	nextDay2AM := time.Date(now.Year(), now.Month(), now.Day()+1, 2, 0, 0, 0, now.Location())
	// Add random offset 0~4 hours to avoid thundering herd
	randomOffset := time.Duration(rand.Int64N(int64(4 * time.Hour)))
	return time.Until(nextDay2AM) + randomOffset
}

// ========== GC: stale revision cleanup ==========

func (s *serviceDiscoveryRegistry) doGarbageCollect() {
	registryID := s.url.GetParam(constant.RegistryIdKey, "")
	metaInfo := metadata.GetMetadataInfo(registryID)
	if metaInfo == nil {
		return
	}
	app := metaInfo.App
	if app == "" {
		return
	}

	// Step 1: List all revisions for this app
	revisions, err := s.metadataReport.ListAppRevisions(app)
	if err != nil {
		logger.Warnf("[Metadata][GC] failed to list app revisions: %v", err)
		return
	}
	if len(revisions) == 0 {
		return
	}

	// Step 2: Filter stale candidates (exceed GC window in days)
	reportURL := s.metadataReportURL()
	if reportURL == nil {
		return
	}
	gcWindowDays := reportURL.GetParamByIntValue(constant.MetadataGCWindowKey, 5)
	if gcWindowDays <= 0 || gcWindowDays > 365 {
		gcWindowDays = 5
	}
	cutoff := time.Now().AddDate(0, 0, -gcWindowDays).UnixMilli()
	candidates := make(map[string]bool)
	for _, rev := range revisions {
		// Skip special revisions
		if rev.Revision == "0" || rev.Revision == "N/A" || rev.Revision == "" || rev.Revision == metaInfo.Revision {
			continue
		}
		// ModifyTime == 0 means metadata produced by versions that did not set lastUpdatedTime.
		// Since we can't reliably determine staleness for those entries, skip GC for them.
		if rev.ModifyTime > 0 && rev.ModifyTime < cutoff {
			candidates[rev.Revision] = true
		}
	}
	if len(candidates) == 0 {
		return
	}

	// Step 3: Get alive instances and their revisions
	instances := s.serviceDiscovery.GetInstances(app)
	aliveRevisions := make(map[string]bool)
	for _, inst := range instances {
		metadata := inst.GetMetadata()
		if metadata == nil {
			continue
		}
		rev := metadata[constant.ExportedServicesRevisionPropertyName]
		if rev != "" {
			aliveRevisions[rev] = true
		}
	}

	// Step 4: Clean up stale revisions not referenced by any alive instance
	for rev := range candidates {
		if aliveRevisions[rev] {
			continue // still referenced, skip
		}
		logger.Infof("[Metadata][GC] cleaning up stale revision: app=%s revision=%s", app, rev)
		if err := s.metadataReport.UnPublishAppMetadata(app, rev); err != nil {
			logger.Warnf("[Metadata][GC] failed to unpublish revision %s: %v", rev, err)
		}
	}
}

func (s *serviceDiscoveryRegistry) Register(url *common.URL) error {
	if !shouldRegister(url) {
		return nil
	}
	common.HandleRegisterIPAndPort(url)
	if id, exist := s.url.GetNonDefaultParam(constant.RegistryIdKey); exist {
		metadata.AddService(id, url)
	}
	metrics.Publish(metricsRegistry.NewServerRegisterEvent(true, time.Now()))
	return s.serviceNameMapping.Map(url)
}

func shouldRegister(url *common.URL) bool {
	side := url.GetParam(constant.SideKey, "")
	if side == constant.SideProvider {
		return true
	}
	logger.Debugf("[Registry][ServiceDiscovery] the URL should not be register, url=%s", url.String())
	return false
}

func (s *serviceDiscoveryRegistry) Subscribe(url *common.URL, notify registry.NotifyListener) error {
	if !shouldSubscribe(url) {
		logger.Infof("[Registry][ServiceDiscovery] service %s is set to not subscribe to instances", url.ServiceKey())
		return nil
	}
	if id, exist := s.url.GetNonDefaultParam(constant.RegistryIdKey); exist {
		metadata.AddSubscribeURL(id, url)
	}
	mappingListener := NewMappingListener(s.url, url, parseServices(url.GetParam(constant.ProvidedBy, "")), notify)
	services := s.getServices(url, mappingListener)
	if services.Empty() {
		logger.Infof("[Registry][ServiceDiscovery] should has at least one way to know which services this interface belongs to,"+
			" either specify 'provided-by' for reference or enable metadata-report center subscription url:%s", url.String())
	} else {
		logger.Infof("[Registry][ServiceDiscovery] find initial mapping applications %q for service %s", services, url.ServiceKey())
		if _, ok := url.GetNonDefaultParam(constant.ProvidedBy); ok {
			// provided-by is an explicit, unchanging initial target set, so it is
			// subscribed directly. Routing it through the mapping change listener
			// treats it as an unchanged mapping and skips SubscribeURL entirely.
			s.SubscribeURL(url, notify, services)
		} else {
			// metadata-report mapping is dynamic: keep the initial subscription on
			// OnEvent so the listener baseline (oldServiceNames) is updated and later
			// mapping updates diff against it instead of re-subscribing.
			err := mappingListener.OnEvent(registry.NewServiceMappingChangedEvent(url.ServiceKey(), services))
			if err != nil {
				logger.Errorf("[Registry][ServiceDiscovery] ServiceInstancesChangedListenerImpl handle error, err=%v", err)
			}
		}
	}
	return nil
}

func (s *serviceDiscoveryRegistry) SubscribeURL(url *common.URL, notify registry.NotifyListener, services *gxset.HashSet) {
	serviceNamesKey := sortServices(services)
	protocolServiceKey := protocolSubscribeKey(url)

	// A destroyed registry accepts no new subscriptions.
	if s.isDestroyed() {
		return
	}

	// Fast path: reuse an already installed listener without touching external calls.
	if listener := s.getServiceListener(serviceNamesKey); listener != nil {
		s.subscribeAndNotify(url, serviceNamesKey, protocolServiceKey, listener, notify)
		return
	}

	// Build the listener and load its initial instances outside s.lock.
	// GetInstances and OnEvent may perform external RPC / metadata-report
	// calls; holding the registry write lock across them would block every
	// other subscribe/unsubscribe on this registry. The lock below only guards
	// the serviceListeners check/install, never the external work.
	listener := NewServiceInstancesChangedListenerWithContext(s.ctx, url.GetParam(constant.ApplicationKey, ""), s.url.GetParam(constant.RegistryIdKey, constant.DefaultKey), services)
	s.loadLatestInstances(listener)

	// Install under a short write lock with a double-check so a concurrent
	// subscriber for the same key does not install a duplicate listener.
	s.lock.Lock()
	if s.destroyed {
		// Destroy ran while the initial GetInstances/metadata phase above was
		// in flight: the listener was invisible to it, so it is not closed.
		// Discard it here instead of installing into a dead registry.
		s.lock.Unlock()
		if impl, ok := listener.(*ServiceInstancesChangedListenerImpl); ok {
			impl.stopMetadataRetry()
		}
		logger.Warnf("[Registry][ServiceDiscovery] discard late subscribe for applications=%s: registry already destroyed", serviceNamesKey)
		return
	}
	if existing := s.serviceListeners[serviceNamesKey]; existing != nil {
		// The loser of the install race is dropped without subscribers; close
		// it so it can never arm a metadata retry or be kept alive by one.
		if impl, ok := listener.(*ServiceInstancesChangedListenerImpl); ok {
			impl.stopMetadataRetry()
		}
		listener = existing
	} else {
		s.serviceListeners[serviceNamesKey] = listener
	}
	s.lock.Unlock()

	s.subscribeAndNotify(url, serviceNamesKey, protocolServiceKey, listener, notify)
}

// getServiceListener returns the listener installed for serviceNamesKey, or nil
// if none has been installed yet, acquired under a read lock.
func (s *serviceDiscoveryRegistry) getServiceListener(serviceNamesKey string) registry.ServiceInstancesChangedListener {
	s.lock.RLock()
	defer s.lock.RUnlock()
	return s.serviceListeners[serviceNamesKey]
}

// loadLatestInstances pushes the current registry snapshot for every subscribed
// application into the listener. It runs outside s.lock: GetInstances and
// OnEvent may perform external RPC / metadata-report calls.
func (s *serviceDiscoveryRegistry) loadLatestInstances(listener registry.ServiceInstancesChangedListener) {
	for _, serviceNameTmp := range listener.GetServiceNames().Values() {
		serviceName := serviceNameTmp.(string)
		instances := s.serviceDiscovery.GetInstances(serviceName)
		logger.Infof("[Registry][ServiceDiscovery] synchronized instance notification on application %s subscription, instance list size %d", serviceName, len(instances))
		if err := listener.OnEvent(&registry.ServiceInstancesChangedEvent{
			ServiceName: serviceName,
			Instances:   instances,
		}); err != nil {
			logger.Warnf("[Registry][ServiceDiscovery] ServiceInstancesChangedListenerImpl handle error, err=%v", err)
		}
	}
}

// subscribeAndNotify registers the notify callback and asynchronously wires the
// listener into the service discovery so the caller does not block on it. A
// failed AddListener is retried in the background with backoff; without the
// retry a transient registry error would leave the consumer permanently stale
// (issue #3624).
func (s *serviceDiscoveryRegistry) subscribeAndNotify(url *common.URL, serviceNamesKey, protocolServiceKey string,
	listener registry.ServiceInstancesChangedListener, notify registry.NotifyListener,
) {
	listener.AddListenerAndNotify(protocolServiceKey, notify)

	logger.Infof("[Registry][ServiceDiscovery] start subscribing to registry for applications=%s with a new go routine", serviceNamesKey)
	go func() {
		if err := s.addInstanceListener(serviceNamesKey, url, listener); err != nil {
			logger.Errorf("[Registry][ServiceDiscovery] add instance listener catch error, url=%s err=%s", url.String(), err.Error())
			s.scheduleSubscribeRetry(serviceNamesKey, &subscribeRetry{listener: listener, url: url})
		}
	}()
}

// addInstanceListener installs the listener into the service discovery and
// publishes the subscribe metrics. A success resolves any pending retry for
// the key so its timer cannot fire an extra (possibly non-idempotent)
// AddListener.
func (s *serviceDiscoveryRegistry) addInstanceListener(serviceNamesKey string, url *common.URL, listener registry.ServiceInstancesChangedListener) error {
	event := metricsMetadata.NewMetadataMetricTimeEvent(metricsMetadata.SubscribeServiceRt)
	err := s.serviceDiscovery.AddListener(listener)
	event.Succ = err == nil
	event.End = time.Now()
	event.Attachment[constant.InterfaceKey] = url.Interface()
	metrics.Publish(event)
	metrics.Publish(metricsRegistry.NewServerSubscribeEvent(err == nil))
	if err == nil {
		s.resolveSubscribeRetry(serviceNamesKey, listener)
	}
	return err
}

var (
	// subscribeRetryInitialDelay is the first backoff delay before retrying a
	// failed AddListener call. Package-level so tests can shrink it.
	subscribeRetryInitialDelay = time.Second
	// subscribeRetryMaxDelay caps the backoff. The retry count itself is
	// intentionally unlimited: a capped count would re-introduce the
	// permanently stale consumer this mechanism fixes.
	subscribeRetryMaxDelay = 30 * time.Second
)

// subscribeRetry is a pending or in-flight AddListener retry for one
// serviceNamesKey. The entry stays in the map while the attempt is in flight
// (inFlight) so a concurrent failure cannot seed a fresh attempts=0 state and
// reset the backoff. canceled is set when the state is resolved by a
// successful subscribe or canceled by unsubscribe/Destroy; an in-flight
// attempt checks it before rescheduling.
type subscribeRetry struct {
	listener registry.ServiceInstancesChangedListener
	url      *common.URL
	timer    *time.Timer
	attempts int
	inFlight bool
	canceled bool
}

// scheduleSubscribeRetry arms the retry timer for serviceNamesKey after a
// failed AddListener call. One pending retry per key: repeated failures share
// the same state instead of stacking new ones. Retries continue with capped
// exponential backoff and jitter until the subscription is established, the
// last subscriber unsubscribes, or the registry is destroyed.
func (s *serviceDiscoveryRegistry) scheduleSubscribeRetry(serviceNamesKey string, state *subscribeRetry) {
	s.lock.Lock()
	defer s.lock.Unlock()
	if s.destroyed || state.canceled {
		return
	}
	if !listenerHasSubscribers(state.listener) {
		// No subscriber left (e.g. unsubscribe raced with a failing retry):
		// do not arm a timer nobody waits for.
		return
	}
	if _, ok := s.subscribeRetries[serviceNamesKey]; ok {
		return
	}
	s.subscribeRetries[serviceNamesKey] = state
	s.armSubscribeRetryLocked(serviceNamesKey, state)
}

// armSubscribeRetryLocked computes the next backoff delay and arms the retry
// timer; caller must hold s.lock and the state must be in the map.
func (s *serviceDiscoveryRegistry) armSubscribeRetryLocked(serviceNamesKey string, state *subscribeRetry) {
	delay := subscribeRetryDelay(state.attempts)
	state.attempts++
	state.timer = time.AfterFunc(delay, func() {
		s.retryAddListener(serviceNamesKey)
	})
	logger.Debugf("[Registry][ServiceDiscovery] instance listener for applications=%s not established, retry in %s", serviceNamesKey, delay)
}

// cancelSubscribeRetry stops a pending AddListener retry, if any.
func (s *serviceDiscoveryRegistry) cancelSubscribeRetry(serviceNamesKey string) {
	s.lock.Lock()
	defer s.lock.Unlock()
	s.cancelSubscribeRetryLocked(serviceNamesKey)
}

// cancelSubscribeRetryLocked stops a pending AddListener retry; caller must
// hold s.lock. An in-flight attempt is not interrupted but will not
// reschedule.
func (s *serviceDiscoveryRegistry) cancelSubscribeRetryLocked(serviceNamesKey string) {
	if state, ok := s.subscribeRetries[serviceNamesKey]; ok {
		state.canceled = true
		if state.timer != nil {
			state.timer.Stop()
		}
		delete(s.subscribeRetries, serviceNamesKey)
	}
}

// resolveSubscribeRetry drops a pending retry after a successful AddListener
// for the same listener: the subscription is established, so the timer must
// not fire an extra (possibly non-idempotent) AddListener. An in-flight
// attempt that later fails sees canceled and does not reschedule.
func (s *serviceDiscoveryRegistry) resolveSubscribeRetry(serviceNamesKey string, listener registry.ServiceInstancesChangedListener) {
	s.lock.Lock()
	defer s.lock.Unlock()
	state, ok := s.subscribeRetries[serviceNamesKey]
	if !ok || state.listener != listener {
		return
	}
	state.canceled = true
	if state.timer != nil {
		state.timer.Stop()
	}
	delete(s.subscribeRetries, serviceNamesKey)
}

// retryAddListener re-runs AddListener after the backoff delay. The state
// stays in the map while the attempt is in flight so concurrent failures
// dedup against it and the backoff attempts keep growing. On success it
// re-syncs the latest instance snapshot so instance changes missed while the
// subscription was down are picked up instead of leaving the consumer on a
// stale view.
func (s *serviceDiscoveryRegistry) retryAddListener(serviceNamesKey string) {
	s.lock.Lock()
	state, ok := s.subscribeRetries[serviceNamesKey]
	if !ok {
		s.lock.Unlock()
		return
	}
	state.timer = nil
	state.inFlight = true
	active := !s.destroyed && !state.canceled &&
		s.serviceListeners[serviceNamesKey] == state.listener &&
		listenerHasSubscribers(state.listener)
	s.lock.Unlock()

	if !active {
		// Registry destroyed, listener replaced, or no subscriber left:
		// drop the state so the loop cannot leak.
		s.lock.Lock()
		if s.subscribeRetries[serviceNamesKey] == state {
			delete(s.subscribeRetries, serviceNamesKey)
		}
		s.lock.Unlock()
		return
	}
	if err := s.addInstanceListener(serviceNamesKey, state.url, state.listener); err != nil {
		s.lock.Lock()
		defer s.lock.Unlock()
		if state.canceled || s.destroyed || !listenerHasSubscribers(state.listener) {
			// A concurrent successful subscribe resolved this state, or the
			// listener is gone: drop it instead of rescheduling.
			if s.subscribeRetries[serviceNamesKey] == state {
				delete(s.subscribeRetries, serviceNamesKey)
			}
			return
		}
		// Re-arm in place: the entry never left the map, so the backoff
		// attempts keep growing and no duplicate state can appear.
		state.inFlight = false
		s.armSubscribeRetryLocked(serviceNamesKey, state)
		logger.Warnf("[Registry][ServiceDiscovery] retry add instance listener failed, applications=%s attempt=%d err=%s",
			serviceNamesKey, state.attempts, err.Error())
		return
	}
	// Success: addInstanceListener already resolved the state.
	if s.isDestroyed() {
		return
	}
	logger.Infof("[Registry][ServiceDiscovery] instance listener for applications=%s established after %d retries, re-syncing latest instances",
		serviceNamesKey, state.attempts)
	s.loadLatestInstances(state.listener)
}

// subscribeRetryDelay returns exponential backoff (initial << attempt) capped
// at subscribeRetryMaxDelay, plus up to 25% jitter to desynchronize retries
// across consumers after a correlated registry failure.
func subscribeRetryDelay(attempt int) time.Duration {
	delay := subscribeRetryMaxDelay
	if attempt >= 0 && attempt < 30 { // guard against shift overflow
		delay = subscribeRetryInitialDelay << attempt
		if delay <= 0 || delay > subscribeRetryMaxDelay {
			delay = subscribeRetryMaxDelay
		}
	}
	return delay + time.Duration(rand.Int64N(int64(delay/4)+1))
}

// isDestroyed reports whether Destroy has run.
func (s *serviceDiscoveryRegistry) isDestroyed() bool {
	s.lock.RLock()
	defer s.lock.RUnlock()
	return s.destroyed
}

// listenerHasSubscribers reports whether the listener still has subscribers.
// Unknown implementations are assumed active so retries are never dropped
// silently.
func listenerHasSubscribers(listener registry.ServiceInstancesChangedListener) bool {
	if impl, ok := listener.(*ServiceInstancesChangedListenerImpl); ok {
		return impl.hasSubscribers()
	}
	return true
}

// protocolSubscribeKey builds the key under which a subscription's notify
// listener is registered on the ServiceInstancesChangedListener. SubscribeURL
// and UnSubscribe must derive it the same way, or the last subscriber is never
// removed and the metadata retry keeps probing after UnSubscribe.
func protocolSubscribeKey(url *common.URL) string {
	protocol := constant.TriProtocol // consume "tri" protocol by default, other protocols need to be specified on reference/consumer explicitly
	if url.Protocol != "" {
		protocol = url.Protocol
	}
	return url.ServiceKey() + ":" + protocol
}

func sortServices(services *gxset.HashSet) string {
	list := make([]string, 0, services.Size())
	for _, v := range services.Values() {
		if s, ok := v.(string); ok && s != "" {
			list = append(list, s)
		}
	}
	sort.Strings(list)
	return strings.Join(list, ",")
}

// LoadSubscribeInstances load subscribe instance
func (s *serviceDiscoveryRegistry) LoadSubscribeInstances(url *common.URL, notify registry.NotifyListener) error {
	return nil
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
		logger.Errorf("[Registry][ServiceDiscovery] get service names catch error, url=%s err=%s", url.String(), err.Error())
		return gxset.NewSet()
	}
	if listener != nil {
		protocolServiceKey := url.ServiceKey() + ":" + url.Protocol
		s.lock.Lock()
		s.serviceMappingListeners[protocolServiceKey] = listener
		s.lock.Unlock()
	}
	return serviceNames
}

func (s *serviceDiscoveryRegistry) stopListen(url *common.URL) {
	protocolServiceKey := url.ServiceKey() + ":" + url.Protocol
	s.lock.Lock()
	listener := s.serviceMappingListeners[protocolServiceKey]
	if listener != nil {
		delete(s.serviceMappingListeners, protocolServiceKey)
		listener.Stop()
	}
	s.lock.Unlock()
}
