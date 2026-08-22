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
	"encoding/gob"
	"maps"
	"math/rand/v2"
	"reflect"
	"sync"
	"time"
)

import (
	gxset "github.com/dubbogo/gost/container/set"
	"github.com/dubbogo/gost/gof/observer"
	"github.com/dubbogo/gost/log/logger"

	perrors "github.com/pkg/errors"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/metadata"
	"dubbo.apache.org/dubbo-go/v3/metadata/info"
	"dubbo.apache.org/dubbo-go/v3/registry"
	"dubbo.apache.org/dubbo-go/v3/registry/servicediscovery/store"
	"dubbo.apache.org/dubbo-go/v3/remoting"
)

var (
	metaCache *store.CacheManager
	cacheOnce sync.Once
)

func initCache(app string) {
	gob.Register(&info.MetadataInfo{})
	fileName := constant.DefaultMetaFileName + app
	cache, err := store.NewCacheManager(constant.DefaultMetaCacheName, fileName, time.Minute*10, constant.DefaultEntrySize, true)
	if err != nil {
		logger.Fatalf("[Registry][ServiceDiscovery] failed to create cache [%s],the err is %v", constant.DefaultMetaCacheName, err)
	}
	metaCache = cache
}

// ServiceInstancesChangedListenerImpl The Service Discovery Changed  Event Listener
type ServiceInstancesChangedListenerImpl struct {
	ctx                context.Context
	app                string
	registryId         string
	serviceNames       *gxset.HashSet
	listeners          map[string]registry.NotifyListener
	serviceUrls        map[string][]*common.URL
	revisionToMetadata map[string]*info.MetadataInfo
	allInstances       map[string][]registry.ServiceInstance
	mutex              sync.Mutex

	// buildMu serializes service URL rebuilds so registry events and metadata
	// retries cannot interleave. Unlike mutex it may be held across metadata RPCs.
	buildMu sync.Mutex
	// unresolvedRevisions tracks revision keys whose metadata fetch failed and
	// must be retried. Guarded by mutex.
	unresolvedRevisions map[string]struct{}
	retryTimer          *time.Timer
	retryAttempts       int
	lastFailureLog      map[string]time.Time
	// closed is set when the owning registry drops this listener for good
	// (Destroy, or a duplicate listener discarded during subscribe). A closed
	// listener never arms a new retry timer. Guarded by mutex.
	closed bool
}

func NewServiceInstancesChangedListener(app string, registryId string, services *gxset.HashSet) registry.ServiceInstancesChangedListener {
	return NewServiceInstancesChangedListenerWithContext(context.Background(), app, registryId, services)
}

// NewServiceInstancesChangedListenerWithContext creates a listener whose
// metadata refreshes are canceled with ctx, such as when its registry closes.
func NewServiceInstancesChangedListenerWithContext(ctx context.Context, app string, registryId string, services *gxset.HashSet) registry.ServiceInstancesChangedListener {
	if ctx == nil {
		ctx = context.Background()
	}
	cacheOnce.Do(func() {
		initCache(app)
	})
	return &ServiceInstancesChangedListenerImpl{
		ctx:                 ctx,
		app:                 app,
		registryId:          registryId,
		serviceNames:        services,
		listeners:           make(map[string]registry.NotifyListener),
		serviceUrls:         make(map[string][]*common.URL),
		revisionToMetadata:  make(map[string]*info.MetadataInfo),
		allInstances:        make(map[string][]registry.ServiceInstance),
		unresolvedRevisions: make(map[string]struct{}),
		lastFailureLog:      make(map[string]time.Time),
	}
}

// OnEvent handles service instance change events by refreshing metadata, rebuilding service URLs, and notifying listeners.
func (lstn *ServiceInstancesChangedListenerImpl) OnEvent(e observer.Event) error {
	ce, ok := e.(*registry.ServiceInstancesChangedEvent)
	if !ok {
		return nil
	}

	logger.Debugf("[Registry][ServiceDiscovery] received instance notification event, service=%s size=%d", ce.ServiceName, len(ce.Instances))

	lstn.mutex.Lock()
	lstn.allInstances[ce.ServiceName] = ce.Instances
	lstn.mutex.Unlock()

	if !lstn.refreshServiceURLs() {
		return perrors.Errorf("metadata unresolved for some revisions of service=%s, retry is scheduled", ce.ServiceName)
	}
	return nil
}

// refreshServiceURLs rebuilds service URLs from the latest instance snapshot and
// notifies subscribers. The build is serialized by buildMu, but lstn.mutex is
// only held while reading or committing in-memory state: metadata RPCs run in
// between without it, so a slow or unreachable provider cannot block event
// processing or retry scheduling. It reports whether every revision resolved;
// unresolved revisions are retried by the shared retry timer.
func (lstn *ServiceInstancesChangedListenerImpl) refreshServiceURLs() bool {
	lstn.buildMu.Lock()
	defer lstn.buildMu.Unlock()

	lstn.mutex.Lock()
	allInstances := make(map[string][]registry.ServiceInstance, len(lstn.allInstances))
	maps.Copy(allInstances, lstn.allInstances)
	cachedMetadata := make(map[string]*info.MetadataInfo, len(lstn.revisionToMetadata))
	maps.Copy(cachedMetadata, lstn.revisionToMetadata)
	lstn.mutex.Unlock()

	revisionToInstances := make(map[string][]registry.ServiceInstance, len(cachedMetadata))
	newRevisionToMetadata := make(map[string]*info.MetadataInfo, len(cachedMetadata))
	// The same service match key can be exported by several revisions.
	// Keep each revision's ServiceInfo so provider-specific params are not collapsed.
	serviceToRevisionServices := make(map[string]map[string]*info.ServiceInfo, len(cachedMetadata))
	unresolved := make(map[string]struct{})

	for _, instances := range allInstances {
		for _, instance := range instances {
			if instance.GetMetadata() == nil {
				logger.Warnf("[Registry][ServiceDiscovery] instance metadata is nil, host=%s", instance.GetHost())
				continue
			}
			revision := instance.GetMetadata()[constant.ExportedServicesRevisionPropertyName]
			if revision == "0" {
				logger.Infof("[Registry][ServiceDiscovery] find instance without valid service metadata, host=%s", instance.GetHost())
				continue
			}
			// MetadataInfo belongs to the provider application, so isolate every cache
			// dimension by provider app. Two provider apps that happen to share a revision
			// (e.g. same interface set exported under different application names) must not
			// collide on a revision-only key. instance.GetServiceName() is the provider app.
			providerApp := instance.GetServiceName()
			key := metadataCacheKey(providerApp, lstn.registryId, revision)

			revisionToInstances[key] = append(revisionToInstances[key], instance)
			metadataInfo := newRevisionToMetadata[key]
			if metadataInfo == nil {
				metadataInfo = cachedMetadata[key]
			}
			if metadataInfo == nil {
				meta, err := metadataInfoFetcher(lstn.ctx, providerApp, instance, revision, lstn.registryId)
				if err != nil {
					// Skip this instance if metadata fetch fails (e.g., old Java Dubbo version)
					// Try next instance with same revision. The revision is recorded as
					// unresolved so it is retried later instead of being dropped silently.
					lstn.logMetadataFetchFailure(key, instance.GetHost(), revision, err)
					unresolved[key] = struct{}{}
					continue
				}
				metadataInfo = meta
			}
			if metadataInfo == nil {
				logger.Warnf("[Registry][ServiceDiscovery] metadata info is nil for instance %s (revision %s), skipping this instance",
					instance.GetHost(), revision)
				unresolved[key] = struct{}{}
				continue
			}
			instance.SetServiceMetadata(metadataInfo)
			for _, service := range metadataInfo.GetServices() {
				matchKey := service.GetMatchKey()
				if serviceToRevisionServices[matchKey] == nil {
					serviceToRevisionServices[matchKey] = make(map[string]*info.ServiceInfo)
				}
				serviceToRevisionServices[matchKey][key] = service
			}

			newRevisionToMetadata[key] = metadataInfo
		}
	}

	newServiceURLs := make(map[string][]*common.URL, len(serviceToRevisionServices))
	for serviceKey, revisionServices := range serviceToRevisionServices {
		urls := make([]*common.URL, 0, 8)
		for key, serviceInfo := range revisionServices {
			for _, i := range revisionToInstances[key] {
				if i != nil {
					urls = append(urls, toInstanceServiceURLs(i, serviceInfo)...)
				}
			}
		}
		newServiceURLs[serviceKey] = urls
	}

	lstn.mutex.Lock()
	lstn.revisionToMetadata = newRevisionToMetadata
	lstn.serviceUrls = newServiceURLs
	lstn.unresolvedRevisions = unresolved
	// Drop throttling state for revisions that resolved or disappeared.
	for key := range lstn.lastFailureLog {
		if _, ok := unresolved[key]; !ok {
			delete(lstn.lastFailureLog, key)
		}
	}
	listeners := make(map[string]registry.NotifyListener, len(lstn.listeners))
	maps.Copy(listeners, lstn.listeners)
	lstn.mutex.Unlock()

	for key, metadataInfo := range newRevisionToMetadata {
		// key is already provider-app scoped and matches the disk cache key format.
		metaCache.Set(key, metadataInfo)
	}

	for key, notifyListener := range listeners {
		urls := newServiceURLs[key]
		events := make([]*registry.ServiceEvent, 0, len(urls))
		for _, url := range urls {
			events = append(events, &registry.ServiceEvent{
				Action:  remoting.EventTypeAdd,
				Service: url,
			})
		}
		notifyListener.NotifyAll(events, func() {})
	}

	lstn.scheduleMetadataRetry()
	return len(unresolved) == 0
}

func toInstanceServiceURLs(instance registry.ServiceInstance, serviceInfo *info.ServiceInfo) []*common.URL {
	urls := instance.ToURLs(serviceInfo)
	// Environment is instance-level routing metadata and is not part of the revision hash.
	// Treat the fresh instance value as authoritative so same-revision restarts
	// can update or clear stale metadata cached by revision.
	metadata := instance.GetMetadata()
	if metadata == nil {
		for _, url := range urls {
			url.DelParam(constant.EnvironmentKey)
		}
		return urls
	}
	environment, ok := metadata[constant.EnvironmentKey]
	for _, url := range urls {
		if ok && len(environment) > 0 {
			url.SetParam(constant.EnvironmentKey, environment)
		} else {
			url.DelParam(constant.EnvironmentKey)
		}
	}
	return urls
}

// AddListenerAndNotify add notify listener and notify to listen service event
func (lstn *ServiceInstancesChangedListenerImpl) AddListenerAndNotify(serviceKey string, notify registry.NotifyListener) {
	lstn.mutex.Lock()
	lstn.listeners[serviceKey] = notify
	urls := lstn.serviceUrls[serviceKey]
	hasUnresolved := len(lstn.unresolvedRevisions) > 0
	lstn.mutex.Unlock()

	if hasUnresolved {
		// A subscriber (re-)attached while metadata is still unresolved; make
		// sure the retry loop is running for it.
		lstn.scheduleMetadataRetry()
	}

	for _, url := range urls {
		notify.Notify(&registry.ServiceEvent{
			Action:  remoting.EventTypeAdd,
			Service: url,
		})
	}
}

// RemoveListener remove notify listener
func (lstn *ServiceInstancesChangedListenerImpl) RemoveListener(serviceKey string) {
	lstn.mutex.Lock()
	defer lstn.mutex.Unlock()
	delete(lstn.listeners, serviceKey)
	if len(lstn.listeners) == 0 && lstn.retryTimer != nil {
		// No subscriber left: stop retrying so the timer does not keep the
		// listener alive after it is dropped.
		lstn.retryTimer.Stop()
		lstn.retryTimer = nil
	}
}

// hasSubscribers reports whether any notify listener is still attached. The
// owning registry uses it to stop pending subscribe retries once the last
// subscriber unsubscribes.
func (lstn *ServiceInstancesChangedListenerImpl) hasSubscribers() bool {
	lstn.mutex.Lock()
	defer lstn.mutex.Unlock()
	return len(lstn.listeners) > 0
}

// GetServiceNames return all listener service names
func (lstn *ServiceInstancesChangedListenerImpl) GetServiceNames() *gxset.HashSet {
	return lstn.serviceNames
}

// Accept return true if the name is the same
func (lstn *ServiceInstancesChangedListenerImpl) Accept(e observer.Event) bool {
	if ce, ok := e.(*registry.ServiceInstancesChangedEvent); ok {
		return lstn.serviceNames.Contains(ce.ServiceName)
	}
	return false
}

// GetPriority returns -1, it will be the first invoked listener
func (lstn *ServiceInstancesChangedListenerImpl) GetPriority() int {
	return -1
}

// GetEventType returns ServiceInstancesChangedEvent
func (lstn *ServiceInstancesChangedListenerImpl) GetEventType() reflect.Type {
	return reflect.TypeFor[*registry.ServiceInstancesChangedEvent]()
}

// metadataCacheKey builds the cache key that isolates MetadataInfo by provider
// application, registry, and revision. MetadataInfo is owned by the provider app,
// so app must be the provider application name (instance.GetServiceName()), never
// the subscribing consumer app. Keying on revision alone would let two provider
// apps that share a revision overwrite each other's metadata.
func metadataCacheKey(app, registryId, revision string) string {
	return app + ":" + registryId + ":" + revision
}

// metadataInfoFetcher resolves MetadataInfo for a revision; a package-level
// indirection so tests can inject transient failures. It follows
// GetMetadataInfoWithContext so listener refreshes are canceled with the
// listener's lifecycle context.
var metadataInfoFetcher = GetMetadataInfoWithContext

// GetMetadataInfo retrieves the MetadataInfo for a service instance by revision.
// Results are cached by app+registryId+revision, where app must be the provider
// application name. For "remote" storage type, it fetches from the metadata report
// and falls back to RPC if the report fails or returns nil. For all other storage
// types (including absent), it uses RPC directly.
func GetMetadataInfo(app string, instance registry.ServiceInstance, revision string, registryId string) (*info.MetadataInfo, error) {
	return GetMetadataInfoWithContext(context.Background(), app, instance, revision, registryId)
}

// GetMetadataInfoWithContext retrieves metadata using the supplied lifecycle
// context for metadata RPC fallbacks.
func GetMetadataInfoWithContext(ctx context.Context, app string, instance registry.ServiceInstance, revision string, registryId string) (*info.MetadataInfo, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	cacheOnce.Do(func() {
		initCache(app)
	})
	cacheKey := metadataCacheKey(app, registryId, revision)
	if metadataInfo, ok := metaCache.Get(cacheKey); ok {
		return metadataInfo.(*info.MetadataInfo), nil
	}

	var metadataInfo *info.MetadataInfo
	var err error
	if getMetadataStorageType(instance) == constant.RemoteMetadataStorageType {
		metadataInfo, err = getRemoteMetadataInfo(ctx, app, instance, revision, registryId)
	} else {
		metadataInfo, err = getMetadataInfoFromRPC(ctx, app, instance, revision, registryId)
	}
	if err != nil {
		return nil, err
	}
	metaCache.Set(cacheKey, metadataInfo)
	return metadataInfo, nil
}

var (
	// metadataRetryInitialDelay is the first backoff delay before retrying a failed
	// metadata fetch. Package-level so tests can shrink it.
	metadataRetryInitialDelay = time.Second
	// metadataRetryMaxDelay caps the backoff. The retry count itself is
	// intentionally unlimited: retries only target instances the registry still
	// reports as alive, and a capped count would re-introduce the permanent
	// empty-directory failure this mechanism fixes.
	metadataRetryMaxDelay = 30 * time.Second
	// metadataFetchFailureLogInterval throttles repeated fetch-failure warnings
	// for the same revision key.
	metadataFetchFailureLogInterval = 5 * time.Minute
)

// stopMetadataRetry marks the listener closed and cancels any pending metadata
// retry. It is called when the owning registry is destroyed and drops this
// listener without RemoveListener, so neither the pending timer nor an
// in-flight refresh can arm a new one afterwards.
func (lstn *ServiceInstancesChangedListenerImpl) stopMetadataRetry() {
	lstn.mutex.Lock()
	defer lstn.mutex.Unlock()
	lstn.closed = true
	if lstn.retryTimer != nil {
		lstn.retryTimer.Stop()
		lstn.retryTimer = nil
	}
}

// scheduleMetadataRetry arms the shared retry timer while unresolved revisions
// remain. Retries replay the latest instance snapshot, so revisions whose
// instances disappeared from the registry are dropped naturally on the next
// run. No timer is armed once the listener is closed or while it has no
// subscribers: AddListenerAndNotify re-arms the retry when a subscriber
// attaches, and a subscriber-less listener must not keep probing metadata.
func (lstn *ServiceInstancesChangedListenerImpl) scheduleMetadataRetry() {
	lstn.mutex.Lock()
	defer lstn.mutex.Unlock()
	if len(lstn.unresolvedRevisions) == 0 || lstn.closed || len(lstn.listeners) == 0 {
		if lstn.retryTimer != nil {
			lstn.retryTimer.Stop()
			lstn.retryTimer = nil
		}
		lstn.retryAttempts = 0
		return
	}
	if lstn.retryTimer != nil {
		// One shared timer per listener: repeated events must not multiply retries.
		return
	}
	delay := metadataRetryDelay(lstn.retryAttempts)
	lstn.retryAttempts++
	lstn.retryTimer = time.AfterFunc(delay, func() {
		lstn.mutex.Lock()
		lstn.retryTimer = nil
		// Re-check under the lock: the last subscriber may have been removed
		// or the listener closed while the timer was pending.
		run := !lstn.closed && len(lstn.listeners) > 0 && len(lstn.unresolvedRevisions) > 0
		lstn.mutex.Unlock()
		if run {
			lstn.refreshServiceURLs()
		}
	})
}

// metadataRetryDelay returns exponential backoff (initial << attempt) capped at
// metadataRetryMaxDelay, plus up to 25% jitter to desynchronize retries across
// consumers after a correlated provider restart.
func metadataRetryDelay(attempt int) time.Duration {
	delay := metadataRetryMaxDelay
	if attempt >= 0 && attempt < 30 { // guard against shift overflow
		delay = metadataRetryInitialDelay << attempt
		if delay <= 0 || delay > metadataRetryMaxDelay {
			delay = metadataRetryMaxDelay
		}
	}
	return delay + time.Duration(rand.Int64N(int64(delay/4)+1))
}

// logMetadataFetchFailure logs a throttled warning for a failed metadata fetch.
func (lstn *ServiceInstancesChangedListenerImpl) logMetadataFetchFailure(key, host, revision string, err error) {
	lstn.mutex.Lock()
	last, logged := lstn.lastFailureLog[key]
	now := time.Now()
	shouldLog := !logged || now.Sub(last) >= metadataFetchFailureLogInterval
	if shouldLog {
		lstn.lastFailureLog[key] = now
	}
	lstn.mutex.Unlock()
	if shouldLog {
		logger.Warnf("[Registry][ServiceDiscovery] failed to get metadata from instance %s (revision %s), err=%v, skipping this instance",
			host, revision, err)
	}
}

func getMetadataStorageType(instance registry.ServiceInstance) string {
	instanceMetadata := instance.GetMetadata()
	if instanceMetadata == nil {
		return constant.DefaultMetadataStorageType
	}

	storageType := instanceMetadata[constant.MetadataStorageTypePropertyName]
	if storageType == "" {
		logger.Warnf("[Metadata] MetadataStorageType not set for instance %s, defaulting to local", instance.GetID())
		return constant.DefaultMetadataStorageType
	}
	return storageType
}

func getMetadataInfoFromRPC(ctx context.Context, app string, instance registry.ServiceInstance, revision string, registryId string) (*info.MetadataInfo, error) {
	metadataInfo, err := metadata.GetMetadataFromRpcWithContext(ctx, revision, instance)
	if err != nil {
		return nil, perrors.Wrapf(err,
			"failed app=%s registry=%s revision=%s", app, registryId, revision)
	}
	return requireMetadataInfo(metadataInfo, app, registryId, revision)
}

func getRemoteMetadataInfo(ctx context.Context, app string, instance registry.ServiceInstance, revision string, registryId string) (*info.MetadataInfo, error) {
	metadataInfo, reportErr := metadata.GetMetadataFromMetadataReport(revision, instance, registryId)
	if reportErr == nil && metadataInfo != nil {
		return metadataInfo, nil
	}
	logMetadataReportFallback(app, registryId, revision, reportErr)

	metadataInfo, rpcErr := metadata.GetMetadataFromRpcWithContext(ctx, revision, instance)
	if rpcErr != nil {
		return nil, wrapMetadataRPCFallbackError(rpcErr, reportErr)
	}
	return requireMetadataInfo(metadataInfo, app, registryId, revision)
}

func logMetadataReportFallback(app, registryId, revision string, reportErr error) {
	if reportErr != nil {
		logger.Errorf("[Metadata][Fallback] report failed, fallback to RPC app=%s registry=%s revision=%s err=%v",
			app, registryId, revision, reportErr)
		return
	}
	logger.Warnf("[Metadata][Fallback] report returned nil metadata, fallback to RPC app=%s registry=%s revision=%s",
		app, registryId, revision)
}

func wrapMetadataRPCFallbackError(rpcErr, reportErr error) error {
	if reportErr != nil {
		// Wrap rpcErr so callers can use errors.Is/As on the primary failure;
		// reportErr is annotated as context since it triggered the fallback.
		return perrors.Wrapf(rpcErr, "both paths failed, reportErr: %v", reportErr)
	}
	return perrors.Wrapf(rpcErr, "RPC fallback failed after report returned nil metadata")
}

func requireMetadataInfo(metadataInfo *info.MetadataInfo, app, registryId, revision string) (*info.MetadataInfo, error) {
	if metadataInfo == nil {
		return nil, perrors.Errorf("got nil metadata from RPC app=%s registry=%s revision=%s",
			app, registryId, revision)
	}
	return metadataInfo, nil
}
