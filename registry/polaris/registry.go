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

package polaris

import (
	"fmt"
	"reflect"
	"strconv"
	"sync"
	"time"
)

import (
	"github.com/dubbogo/gost/log/logger"

	perrors "github.com/pkg/errors"

	api "github.com/polarismesh/polaris-go"
	"github.com/polarismesh/polaris-go/pkg/model"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/common/extension"
	"dubbo.apache.org/dubbo-go/v3/registry"
	"dubbo.apache.org/dubbo-go/v3/remoting"
	"dubbo.apache.org/dubbo-go/v3/remoting/polaris"
)

const (
	RegistryConnDelay           = 3
	defaultHeartbeatIntervalSec = 5
)

func init() {
	extension.SetRegistry(constant.PolarisKey, newPolarisRegistry)
}

// newPolarisRegistry will create new instance
func newPolarisRegistry(url *common.URL) (registry.Registry, error) {
	if err := polaris.InitSDKContext(url); err != nil {
		return &polarisRegistry{}, err
	}

	providerApi, err := polaris.GetProviderAPI()
	if err != nil {
		return nil, err
	}

	consumerApi, err := polaris.GetConsumerAPI()
	if err != nil {
		return nil, err
	}

	pRegistry := &polarisRegistry{
		url:                       url,
		namespace:                 url.GetParam(constant.RegistryNamespaceKey, constant.PolarisDefaultNamespace),
		provider:                  providerApi,
		consumer:                  consumerApi,
		registryUrls:              make([]*common.URL, 0, 4),
		watchers:                  map[string]*PolarisServiceWatcher{},
		initialSubscribeInstances: make(map[initialSubscribeInstancesKey]*initialSubscribeInstancesEntry),
	}

	return pRegistry, nil
}

type initialSubscribeInstancesKey struct {
	serviceName string
	notify      registry.NotifyListener
}

type initialSubscribeInstancesEntry struct {
	instances []model.Instance
}

type polarisRegistry struct {
	namespace    string
	url          *common.URL
	consumer     api.ConsumerAPI
	provider     api.ProviderAPI
	lock         sync.RWMutex
	registryUrls []*common.URL
	listenerLock sync.RWMutex
	watchers     map[string]*PolarisServiceWatcher
	// initialSubscribeInstances temporarily stores valid LoadSubscribeInstances
	// results, keyed by service and supported NotifyListener identity. Subscribe
	// binds them to the corresponding listener/subscriber and clears the entry
	// after listener creation succeeds. Watchers do not hold a global initial
	// baseline.
	initialSubscribeInstancesLock sync.Mutex
	initialSubscribeInstances     map[initialSubscribeInstancesKey]*initialSubscribeInstancesEntry
}

// Register will register the service @url to its polaris registry center.
func (pr *polarisRegistry) Register(url *common.URL) error {
	if getCategory(url) != "providers" {
		return nil
	}

	serviceName := url.Interface()
	request := createRegisterParam(url, serviceName)
	request.Namespace = pr.namespace
	resp, err := pr.provider.RegisterInstance(request)
	if err != nil {
		return err
	}

	if resp.Existed {
		logger.Warnf("[Registry][Polaris] instance already regist, namespace=%+v service=%+v host=%+v port=%+v",
			request.Namespace, request.Service, request.Host, request.Port)
	}
	url.SetParam(constant.PolarisInstanceID, resp.InstanceID)

	pr.lock.Lock()
	pr.registryUrls = append(pr.registryUrls, url)
	pr.lock.Unlock()

	return nil
}

// UnRegister returns nil if unregister successfully. If not, returns an error.
func (pr *polarisRegistry) UnRegister(url *common.URL) error {
	request := createDeregisterParam(url, url.Interface())
	request.Namespace = pr.namespace
	if err := pr.provider.Deregister(request); err != nil {
		return perrors.WithMessagef(err, "fail to deregister(conf:%+v)", url)
	}
	return nil
}

// Subscribe returns nil if subscribing registry successfully. If not returns an error.
func (pr *polarisRegistry) Subscribe(url *common.URL, notifyListener registry.NotifyListener) error {

	role, _ := strconv.Atoi(url.GetParam(constant.RegistryRoleKey, ""))
	if role != common.CONSUMER {
		return nil
	}
	serviceName := url.Interface()
	key, cacheable, err := initialSubscribeInstancesKeyFor(serviceName, notifyListener)
	if err != nil {
		return err
	}
	timer := time.NewTimer(time.Duration(RegistryConnDelay) * time.Second)
	defer timer.Stop()

	for {
		listener, err := pr.createPolarisListenerWithIdentity(serviceName, key, cacheable)
		if err != nil {
			logger.Warnf("[Registry][Polaris] create listener failed, service=%s, err=%v", serviceName, perrors.WithStack(err))
			<-timer.C
			timer.Reset(time.Duration(RegistryConnDelay) * time.Second)
			continue
		}

		for {
			notification, err := listener.nextNotification()

			if err != nil {
				logger.Warnf("[Registry][Polaris] Selector.watch() = err=%v", perrors.WithStack(err))
				listener.Close()
				return err
			}
			notifyPolarisNotification(notification, notifyListener)
		}
	}
}

func notifyPolarisNotification(
	notification *polarisNotification,
	notify registry.NotifyListener,
) {
	switch notification.kind {
	case incrementalNotification:
		for _, instance := range notification.instances {
			serviceURL := generateUrl(instance)
			if serviceURL == nil {
				continue
			}
			notify.Notify(&registry.ServiceEvent{
				Action:  notification.eventType,
				Service: serviceURL,
			})
		}
	case fullSnapshotNotification:
		events := make([]*registry.ServiceEvent, 0, len(notification.instances))
		for _, instance := range notification.instances {
			serviceURL := generateUrl(instance)
			if serviceURL == nil {
				continue
			}
			events = append(events, &registry.ServiceEvent{
				Action:  remoting.EventTypeUpdate,
				Service: serviceURL,
			})
		}
		notify.NotifyAll(events, func() {})
	}
}

// createPolarisListener creates or reuses the watcher and binds the pending
// synchronous-load instances to the newly created listener.
func (pr *polarisRegistry) createPolarisListener(
	serviceName string,
	notify registry.NotifyListener,
) (*polarisListener, error) {
	key, cacheable, err := initialSubscribeInstancesKeyFor(serviceName, notify)
	if err != nil {
		return nil, err
	}
	return pr.createPolarisListenerWithIdentity(serviceName, key, cacheable)
}

func (pr *polarisRegistry) createPolarisListenerWithIdentity(
	serviceName string,
	key initialSubscribeInstancesKey,
	cacheable bool,
) (*polarisListener, error) {
	watcher, err := pr.createPolarisWatcher(serviceName)
	if err != nil {
		return nil, err
	}
	var entry *initialSubscribeInstancesEntry
	var instances []model.Instance
	mode := reconcileWithFullSnapshot
	if cacheable {
		entry, instances = pr.loadInitialSubscribeInstances(key)
		mode = reconcileWithBaseline
	}
	listener, err := newPolarisListener(watcher, instances, mode)
	if err != nil {
		return nil, err
	}
	if cacheable {
		pr.completeInitialSubscribeInstances(key, entry)
	}
	return listener, nil
}

// UnSubscribe returns nil if unsubscribing registry successfully. If not returns an error.
func (pr *polarisRegistry) UnSubscribe(url *common.URL, notifyListener registry.NotifyListener) error {
	// TODO wait polaris support it
	return perrors.New("UnSubscribe not support in polarisRegistry")
}

// LoadSubscribeInstances load subscribe instance
func (pr *polarisRegistry) LoadSubscribeInstances(url *common.URL, notify registry.NotifyListener) error {
	serviceName := url.Interface()
	key, cacheable, err := initialSubscribeInstancesKeyFor(serviceName, notify)
	if err != nil {
		return err
	}
	resp, err := pr.consumer.GetInstances(&api.GetInstancesRequest{
		GetInstancesRequest: model.GetInstancesRequest{
			Service:   serviceName,
			Namespace: pr.namespace,
		},
	})
	if err != nil {
		return perrors.New(fmt.Sprintf("could not query the instances for serviceName=%s,namespace=%s,error=%v",
			serviceName, pr.namespace, err))
	}
	validInstances := make([]model.Instance, 0, len(resp.Instances))
	initialEvents := make([]*registry.ServiceEvent, 0, len(resp.Instances))
	for i := range resp.Instances {
		if newUrl := generateUrl(resp.Instances[i]); newUrl != nil {
			validInstances = append(validInstances, resp.Instances[i])
			initialEvents = append(initialEvents, &registry.ServiceEvent{Action: remoting.EventTypeAdd, Service: newUrl})
		}
	}
	if cacheable {
		pr.storeInitialSubscribeInstances(key, validInstances)
	}
	for _, event := range initialEvents {
		notify.Notify(event)
	}
	return nil
}

func initialSubscribeInstancesKeyFor(
	serviceName string,
	notify registry.NotifyListener,
) (initialSubscribeInstancesKey, bool, error) {
	if isNilNotifyListener(notify) {
		return initialSubscribeInstancesKey{}, false, fmt.Errorf("notify listener type %T is nil", notify)
	}
	if !reflect.ValueOf(notify).Comparable() {
		return initialSubscribeInstancesKey{}, false, nil
	}
	return initialSubscribeInstancesKey{serviceName: serviceName, notify: notify}, true, nil
}

func isNilNotifyListener(notify registry.NotifyListener) bool {
	if notify == nil {
		return true
	}
	value := reflect.ValueOf(notify)
	switch value.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice:
		return value.IsNil()
	default:
		return false
	}
}

// storeInitialSubscribeInstances accumulates a defensive copy of every valid
// instance that successful LoadSubscribeInstances calls may have notified for
// this listener. Subscribe consumes the merged listener-scoped baseline once.
func (pr *polarisRegistry) storeInitialSubscribeInstances(key initialSubscribeInstancesKey, instances []model.Instance) {
	pr.initialSubscribeInstancesLock.Lock()
	defer pr.initialSubscribeInstancesLock.Unlock()

	var previous []model.Instance
	if entry := pr.initialSubscribeInstances[key]; entry != nil {
		previous = entry.instances
	}
	merged := mergeInitialSubscribeInstances(previous, instances)
	if len(merged) == 0 {
		return
	}
	if pr.initialSubscribeInstances == nil {
		pr.initialSubscribeInstances = make(map[initialSubscribeInstancesKey]*initialSubscribeInstancesEntry)
	}
	pr.initialSubscribeInstances[key] = &initialSubscribeInstancesEntry{
		instances: merged,
	}
}

func mergeInitialSubscribeInstances(previous, current []model.Instance) []model.Instance {
	currentByKey := make(map[model.InstanceKey]model.Instance, len(current))
	currentOrder := make([]model.InstanceKey, 0, len(current))
	for _, instance := range current {
		key := instance.GetInstanceKey()
		if _, exists := currentByKey[key]; !exists {
			currentOrder = append(currentOrder, key)
		}
		currentByKey[key] = instance
	}

	merged := make([]model.Instance, 0, len(previous)+len(currentOrder))
	seen := make(map[model.InstanceKey]struct{}, len(previous)+len(currentOrder))
	for _, instance := range previous {
		key := instance.GetInstanceKey()
		if _, exists := seen[key]; exists {
			continue
		}
		if newer, exists := currentByKey[key]; exists {
			instance = newer
		}
		merged = append(merged, instance)
		seen[key] = struct{}{}
	}
	for _, key := range currentOrder {
		if _, exists := seen[key]; exists {
			continue
		}
		merged = append(merged, currentByKey[key])
		seen[key] = struct{}{}
	}
	return merged
}

func (pr *polarisRegistry) loadInitialSubscribeInstances(
	key initialSubscribeInstancesKey,
) (*initialSubscribeInstancesEntry, []model.Instance) {
	pr.initialSubscribeInstancesLock.Lock()
	defer pr.initialSubscribeInstancesLock.Unlock()

	entry := pr.initialSubscribeInstances[key]
	if entry == nil {
		return nil, nil
	}
	return entry, copyInstances(entry.instances)
}

func (pr *polarisRegistry) completeInitialSubscribeInstances(
	key initialSubscribeInstancesKey,
	entry *initialSubscribeInstancesEntry,
) {
	pr.initialSubscribeInstancesLock.Lock()
	defer pr.initialSubscribeInstancesLock.Unlock()

	if pr.initialSubscribeInstances[key] == entry {
		delete(pr.initialSubscribeInstances, key)
	}
}

// GetURL returns polaris registry's url.
func (pr *polarisRegistry) GetURL() *common.URL {
	return pr.url
}

func (pr *polarisRegistry) createPolarisWatcher(serviceName string) (*PolarisServiceWatcher, error) {

	pr.listenerLock.Lock()
	defer pr.listenerLock.Unlock()

	if _, exist := pr.watchers[serviceName]; !exist {
		subscribeParam := &api.WatchServiceRequest{
			WatchServiceRequest: model.WatchServiceRequest{
				Key: model.ServiceKey{
					Namespace: pr.namespace,
					Service:   serviceName,
				},
			},
		}

		watcher, err := newPolarisWatcher(subscribeParam, pr.consumer)
		if err != nil {
			return nil, err
		}
		pr.watchers[serviceName] = watcher
	}

	return pr.watchers[serviceName], nil
}

// Destroy stop polaris registry.
func (pr *polarisRegistry) Destroy() {
	for i := range pr.registryUrls {
		url := pr.registryUrls[i]
		err := pr.UnRegister(url)
		logger.Infof("[Registry][Polaris] deRegister URL=%+v", url)
		if err != nil {
			logger.Errorf("[Registry][Polaris] deRegister URL=%+v err=%v", url, err)
		}
	}
	pr.initialSubscribeInstancesLock.Lock()
	clear(pr.initialSubscribeInstances)
	pr.initialSubscribeInstances = nil
	pr.initialSubscribeInstancesLock.Unlock()
}

// IsAvailable always return true when use polaris
func (pr *polarisRegistry) IsAvailable() bool {
	return true
}

// createRegisterParam convert dubbo url to polaris instance register request
func createRegisterParam(url *common.URL, serviceName string) *api.InstanceRegisterRequest {
	common.HandleRegisterIPAndPort(url)
	port, _ := strconv.Atoi(url.Port)

	metadata := make(map[string]string, len(url.GetParams()))
	url.RangeParams(func(key, value string) bool {
		metadata[key] = value
		return true
	})
	metadata[constant.PolarisDubboPath] = url.Path

	ver := url.GetParam("version", "")

	req := &api.InstanceRegisterRequest{
		InstanceRegisterRequest: model.InstanceRegisterRequest{
			Service:  serviceName,
			Host:     url.Ip,
			Port:     port,
			Protocol: &url.Protocol,
			Version:  &ver,
			Metadata: metadata,
		},
	}

	req.SetTTL(defaultHeartbeatIntervalSec)

	return req
}

// createDeregisterParam convert dubbo url to polaris instance deregister request
func createDeregisterParam(url *common.URL, serviceName string) *api.InstanceDeRegisterRequest {
	common.HandleRegisterIPAndPort(url)
	port, _ := strconv.Atoi(url.Port)
	return &api.InstanceDeRegisterRequest{
		InstanceDeRegisterRequest: model.InstanceDeRegisterRequest{
			Service: serviceName,
			Host:    url.Ip,
			Port:    port,
		},
	}
}
