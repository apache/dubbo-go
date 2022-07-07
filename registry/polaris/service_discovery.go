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
	"context"
	"fmt"
	"net/url"
	"sync"
	"time"
)

import (
	gxset "github.com/dubbogo/gost/container/set"
	gxpage "github.com/dubbogo/gost/hash/page"
	"github.com/dubbogo/gost/log/logger"

	perrors "github.com/pkg/errors"

	"github.com/polarismesh/polaris-go/api"
	"github.com/polarismesh/polaris-go/pkg/model"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/config"
	"dubbo.apache.org/dubbo-go/v3/registry"
	"dubbo.apache.org/dubbo-go/v3/remoting"
	"dubbo.apache.org/dubbo-go/v3/remoting/polaris"
)

// newPolarisServiceDiscovery will create new service discovery instance
func newPolarisServiceDiscovery() (registry.ServiceDiscovery, error) {
	metadataReportConfig := config.GetMetadataReportConfg()
	url := common.NewURLWithOptions(
		common.WithParams(make(url.Values)),
		common.WithParamsValue(constant.RegistryTimeoutKey, metadataReportConfig.Timeout))
	url.Location = metadataReportConfig.Address

	sdkCtx, namespace, err := polaris.GetPolarisConfig(url)

	if err != nil {
		return nil, perrors.WithMessage(err, "create polaris namingClient failed.")
	}

	descriptor := fmt.Sprintf("polaris-service-discovery[%s]", metadataReportConfig.Address)

	newInstance := &polarisServiceDiscovery{
		namespace:         namespace,
		descriptor:        descriptor,
		instanceLock:      &sync.RWMutex{},
		consumer:          api.NewConsumerAPIByContext(sdkCtx),
		provider:          api.NewProviderAPIByContext(sdkCtx),
		registryInstances: make(map[string]*PolarisHeartbeat),
		listenerLock:      &sync.RWMutex{},
		watchers:          make(map[string]*PolarisServiceWatcher),
	}
	return newInstance, nil
}

type polarisServiceDiscovery struct {
	namespace         string
	descriptor        string
	provider          api.ProviderAPI
	consumer          api.ConsumerAPI
	services          *gxset.HashSet
	instanceLock      *sync.RWMutex
	registryInstances map[string]*PolarisHeartbeat
	watchers          map[string]*PolarisServiceWatcher
	listenerLock      *sync.RWMutex
}

// Destroy destroy polarisServiceDiscovery, will do unregister all ServiceInstance
// and close polaris.ConsumerAPI and polaris.ProviderAPI
func (polaris *polarisServiceDiscovery) Destroy() error {
	for _, inst := range polaris.registryInstances {

		inst.cancel()

		err := polaris.Unregister(inst.instance)
		logger.Infof("Unregister polaris instance:%+v", inst)
		if err != nil {
			logger.Errorf("Unregister polaris instance:%+v, err:%+v", inst, err)
		}
	}
	polaris.provider.Destroy()
	polaris.consumer.Destroy()
	return nil
}

// Register do register for ServiceInstance
func (polaris *polarisServiceDiscovery) Register(instance registry.ServiceInstance) error {

	ins := convertToRegisterInstance(polaris.namespace, instance)
	resp, err := polaris.provider.Register(ins)
	if err != nil {
		return perrors.WithMessage(err, "could not register the instance. "+instance.GetServiceName())
	}

	if resp.Existed {
		logger.Warnf("instance already regist, namespace:%+v, service:%+v, host:%+v, port:%+v",
			polaris.namespace, instance.GetServiceName(), instance.GetHost(), instance.GetPort())
	}

	ctx, cancel := context.WithCancel(context.Background())
	go polaris.doHeartbeat(ctx, ins)

	polaris.instanceLock.Lock()
	defer polaris.instanceLock.Unlock()

	polaris.registryInstances[getInstanceKey(polaris.namespace, instance)] = &PolarisHeartbeat{
		cancel:   cancel,
		instance: instance,
	}
	polaris.services.Add(instance.GetServiceName())

	return nil
}

// Update update ServiceInstance info
func (polaris *polarisServiceDiscovery) Update(instance registry.ServiceInstance) error {
	err := polaris.Unregister(instance)
	if err != nil {
		return perrors.WithStack(err)
	}
	polaris.services.Add(instance.GetServiceName())
	return polaris.Register(instance)
}

// Unregister do Unregister for ServiceInstance
func (polaris *polarisServiceDiscovery) Unregister(instance registry.ServiceInstance) error {

	func() {
		polaris.instanceLock.Lock()
		defer polaris.instanceLock.Unlock()

		key := getInstanceKey(polaris.namespace, instance)
		if heartbeat, exist := polaris.registryInstances[key]; exist {
			heartbeat.cancel()
			delete(polaris.registryInstances, key)
		}
	}()

	err := polaris.provider.Deregister(convertToDeregisterInstance(polaris.namespace, instance))
	if err != nil {
		return perrors.WithMessage(err, "Could not unregister the instance. "+instance.GetServiceName())
	}

	polaris.services.Remove(instance.GetServiceName())
	return nil
}

func (polaris *polarisServiceDiscovery) GetDefaultPageSize() int {
	return registry.DefaultPageSize
}

func (polaris *polarisServiceDiscovery) GetServices() *gxset.HashSet {
	return polaris.services
}

// GetInstances will return all service instances with serviceName
func (polaris *polarisServiceDiscovery) GetInstances(serviceName string) []registry.ServiceInstance {
	resp, err := polaris.consumer.GetAllInstances(&api.GetAllInstancesRequest{
		GetAllInstancesRequest: model.GetAllInstancesRequest{
			Service:   serviceName,
			Namespace: polaris.namespace,
		},
	})

	if err != nil {
		logger.Errorf("Could not query the instances for service: %+v . It happened err %+v", serviceName, err)
		return make([]registry.ServiceInstance, 0)
	}
	res := make([]registry.ServiceInstance, 0, len(resp.Instances))
	for _, ins := range resp.Instances {
		metadata := ins.GetMetadata()
		res = append(res, &registry.DefaultServiceInstance{
			ID:          ins.GetId(),
			ServiceName: serviceName,
			Host:        ins.GetHost(),
			Port:        int(ins.GetPort()),
			Enable:      !ins.IsIsolated(),
			Healthy:     ins.IsHealthy(),
			Metadata:    metadata,
		})
	}
	return res
}

// GetInstancesByPage will return a page containing instances of ServiceInstance with the serviceName
// the page will start at offset
func (polaris *polarisServiceDiscovery) GetInstancesByPage(serviceName string, offset int, pageSize int) gxpage.Pager {
	all := polaris.GetInstances(serviceName)
	res := make([]interface{}, 0, pageSize)
	for i := offset; i < len(all) && i < offset+pageSize; i++ {
		res = append(res, all[i])
	}
	return gxpage.NewPage(offset, pageSize, res, len(all))
}

// GetHealthyInstancesByPage will return a page containing instances of ServiceInstance.
// The param healthy indices that the instance should be healthy or not.
// The page will start at offset
func (polaris *polarisServiceDiscovery) GetHealthyInstancesByPage(serviceName string, offset int, pageSize int, healthy bool) gxpage.Pager {
	all := polaris.GetInstances(serviceName)
	res := make([]interface{}, 0, pageSize)
	// could not use res = all[a:b] here because the res should be []interface{}, not []ServiceInstance
	var (
		i     = offset
		count = 0
	)
	for i < len(all) && count < pageSize {
		ins := all[i]
		if ins.IsHealthy() == healthy {
			res = append(res, all[i])
			count++
		}
		i++
	}
	return gxpage.NewPage(offset, pageSize, res, len(all))
}

// GetRequestInstances get all instances by the specified service names
func (polaris *polarisServiceDiscovery) GetRequestInstances(serviceNames []string, offset int, requestedSize int) map[string]gxpage.Pager {
	res := make(map[string]gxpage.Pager, len(serviceNames))
	for _, name := range serviceNames {
		res[name] = polaris.GetInstancesByPage(name, offset, requestedSize)
	}
	return res
}

// AddListener add listener for ServiceInstancesChangedListener
func (polaris *polarisServiceDiscovery) AddListener(listener registry.ServiceInstancesChangedListener) error {

	for _, val := range listener.GetServiceNames().Values() {
		serviceName := val.(string)
		watcher, err := polaris.createPolarisWatcherIfAbsent(serviceName)
		if err != nil {
			return err
		}

		watcher.AddSubscriber(func(et remoting.EventType, instances []model.Instance) {
			dubboInstances := make([]registry.ServiceInstance, 0, len(instances))
			for _, instance := range instances {
				dubboInstances = append(dubboInstances, &registry.DefaultServiceInstance{
					ID:          instance.GetId(),
					ServiceName: instance.GetService(),
					Host:        instance.GetHost(),
					Port:        int(instance.GetPort()),
					Enable:      !instance.IsIsolated(),
					Healthy:     instance.IsHealthy(),
					Metadata:    instance.GetMetadata(),
					GroupName:   instance.GetMetadata()[constant.PolarisDubboGroup],
				})
			}

			listener.OnEvent(registry.NewServiceInstancesChangedEvent(serviceName, dubboInstances))
		})
	}

	return nil
}

// createPolarisWatcherIfAbsent Calculate whether the corresponding PolarisWatcher needs to be created,
// if it does not exist, create one, otherwise return the existing one
func (polaris *polarisServiceDiscovery) createPolarisWatcherIfAbsent(serviceName string) (*PolarisServiceWatcher, error) {

	polaris.listenerLock.Lock()
	defer polaris.listenerLock.Unlock()

	if _, exist := polaris.watchers[serviceName]; !exist {
		subscribeParam := &api.WatchServiceRequest{
			WatchServiceRequest: model.WatchServiceRequest{
				Key: model.ServiceKey{
					Namespace: polaris.namespace,
					Service:   serviceName,
				},
			},
		}

		watcher, err := newPolarisWatcher(subscribeParam, polaris.consumer)
		if err != nil {
			return nil, err
		}
		polaris.watchers[serviceName] = watcher
	}

	return polaris.watchers[serviceName], nil
}

// String retuen descriptor
func (polaris *polarisServiceDiscovery) String() string {
	return polaris.descriptor
}

func convertToRegisterInstance(namespace string, instance registry.ServiceInstance) *api.InstanceRegisterRequest {

	health := instance.IsHealthy()
	isolate := instance.IsEnable()
	ttl := 5

	return &api.InstanceRegisterRequest{
		InstanceRegisterRequest: model.InstanceRegisterRequest{
			Service:   instance.GetServiceName(),
			Namespace: namespace,
			Host:      instance.GetHost(),
			Port:      instance.GetPort(),
			Protocol:  &protocolForDubboGO,
			Metadata:  instance.GetMetadata(),
			Healthy:   &health,
			Isolate:   &isolate,
			TTL:       &ttl,
		},
	}
}

func convertToDeregisterInstance(namespace string, instance registry.ServiceInstance) *api.InstanceDeRegisterRequest {
	return &api.InstanceDeRegisterRequest{
		InstanceDeRegisterRequest: model.InstanceDeRegisterRequest{
			Service:   instance.GetServiceName(),
			Namespace: namespace,
			Host:      instance.GetHost(),
			Port:      instance.GetPort(),
		},
	}
}

// doHeartbeat Since polaris does not support automatic reporting of instance heartbeats, separate logic is
//  needed to implement it
func (polaris *polarisServiceDiscovery) doHeartbeat(ctx context.Context, ins *api.InstanceRegisterRequest) {
	ticker := time.NewTicker(time.Duration(4) * time.Second)

	heartbeat := &api.InstanceHeartbeatRequest{
		InstanceHeartbeatRequest: model.InstanceHeartbeatRequest{
			Service:   ins.Service,
			Namespace: ins.Namespace,
			Host:      ins.Host,
			Port:      ins.Port,
		},
	}

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			polaris.provider.Heartbeat(heartbeat)
		}
	}
}
