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
		initialSubscribeInstances: make(map[initialSubscribeInstancesKey][]model.Instance),
		initialSubscribeVersions:  make(map[initialSubscribeInstancesKey]uint64),
	}

	return pRegistry, nil
}

type initialSubscribeInstancesKey struct {
	serviceName string
	notify      registry.NotifyListener
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
	// clears the instances, version, and listener reference after a successful
	// transfer. The watcher does not hold a global initial baseline.
	initialSubscribeInstancesLock sync.Mutex
	initialSubscribeInstances     map[initialSubscribeInstancesKey][]model.Instance
	initialSubscribeVersions      map[initialSubscribeInstancesKey]uint64
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
	timer := time.NewTimer(time.Duration(RegistryConnDelay) * time.Second)
	defer timer.Stop()

	for {
		listener, err := pr.createPolarisListener(serviceName, notifyListener, newPolarisListener)
		if err != nil {
			logger.Warnf("[Registry][Polaris] create listener failed, service=%s, err=%v", serviceName, perrors.WithStack(err))
			<-timer.C
			timer.Reset(time.Duration(RegistryConnDelay) * time.Second)
			continue
		}

		for {
			serviceEvent, err := listener.Next()

			if err != nil {
				logger.Warnf("[Registry][Polaris] Selector.watch() = err=%v", perrors.WithStack(err))
				listener.Close()
				return err
			}
			logger.Infof("[Registry][Polaris] update begin, event=%v", serviceEvent.String())
			notifyListener.Notify(serviceEvent)
		}
	}
}

// createPolarisListener creates or reuses the watcher and binds the pending
// synchronous-load instances to the newly created listener.
func (pr *polarisRegistry) createPolarisListener(
	serviceName string,
	notify registry.NotifyListener,
	newListener func(*PolarisServiceWatcher, []model.Instance) (*polarisListener, error),
) (*polarisListener, error) {
	key := newInitialSubscribeInstancesKey(serviceName, notify)
	watcher, err := pr.createPolarisWatcher(serviceName)
	if err != nil {
		return nil, err
	}
	initialSubscribeInstances, found, initialSubscribeVersion := pr.takeInitialSubscribeInstancesWithVersion(key)
	listener, err := newListener(watcher, initialSubscribeInstances)
	if err != nil {
		if found {
			pr.restoreInitialSubscribeInstances(key, initialSubscribeInstances, initialSubscribeVersion)
		}
		return nil, err
	}
	pr.completeInitialSubscribeInstances(key, initialSubscribeVersion)
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
	key := newInitialSubscribeInstancesKey(serviceName, notify)
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
	initialSubscribeInstances := make([]model.Instance, 0, len(resp.Instances))
	for i := range resp.Instances {
		if newUrl := generateUrl(resp.Instances[i]); newUrl != nil {
			notify.Notify(&registry.ServiceEvent{Action: remoting.EventTypeAdd, Service: newUrl})
			initialSubscribeInstances = append(initialSubscribeInstances, resp.Instances[i])
		}
	}
	pr.storeInitialSubscribeInstances(key, initialSubscribeInstances)
	return nil
}

func newInitialSubscribeInstancesKey(serviceName string, notify registry.NotifyListener) initialSubscribeInstancesKey {
	value := reflect.ValueOf(notify)
	if !value.IsValid() || value.Kind() != reflect.Pointer || value.IsNil() {
		// This service-scoped fallback provides compatibility and panic safety for
		// unsupported listener identities. It does not guarantee precise isolation
		// between multiple unsupported listeners for the same service.
		return initialSubscribeInstancesKey{serviceName: serviceName}
	}
	return initialSubscribeInstancesKey{serviceName: serviceName, notify: notify}
}

// storeInitialSubscribeInstances keeps valid instances notified by
// LoadSubscribeInstances until Subscribe transfers them to the same supported
// notify-listener identity. A successful transfer clears the instances,
// version, and listener reference. Watchers do not hold a global initial
// baseline.
func (pr *polarisRegistry) storeInitialSubscribeInstances(key initialSubscribeInstancesKey, instances []model.Instance) {
	pr.initialSubscribeInstancesLock.Lock()
	defer pr.initialSubscribeInstancesLock.Unlock()

	if pr.initialSubscribeVersions == nil {
		pr.initialSubscribeVersions = make(map[initialSubscribeInstancesKey]uint64)
	}
	pr.initialSubscribeVersions[key]++
	if len(instances) == 0 {
		delete(pr.initialSubscribeInstances, key)
		return
	}
	if pr.initialSubscribeInstances == nil {
		pr.initialSubscribeInstances = make(map[initialSubscribeInstancesKey][]model.Instance)
	}
	pr.initialSubscribeInstances[key] = append([]model.Instance(nil), instances...)
}

// takeInitialSubscribeInstances transfers synchronous LoadSubscribeInstances
// state to the matching asynchronous listener exactly once.
func (pr *polarisRegistry) takeInitialSubscribeInstances(serviceName string, notify registry.NotifyListener) ([]model.Instance, bool) {
	key := newInitialSubscribeInstancesKey(serviceName, notify)
	instances, ok, initialSubscribeVersion := pr.takeInitialSubscribeInstancesWithVersion(key)
	pr.completeInitialSubscribeInstances(key, initialSubscribeVersion)
	return instances, ok
}

func (pr *polarisRegistry) takeInitialSubscribeInstancesWithVersion(key initialSubscribeInstancesKey) ([]model.Instance, bool, uint64) {
	pr.initialSubscribeInstancesLock.Lock()
	defer pr.initialSubscribeInstancesLock.Unlock()

	instances, ok := pr.initialSubscribeInstances[key]
	if !ok {
		return nil, false, pr.initialSubscribeVersions[key]
	}
	delete(pr.initialSubscribeInstances, key)
	return copyInstances(instances), true, pr.initialSubscribeVersions[key]
}

func (pr *polarisRegistry) completeInitialSubscribeInstances(key initialSubscribeInstancesKey, initialSubscribeVersion uint64) {
	pr.initialSubscribeInstancesLock.Lock()
	defer pr.initialSubscribeInstancesLock.Unlock()

	if pr.initialSubscribeVersions[key] != initialSubscribeVersion {
		return
	}
	if _, pending := pr.initialSubscribeInstances[key]; pending {
		return
	}
	delete(pr.initialSubscribeVersions, key)
}

// restoreInitialSubscribeInstances puts back instances when listener creation
// fails. A newer synchronous load wins if it already stored another version.
func (pr *polarisRegistry) restoreInitialSubscribeInstances(key initialSubscribeInstancesKey, instances []model.Instance, initialSubscribeVersion uint64) {
	pr.initialSubscribeInstancesLock.Lock()
	defer pr.initialSubscribeInstancesLock.Unlock()

	if pr.initialSubscribeVersions[key] != initialSubscribeVersion {
		return
	}
	if _, exists := pr.initialSubscribeInstances[key]; exists {
		return
	}
	if pr.initialSubscribeInstances == nil {
		pr.initialSubscribeInstances = make(map[initialSubscribeInstancesKey][]model.Instance)
	}
	pr.initialSubscribeInstances[key] = copyInstances(instances)
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
