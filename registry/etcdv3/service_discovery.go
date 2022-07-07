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

package etcdv3

import (
	"fmt"
	"strings"
	"sync"
)

import (
	gxset "github.com/dubbogo/gost/container/set"
	gxetcd "github.com/dubbogo/gost/database/kv/etcd/v3"
	gxpage "github.com/dubbogo/gost/hash/page"
	"github.com/dubbogo/gost/log/logger"

	"github.com/hashicorp/vault/sdk/helper/jsonutil"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/common/extension"
	"dubbo.apache.org/dubbo-go/v3/registry"
	"dubbo.apache.org/dubbo-go/v3/remoting"
	"dubbo.apache.org/dubbo-go/v3/remoting/etcdv3"
)

const (
	ROOT = "/services"
)

var initLock sync.Mutex

func init() {
	extension.SetServiceDiscovery(constant.EtcdV3Key, newEtcdV3ServiceDiscovery)
}

// new etcd service discovery struct
type etcdV3ServiceDiscovery struct {
	// descriptor is a short string about the basic information of this instance
	descriptor string
	// client is current Etcdv3 client
	client *gxetcd.Client
	// serviceInstance is current serviceInstance
	serviceInstance *registry.ServiceInstance
	// services is when register or update will add service name
	services *gxset.HashSet
	// child listener
	childListenerMap    map[string]*etcdv3.EventListener
	instanceListenerMap map[string]*gxset.HashSet
}

// basic information of this instance
func (e *etcdV3ServiceDiscovery) String() string {
	return e.descriptor
}

// Destroy service discovery
func (e *etcdV3ServiceDiscovery) Destroy() error {
	if e.client != nil {
		e.client.Close()
	}
	return nil
}

// Register will register an instance of ServiceInstance to registry
func (e *etcdV3ServiceDiscovery) Register(instance registry.ServiceInstance) error {
	e.serviceInstance = &instance

	path := toPath(instance)

	if nil != e.client {
		ins, err := jsonutil.EncodeJSON(instance)
		if err == nil {
			err = e.client.RegisterTemp(path, string(ins))
			if err != nil {
				logger.Errorf("cannot register the instance: %s", string(ins), err)
			} else {
				e.services.Add(instance.GetServiceName())
			}
		}
	}

	return nil
}

// Update will update the data of the instance in registry
func (e *etcdV3ServiceDiscovery) Update(instance registry.ServiceInstance) error {
	path := toPath(instance)

	if nil != e.client {
		ins, err := jsonutil.EncodeJSON(instance)
		if err == nil {
			if err = e.client.RegisterTemp(path, string(ins)); err != nil {
				logger.Warnf("etcdV3ServiceDiscovery.client.RegisterTemp(path:%v, instance:%v) = error:%v",
					path, string(ins), err)
			}
			e.services.Add(instance.GetServiceName())
		}
	}

	return nil
}

// Unregister will unregister this instance from registry
func (e *etcdV3ServiceDiscovery) Unregister(instance registry.ServiceInstance) error {
	path := toPath(instance)

	if nil != e.client {
		err := e.client.Delete(path)
		e.services.Remove(instance.GetServiceName())
		e.serviceInstance = nil
		return err
	}

	return nil
}

// GetDefaultPageSize will return the default page size
func (e *etcdV3ServiceDiscovery) GetDefaultPageSize() int {
	return registry.DefaultPageSize
}

// GetServices will return the all service names.
func (e *etcdV3ServiceDiscovery) GetServices() *gxset.HashSet {
	return e.services
}

// GetInstances will return all service instances with serviceName
func (e *etcdV3ServiceDiscovery) GetInstances(serviceName string) []registry.ServiceInstance {
	if nil != e.client {
		// get keys and values
		_, vList, err := e.client.GetChildrenKVList(toParentPath(serviceName))
		if nil == err {
			serviceInstances := make([]registry.ServiceInstance, 0, len(vList))
			for _, v := range vList {
				instance := &registry.DefaultServiceInstance{}
				err = jsonutil.DecodeJSON([]byte(v), &instance)
				if nil == err {
					serviceInstances = append(serviceInstances, instance)
				}
			}
			return serviceInstances
		}
		logger.Infof("could not getChildrenKVList the err is:%v", err)
	}

	return make([]registry.ServiceInstance, 0)
}

// GetInstancesByPage will return a page containing instances of ServiceInstance with the serviceName
// the page will start at offset
func (e *etcdV3ServiceDiscovery) GetInstancesByPage(serviceName string, offset int, pageSize int) gxpage.Pager {
	all := e.GetInstances(serviceName)

	res := make([]interface{}, 0, pageSize)

	for i := offset; i < len(all) && i < offset+pageSize; i++ {
		res = append(res, all[i])
	}

	return gxpage.NewPage(offset, pageSize, res, len(all))
}

// GetHealthyInstancesByPage will return a page containing instances of ServiceInstance.
// The param healthy indices that the instance should be healthy or not.
// The page will start at offset
func (e *etcdV3ServiceDiscovery) GetHealthyInstancesByPage(serviceName string, offset int, pageSize int, healthy bool) gxpage.Pager {
	all := e.GetInstances(serviceName)
	res := make([]interface{}, 0, pageSize)

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

// GetRequestInstances Batch get all instances by the specified service names
func (e *etcdV3ServiceDiscovery) GetRequestInstances(serviceNames []string, offset int, requestedSize int) map[string]gxpage.Pager {
	res := make(map[string]gxpage.Pager, len(serviceNames))
	for _, name := range serviceNames {
		res[name] = e.GetInstancesByPage(name, offset, requestedSize)
	}
	return res
}

// AddListener adds a new ServiceInstancesChangedListenerImpl
// see addServiceInstancesChangedListener in Java
func (e *etcdV3ServiceDiscovery) AddListener(listener registry.ServiceInstancesChangedListener) error {
	for _, t := range listener.GetServiceNames().Values() {
		err := e.registerServiceInstanceListener(t.(string), listener)
		if err != nil {
			return err
		}

		err = e.registerServiceWatcher(t.(string))
		if err != nil {
			return err
		}
	}
	return nil
}

// Convert instance to dubbo path
func toPath(instance registry.ServiceInstance) string {
	if instance == nil {
		return ""
	}
	// like: /services/servicename1/host(127.0.0.1)/8080
	return fmt.Sprintf("%s%d", ROOT+constant.PathSeparator+instance.GetServiceName()+constant.PathSeparator+instance.GetHost()+constant.KeySeparator, instance.GetPort())
}

// to dubbo service path
func toParentPath(serviceName string) string {
	return ROOT + constant.PathSeparator + serviceName
}

// register service instance listener, instance listener and watcher are matched through serviceName
func (e *etcdV3ServiceDiscovery) registerServiceInstanceListener(serviceName string, listener registry.ServiceInstancesChangedListener) error {
	initLock.Lock()
	defer initLock.Unlock()

	set, found := e.instanceListenerMap[serviceName]
	if !found {
		set = gxset.NewSet(listener)
		set.Add(listener)
		e.instanceListenerMap[serviceName] = set
		return nil
	}
	set.Add(listener)
	return nil
}

// register service watcher
func (e *etcdV3ServiceDiscovery) registerServiceWatcher(serviceName string) error {
	initLock.Lock()
	defer initLock.Unlock()

	path := toParentPath(serviceName)

	listener, found := e.childListenerMap[serviceName]

	if !found {
		listener = etcdv3.NewEventListener(e.client)
		e.childListenerMap[serviceName] = listener
	}

	listener.ListenServiceEvent(path, e)

	return nil
}

// DataChange when child data change should DispatchEventByServiceName
func (e *etcdV3ServiceDiscovery) DataChange(eventType remoting.Event) bool {
	if eventType.Action == remoting.EventTypeUpdate {
		instance := &registry.DefaultServiceInstance{}
		err := jsonutil.DecodeJSON([]byte(eventType.Content), &instance)
		if err != nil {
			instance.ServiceName = ""
		}

		// notify instance listener instance change
		name := instance.ServiceName
		instances := e.GetInstances(name)
		for _, lis := range e.instanceListenerMap[instance.ServiceName].Values() {
			var instanceLis registry.ServiceInstancesChangedListener
			instanceLis = lis.(registry.ServiceInstancesChangedListener)
			err = instanceLis.OnEvent(registry.NewServiceInstancesChangedEvent(name, instances))
		}
		if err != nil {
			return false
		}
	}

	return true
}

// newEtcdv3ServiceDiscovery
func newEtcdV3ServiceDiscovery(url *common.URL) (registry.ServiceDiscovery, error) {
	initLock.Lock()
	defer initLock.Unlock()

	timeout := url.GetParamDuration(constant.RegistryTimeoutKey, constant.DefaultRegTimeout)

	logger.Infof("etcd address is: %v,timeout is:%s", url.Location, timeout.String())

	client := etcdv3.NewServiceDiscoveryClient(
		gxetcd.WithName(gxetcd.RegistryETCDV3Client),
		gxetcd.WithTimeout(timeout),
		gxetcd.WithEndpoints(strings.Split(url.Location, ",")...),
	)

	descriptor := fmt.Sprintf("etcd-service-discovery[%s]", url.Location)

	return &etcdV3ServiceDiscovery{
		descriptor:          descriptor,
		client:              client,
		serviceInstance:     nil,
		services:            gxset.NewSet(),
		childListenerMap:    make(map[string]*etcdv3.EventListener),
		instanceListenerMap: make(map[string]*gxset.HashSet)}, nil
}
