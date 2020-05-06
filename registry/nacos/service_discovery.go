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
	"github.com/dubbogo/gost/container/set"
	"github.com/dubbogo/gost/page"
	"github.com/nacos-group/nacos-sdk-go/model"
	"github.com/nacos-group/nacos-sdk-go/vo"
	perrors "github.com/pkg/errors"
)

import (
	"github.com/apache/dubbo-go/common"
	"github.com/apache/dubbo-go/common/constant"
	"github.com/apache/dubbo-go/common/extension"
	"github.com/apache/dubbo-go/common/logger"
	"github.com/apache/dubbo-go/registry"
)

const (
	defaultGroup = "DEFAULT_GROUP"
	idKey        = "id"
)

// init will put the service discovery into extension
func init() {
	extension.SetServiceDiscovery(constant.NACOS_KEY, newNacosServiceDiscovery)
}

// nacosServiceDiscovery is the implementation of service discovery based on nacos.
// There is a problem, the go client for nacos does not support the id field.
// we will use the metadata to store the id of ServiceInstance
type nacosServiceDiscovery struct {
	nacosBaseRegistry
	group string
}

// Destroy will close the service discovery.
// Actually, it only marks the naming client as null and then return
func (n *nacosServiceDiscovery) Destroy() error {
	n.namingClient = nil
	return nil
}

// Register will register the service to nacos
func (n *nacosServiceDiscovery) Register(instance registry.ServiceInstance) error {
	ins := n.toRegisterInstance(instance)
	ok, err := n.namingClient.RegisterInstance(ins)
	if err != nil || !ok {
		return perrors.WithMessage(err, "Could not register the instance. "+instance.GetServiceName())
	}
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
	ok, err := n.namingClient.DeregisterInstance(n.toDeregisterInstance(instance))
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
	services, err := n.namingClient.GetAllServicesInfo(vo.GetAllServiceInfoParam{
		GroupName: n.group,
	})

	res := gxset.NewSet()
	if err != nil {
		logger.Errorf("Could not query the services: %v", err)
		return res
	}

	for _, e := range services {
		res.Add(e.Name)
	}
	return res
}

// GetInstances will return the instances of serviceName and the group
func (n *nacosServiceDiscovery) GetInstances(serviceName string) []registry.ServiceInstance {
	instances, err := n.namingClient.SelectAllInstances(vo.SelectAllInstancesParam{
		ServiceName: serviceName,
		GroupName:   n.group,
	})
	if err != nil {
		logger.Errorf("Could not query the instances for service: " + serviceName + ", group: " + n.group)
		return make([]registry.ServiceInstance, 0, 0)
	}
	res := make([]registry.ServiceInstance, 0, len(instances))
	for _, ins := range instances {
		metadata := ins.Metadata
		id := metadata[idKey]

		delete(metadata, idKey)

		res = append(res, &registry.DefaultServiceInstance{
			Id:          id,
			ServiceName: ins.ServiceName,
			Host:        ins.Ip,
			Port:        int(ins.Port),
			Enable:      ins.Enable,
			Healthy:     ins.Healthy,
			Metadata:    metadata,
		})
	}

	return res
}

// GetInstancesByPage will return the instances
// Due to nacos client does not support pagination, so we have to query all instances and then return part of them
func (n *nacosServiceDiscovery) GetInstancesByPage(serviceName string, offset int, pageSize int) gxpage.Pager {
	all := n.GetInstances(serviceName)
	res := make([]interface{}, 0, pageSize)
	// could not use res = all[a:b] here because the res should be []interface{}, not []ServiceInstance
	for i := offset; i < len(all) && i < offset+pageSize; i++ {
		res = append(res, all[i])
	}
	return gxpage.New(offset, pageSize, res, len(all))
}

// GetHealthyInstancesByPage will return the instance
// The nacos client has an API SelectInstances, which has a parameter call HealthyOnly.
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
	return gxpage.New(offset, pageSize, res, len(all))
}

// GetRequestInstances will return the instances
// The nacos client doesn't have batch API, so we should query those serviceNames one by one.
func (n *nacosServiceDiscovery) GetRequestInstances(serviceNames []string, offset int, requestedSize int) map[string]gxpage.Pager {
	res := make(map[string]gxpage.Pager, len(serviceNames))
	for _, name := range serviceNames {
		res[name] = n.GetInstancesByPage(name, offset, requestedSize)
	}
	return res
}

// AddListener will add a listener
func (n *nacosServiceDiscovery) AddListener(listener *registry.ServiceInstancesChangedListener) error {
	return n.namingClient.Subscribe(&vo.SubscribeParam{
		ServiceName: listener.ServiceName,
		SubscribeCallback: func(services []model.SubscribeService, err error) {
			if err != nil {
				logger.Errorf("Could not handle the subscribe notification because the err is not nil."+
					" service name: %s, err: %v", listener.ServiceName, err)
			}
			instances := make([]registry.ServiceInstance, 0, len(services))
			for _, service := range services {
				// we won't use the nacos instance id here but use our instance id
				metadata := service.Metadata
				id := metadata[idKey]

				delete(metadata, idKey)

				instances = append(instances, &registry.DefaultServiceInstance{
					Id:          id,
					ServiceName: service.ServiceName,
					Host:        service.Ip,
					Port:        int(service.Port),
					Enable:      service.Enable,
					Healthy:     true,
					Metadata:    metadata,
				})
			}

			e := n.DispatchEventForInstances(listener.ServiceName, instances)
			if e != nil {
				logger.Errorf("Dispatching event got exception, service name: %s, err: %v", listener.ServiceName, err)
			}
		},
	})
}

// DispatchEventByServiceName will dispatch the event for the service with the service name
func (n *nacosServiceDiscovery) DispatchEventByServiceName(serviceName string) error {
	return n.DispatchEventForInstances(serviceName, n.GetInstances(serviceName))
}

// DispatchEventForInstances will dispatch the event to those instances
func (n *nacosServiceDiscovery) DispatchEventForInstances(serviceName string, instances []registry.ServiceInstance) error {
	return n.DispatchEvent(registry.NewServiceInstancesChangedEvent(serviceName, instances))
}

// DispatchEvent will dispatch the event
func (n *nacosServiceDiscovery) DispatchEvent(event *registry.ServiceInstancesChangedEvent) error {
	extension.GetGlobalDispatcher().Dispatch(event)
	return nil
}

// toRegisterInstance convert the ServiceInstance to RegisterInstanceParam
// the Ephemeral will be true
func (n *nacosServiceDiscovery) toRegisterInstance(instance registry.ServiceInstance) vo.RegisterInstanceParam {
	metadata := instance.GetMetadata()
	if metadata == nil {
		metadata = make(map[string]string, 1)
	}
	metadata[idKey] = instance.GetId()
	return vo.RegisterInstanceParam{
		ServiceName: instance.GetServiceName(),
		Ip:          instance.GetHost(),
		Port:        uint64(instance.GetPort()),
		Metadata:    metadata,
		Enable:      instance.IsEnable(),
		Healthy:     instance.IsHealthy(),
		GroupName:   n.group,
		Ephemeral:   true,
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

// toDeregisterInstance will create new service discovery instance
func newNacosServiceDiscovery(url *common.URL) (registry.ServiceDiscovery, error) {

	base, err := newBaseRegistry(url)
	if err != nil {
		return nil, perrors.WithStack(err)
	}
	return &nacosServiceDiscovery{
		nacosBaseRegistry: base,
		group:             url.GetParam(constant.NACOS_GROUP, defaultGroup),
	}, nil
}
