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

//@Author: chuntaojun <liaochuntao@live.com>
//@Description:
//@Time: 2021/11/19 02:47

package polaris

import (
	"context"
	"fmt"
	"net/url"
	"sync"
	"time"

	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/common/logger"
	"dubbo.apache.org/dubbo-go/v3/config"
	"dubbo.apache.org/dubbo-go/v3/registry"
	"dubbo.apache.org/dubbo-go/v3/remoting/polaris"
	gxset "github.com/dubbogo/gost/container/set"
	gxpage "github.com/dubbogo/gost/hash/page"
	perrors "github.com/pkg/errors"
	"github.com/polarismesh/polaris-go/api"
	"github.com/polarismesh/polaris-go/pkg/model"
)

// newPolarisServiceDiscovery will create new service discovery instance
func newPolarisServiceDiscovery() (registry.ServiceDiscovery, error) {
	metadataReportConfig := config.GetMetadataReportConfg()
	url := common.NewURLWithOptions(
		common.WithParams(make(url.Values)),
		common.WithParamsValue(constant.REGISTRY_TIMEOUT_KEY, metadataReportConfig.Timeout))
	url.Location = metadataReportConfig.Address

	sdkCtx, namespace, err := polaris.GetPolarisConfig(url)

	if err != nil {
		return nil, perrors.WithMessage(err, "create polaris namingClient failed.")
	}

	descriptor := fmt.Sprintf("polaris-service-discovery[%s]", metadataReportConfig.Address)

	newInstance := &polarisServiceDiscovery{
		namespace:           namespace,
		descriptor:          descriptor,
		lock:                &sync.RWMutex{},
		consumer:            api.NewConsumerAPIByContext(sdkCtx),
		provider:            api.NewProviderAPIByContext(sdkCtx),
		registryInstances:   make(map[string]*heartbeatJob),
		instanceListenerMap: make(map[string]*gxset.HashSet),
	}
	return newInstance, nil
}

type polarisServiceDiscovery struct {
	namespace           string
	descriptor          string
	provider            api.ProviderAPI
	consumer            api.ConsumerAPI
	lock                *sync.RWMutex
	registryInstances   map[string]*heartbeatJob
	instanceListenerMap map[string]*gxset.HashSet
	services            *gxset.HashSet
	listenerLock        *sync.RWMutex
}

// Destroy
//  @receiver polaris
//  @return error
func (polaris *polarisServiceDiscovery) Destroy() error {
	for _, inst := range polaris.registryInstances {

		inst.cancel()

		err := polaris.Unregister(inst.instance)
		logger.Infof("Unregister nacos instance:%+v", inst)
		if err != nil {
			logger.Errorf("Unregister nacos instance:%+v, err:%+v", inst, err)
		}
	}
	polaris.provider.Destroy()
	polaris.consumer.Destroy()
	return nil
}

// Register
//  @receiver polaris
//  @param instance
//  @return error
func (polaris *polarisServiceDiscovery) Register(instance registry.ServiceInstance) error {

	ins := convertToRegisterInstance(polaris.namespace, instance)
	resp, err := polaris.provider.Register(ins)
	if err != nil {
		return perrors.WithMessage(err, "could not register the instance. "+instance.GetServiceName())
	}

	if resp.Existed {
		return fmt.Errorf("instance already regist, namespace:%+v, service:%+v, host:%+v, port:%+v",
			polaris.namespace, instance.GetServiceName(), instance.GetHost(), instance.GetPort())
	}

	ctx, cancel := context.WithCancel(context.Background())
	go polaris.doHeartbeat(ctx, ins)

	polaris.lock.Lock()
	defer polaris.lock.Unlock()
	polaris.registryInstances[getInstanceKey(polaris.namespace, instance)] = &heartbeatJob{
		cancel:   cancel,
		instance: instance,
	}

	polaris.services.Add(instance.GetServiceName())
	return nil
}

// Update
//  @receiver polaris
//  @param instance
//  @return error
func (polaris *polarisServiceDiscovery) Update(instance registry.ServiceInstance) error {
	err := polaris.Unregister(instance)
	if err != nil {
		return perrors.WithStack(err)
	}
	polaris.services.Add(instance.GetServiceName())
	return polaris.Register(instance)
}

// Unregister
//  @receiver polaris
//  @param instance
//  @return error
func (polaris *polarisServiceDiscovery) Unregister(instance registry.ServiceInstance) error {

	func() {
		polaris.lock.Lock()
		defer polaris.lock.Unlock()

		key := getInstanceKey(polaris.namespace, instance)
		if heartbeat, ok := polaris.registryInstances[key]; ok {
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

// GetDefaultPageSize
//  @receiver polaris
//  @return int
func (polaris *polarisServiceDiscovery) GetDefaultPageSize() int {
	return registry.DefaultPageSize
}

// GetServices
//  @receiver polaris
//  @return *gxset.HashSet
func (polaris *polarisServiceDiscovery) GetServices() *gxset.HashSet {
	return polaris.services
}

// GetInstances
//  @receiver polaris
//  @param serviceName
//  @return []registry.ServiceInstance
func (polaris *polarisServiceDiscovery) GetInstances(serviceName string) []registry.ServiceInstance {
	resp, err := polaris.consumer.GetInstances(&api.GetInstancesRequest{
		GetInstancesRequest: model.GetInstancesRequest{
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

// GetInstancesByPage
//  @receiver polaris
//  @param serviceName
//  @param offset
//  @param pageSize
//  @return gxpage.Pager
func (polaris *polarisServiceDiscovery) GetInstancesByPage(serviceName string, offset int, pageSize int) gxpage.Pager {
	all := polaris.GetInstances(serviceName)
	res := make([]interface{}, 0, pageSize)
	for i := offset; i < len(all) && i < offset+pageSize; i++ {
		res = append(res, all[i])
	}
	return gxpage.NewPage(offset, pageSize, res, len(all))
}

// GetHealthyInstancesByPage
//  @receiver polaris
//  @param serviceName
//  @param offset
//  @param pageSize
//  @param healthy
//  @return gxpage.Pager
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

// GetRequestInstances
//  @receiver polaris
//  @param serviceNames
//  @param offset
//  @param requestedSize
//  @return map
func (polaris *polarisServiceDiscovery) GetRequestInstances(serviceNames []string, offset int, requestedSize int) map[string]gxpage.Pager {
	res := make(map[string]gxpage.Pager, len(serviceNames))
	for _, name := range serviceNames {
		res[name] = polaris.GetInstancesByPage(name, offset, requestedSize)
	}
	return res
}

// AddListener
//  @receiver polaris
//  @param listener
//  @return error
func (polaris *polarisServiceDiscovery) AddListener(listener registry.ServiceInstancesChangedListener) error {
	return nil
}

// String
//  @receiver n
//  @return string
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

type heartbeatJob struct {
	cancel   context.CancelFunc
	instance registry.ServiceInstance
}
