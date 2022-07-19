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

package nacos

import (
	"fmt"
	"sync"
)

import (
	gxset "github.com/dubbogo/gost/container/set"
	nacosClient "github.com/dubbogo/gost/database/kv/nacos"
	gxpage "github.com/dubbogo/gost/hash/page"
	"github.com/dubbogo/gost/log/logger"

	"github.com/nacos-group/nacos-sdk-go/model"
	"github.com/nacos-group/nacos-sdk-go/vo"

	perrors "github.com/pkg/errors"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/common/extension"
	"dubbo.apache.org/dubbo-go/v3/registry"
	"dubbo.apache.org/dubbo-go/v3/remoting/nacos"
)

const (
	defaultGroup = constant.ServiceDiscoveryDefaultGroup
	idKey        = "id"
)

func init() {
	extension.SetServiceDiscovery(constant.NacosKey, newNacosServiceDiscovery)
}

// nacosServiceDiscovery is the implementation of service discovery based on nacos.
// There is a problem, the go namingClient for nacos does not support the id field.
// we will use the metadata to store the id of ServiceInstance
type nacosServiceDiscovery struct {
	group string
	// descriptor is a short string about the basic information of this instance
	descriptor string

	// namingClient is the Nacos' namingClient
	namingClient *nacosClient.NacosNamingClient
	// cache registry instances
	registryInstances []registry.ServiceInstance

	instanceListenerMap map[string]*gxset.HashSet
	listenerLock        sync.Mutex
}

// Destroy will close the service discovery.
// Actually, it only marks the naming namingClient as null and then return
func (n *nacosServiceDiscovery) Destroy() error {
	for _, inst := range n.registryInstances {
		err := n.Unregister(inst)
		logger.Infof("Unregister nacos instance:%+v", inst)
		if err != nil {
			logger.Errorf("Unregister nacos instance:%+v, err:%+v", inst, err)
		}
	}
	n.namingClient.Close()
	return nil
}

// Register will register the service to nacos
func (n *nacosServiceDiscovery) Register(instance registry.ServiceInstance) error {
	ins := n.toRegisterInstance(instance)
	ok, err := n.namingClient.Client().RegisterInstance(ins)
	if err != nil || !ok {
		return perrors.WithMessage(err, "Could not register the instance. "+instance.GetServiceName())
	}
	n.registryInstances = append(n.registryInstances, instance)
	return nil
}

// Update will update the information
// However, because nacos client doesn't support the update API,
// so we should unregister the instance and then register it again.
// the error handling is hard to implement
func (n *nacosServiceDiscovery) Update(instance registry.ServiceInstance) error {
	// TODO(wait for nacos support)
	err := n.Unregister(instance)
	if err != nil {
		return perrors.WithStack(err)
	}
	return n.Register(instance)
}

// Unregister will unregister the instance
func (n *nacosServiceDiscovery) Unregister(instance registry.ServiceInstance) error {
	ok, err := n.namingClient.Client().DeregisterInstance(n.toDeregisterInstance(instance))
	if err != nil || !ok {
		return perrors.WithMessage(err, "Could not unregister the instance. "+instance.GetServiceName())
	}
	return nil
}

// GetDefaultPageSize will return the constant registry.DefaultPageSize
func (n *nacosServiceDiscovery) GetDefaultPageSize() int {
	return registry.DefaultPageSize
}

// GetServices will return the all services
func (n *nacosServiceDiscovery) GetServices() *gxset.HashSet {
	services, err := n.namingClient.Client().GetAllServicesInfo(vo.GetAllServiceInfoParam{
		GroupName: n.group,
	})

	res := gxset.NewSet()
	if err != nil {
		logger.Errorf("Could not query the services: %v", err)
		return res
	}

	for _, e := range services.Doms {
		res.Add(e)
	}
	return res
}

// GetInstances will return the instances of serviceName and the group
func (n *nacosServiceDiscovery) GetInstances(serviceName string) []registry.ServiceInstance {
	instances, err := n.namingClient.Client().SelectAllInstances(vo.SelectAllInstancesParam{
		ServiceName: serviceName,
		GroupName:   n.group,
	})
	if err != nil {
		logger.Errorf("Could not query the instances for service: %+v, group: %+v . It happened err %+v",
			serviceName, n.group, err)
		return make([]registry.ServiceInstance, 0)
	}
	res := make([]registry.ServiceInstance, 0, len(instances))
	for _, ins := range instances {
		metadata := ins.Metadata
		id := metadata[idKey]

		delete(metadata, idKey)

		res = append(res, &registry.DefaultServiceInstance{
			ID: id,
			// ins.ServiceName is nacos service name like 'DEFAULT_GROUP@@MyAppName",
			// which is not the service name we wanted, so we use serviceName directly.
			ServiceName: serviceName,
			Host:        ins.Ip,
			Port:        int(ins.Port),
			Enable:      ins.Enable,
			Healthy:     ins.Healthy,
			Metadata:    metadata,
			GroupName:   n.group,
		})
	}
	return res
}

// GetInstancesByPage will return the instances
// Due to nacos namingClient does not support pagination, so we have to query all instances and then return part of them
func (n *nacosServiceDiscovery) GetInstancesByPage(serviceName string, offset int, pageSize int) gxpage.Pager {
	all := n.GetInstances(serviceName)
	res := make([]interface{}, 0, pageSize)
	// could not use res = all[a:b] here because the res should be []interface{}, not []ServiceInstance
	for i := offset; i < len(all) && i < offset+pageSize; i++ {
		res = append(res, all[i])
	}
	return gxpage.NewPage(offset, pageSize, res, len(all))
}

// GetHealthyInstancesByPage will return the instance
// The nacos namingClient has an API SelectInstances, which has a parameter call HealthyOnly.
// However, the healthy parameter in this method maybe false. So we can not use that API.
// Thus, we must query all instances and then do filter
func (n *nacosServiceDiscovery) GetHealthyInstancesByPage(serviceName string, offset int, pageSize int, healthy bool) gxpage.Pager {
	all := n.GetInstances(serviceName)
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

// GetRequestInstances will return the instances
// The nacos namingClient doesn't have batch API, so we should query those serviceNames one by one.
func (n *nacosServiceDiscovery) GetRequestInstances(serviceNames []string, offset int, requestedSize int) map[string]gxpage.Pager {
	res := make(map[string]gxpage.Pager, len(serviceNames))
	for _, name := range serviceNames {
		res[name] = n.GetInstancesByPage(name, offset, requestedSize)
	}
	return res
}

func (n *nacosServiceDiscovery) registerInstanceListener(listener registry.ServiceInstancesChangedListener) {
	n.listenerLock.Lock()
	defer n.listenerLock.Unlock()

	for _, t := range listener.GetServiceNames().Values() {
		serviceName, ok := t.(string)
		if !ok {
			logger.Errorf("service name error %s", t)
			continue
		}
		listenerSet, found := n.instanceListenerMap[serviceName]
		if !found {
			listenerSet = gxset.NewSet(listener)
			listenerSet.Add(listener)
			n.instanceListenerMap[serviceName] = listenerSet
		} else {
			listenerSet.Add(listener)
		}
	}
}

// AddListener will add a listener
func (n *nacosServiceDiscovery) AddListener(listener registry.ServiceInstancesChangedListener) error {
	n.registerInstanceListener(listener)
	for _, t := range listener.GetServiceNames().Values() {
		serviceName := t.(string)
		err := n.namingClient.Client().Subscribe(&vo.SubscribeParam{
			ServiceName: serviceName,
			GroupName:   n.group,
			SubscribeCallback: func(services []model.SubscribeService, err error) {
				if err != nil {
					logger.Errorf("Could not handle the subscribe notification because the err is not nil."+
						" service name: %s, err: %v", serviceName, err)
				}
				instances := make([]registry.ServiceInstance, 0, len(services))
				for _, service := range services {
					// we won't use the nacos instance id here but use our instance id
					metadata := service.Metadata
					id := metadata[idKey]

					delete(metadata, idKey)

					instances = append(instances, &registry.DefaultServiceInstance{
						ID:          id,
						ServiceName: service.ServiceName,
						Host:        service.Ip,
						Port:        int(service.Port),
						Enable:      service.Enable,
						Healthy:     true,
						Metadata:    metadata,
						GroupName:   n.group,
					})
				}

				var e error
				for _, lis := range n.instanceListenerMap[serviceName].Values() {
					var instanceListener registry.ServiceInstancesChangedListener
					instanceListener = lis.(registry.ServiceInstancesChangedListener)
					e = instanceListener.OnEvent(registry.NewServiceInstancesChangedEvent(serviceName, instances))
				}

				if e != nil {
					logger.Errorf("Dispatching event got exception, service name: %s, err: %v", serviceName, err)
				}
			},
		})
		if err != nil {
			return err
		}
	}
	return nil
}

// toRegisterInstance convert the ServiceInstance to RegisterInstanceParam
// the Ephemeral will be true
func (n *nacosServiceDiscovery) toRegisterInstance(instance registry.ServiceInstance) vo.RegisterInstanceParam {
	metadata := instance.GetMetadata()
	if metadata == nil {
		metadata = make(map[string]string, 1)
	}
	metadata[idKey] = instance.GetID()
	return vo.RegisterInstanceParam{
		ServiceName: instance.GetServiceName(),
		Ip:          instance.GetHost(),
		Port:        uint64(instance.GetPort()),
		Metadata:    metadata,
		// We must specify the weight since Java nacos namingClient will ignore the instance whose weight is 0
		Weight:    1,
		Enable:    instance.IsEnable(),
		Healthy:   instance.IsHealthy(),
		GroupName: n.group,
		Ephemeral: true,
	}
}

// toDeregisterInstance will convert the ServiceInstance to DeregisterInstanceParam
func (n *nacosServiceDiscovery) toDeregisterInstance(instance registry.ServiceInstance) vo.DeregisterInstanceParam {
	return vo.DeregisterInstanceParam{
		ServiceName: instance.GetServiceName(),
		Ip:          instance.GetHost(),
		Port:        uint64(instance.GetPort()),
		GroupName:   n.group,
	}
}

func (n *nacosServiceDiscovery) String() string {
	return n.descriptor
}

// newNacosServiceDiscovery will create new service discovery instance
func newNacosServiceDiscovery(url *common.URL) (registry.ServiceDiscovery, error) {
	discoveryURL := common.NewURLWithOptions(
		common.WithParams(url.GetParams()),
		common.WithParamsValue(constant.TimeoutKey, url.GetParam(constant.RegistryTimeoutKey, constant.DefaultRegTimeout)),
		common.WithParamsValue(constant.NacosGroupKey, url.GetParam(constant.RegistryGroupKey, defaultGroup)),
		common.WithParamsValue(constant.NacosUsername, url.Username),
		common.WithParamsValue(constant.NacosPassword, url.Password),
		common.WithParamsValue(constant.NacosNamespaceID, url.GetParam(constant.RegistryNamespaceKey, "")))
	discoveryURL.Location = url.Location
	discoveryURL.Username = url.Username
	discoveryURL.Password = url.Password
	client, err := nacos.NewNacosClientByURL(discoveryURL)
	if err != nil {
		return nil, perrors.WithMessage(err, "create nacos namingClient failed.")
	}

	descriptor := fmt.Sprintf("nacos-service-discovery[%s]", discoveryURL.Location)

	group := url.GetParam(constant.RegistryGroupKey, defaultGroup)
	newInstance := &nacosServiceDiscovery{
		group:               group,
		namingClient:        client,
		descriptor:          descriptor,
		registryInstances:   []registry.ServiceInstance{},
		instanceListenerMap: make(map[string]*gxset.HashSet),
	}
	return newInstance, nil
}
