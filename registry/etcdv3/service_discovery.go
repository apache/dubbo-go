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
	"time"
)

import (
	gxset "github.com/dubbogo/gost/container/set"
	gxpage "github.com/dubbogo/gost/page"
	"github.com/hashicorp/vault/sdk/helper/jsonutil"
	perrors "github.com/pkg/errors"
)

import (
	"github.com/apache/dubbo-go/common/constant"
	"github.com/apache/dubbo-go/common/extension"
	"github.com/apache/dubbo-go/common/logger"
	"github.com/apache/dubbo-go/config"
	"github.com/apache/dubbo-go/registry"
	"github.com/apache/dubbo-go/remoting"
	"github.com/apache/dubbo-go/remoting/etcdv3"
)

const (
	ROOT = "/services"
)

var (
	initLock sync.Mutex
)

func init() {
	extension.SetServiceDiscovery(constant.ETCDV3_KEY, newEtcdV3ServiceDiscovery)
}

// new etcd service discovery struct
type etcdV3ServiceDiscovery struct {
	// descriptor is a short string about the basic information of this instance
	descriptor string
	// client is current Etcdv3 client
	client *etcdv3.Client
	// serviceInstance is current serviceInstance
	serviceInstance *registry.ServiceInstance
	// services is when register or update will add service name
	services *gxset.HashSet
	// child listener
	childListenerMap map[string]*etcdv3.EventListener
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
		if nil == err {
			e.client.RegisterTemp(path, string(ins))
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

// ----------------- discovery -------------------
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

	return make([]registry.ServiceInstance, 0, 0)
}

// GetInstancesByPage will return a page containing instances of ServiceInstance with the serviceName
// the page will start at offset
func (e *etcdV3ServiceDiscovery) GetInstancesByPage(serviceName string, offset int, pageSize int) gxpage.Pager {

	all := e.GetInstances(serviceName)

	res := make([]interface{}, 0, pageSize)

	for i := offset; i < len(all) && i < offset+pageSize; i++ {
		res = append(res, all[i])
	}

	return gxpage.New(offset, pageSize, res, len(all))
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
	return gxpage.New(offset, pageSize, res, len(all))
}

// Batch get all instances by the specified service names
func (e *etcdV3ServiceDiscovery) GetRequestInstances(serviceNames []string, offset int, requestedSize int) map[string]gxpage.Pager {
	res := make(map[string]gxpage.Pager, len(serviceNames))
	for _, name := range serviceNames {
		res[name] = e.GetInstancesByPage(name, offset, requestedSize)
	}
	return res
}

// ----------------- event ----------------------
// AddListener adds a new ServiceInstancesChangedListener
// see addServiceInstancesChangedListener in Java
func (e *etcdV3ServiceDiscovery) AddListener(listener *registry.ServiceInstancesChangedListener) error {
	return e.registerSreviceWatcher(listener.ServiceName)
}

// DispatchEventByServiceName dispatches the ServiceInstancesChangedEvent to service instance whose name is serviceName
func (e *etcdV3ServiceDiscovery) DispatchEventByServiceName(serviceName string) error {
	return e.DispatchEventForInstances(serviceName, e.GetInstances(serviceName))
}

// DispatchEventForInstances dispatches the ServiceInstancesChangedEvent to target instances
func (e *etcdV3ServiceDiscovery) DispatchEventForInstances(serviceName string, instances []registry.ServiceInstance) error {
	return e.DispatchEvent(registry.NewServiceInstancesChangedEvent(serviceName, instances))
}

// DispatchEvent dispatches the event
func (e *etcdV3ServiceDiscovery) DispatchEvent(event *registry.ServiceInstancesChangedEvent) error {
	extension.GetGlobalDispatcher().Dispatch(event)
	return nil
}

// Convert instance to dubbo path
func toPath(instance registry.ServiceInstance) string {
	if instance == nil {
		return ""
	}
	// like: /services/servicename1/host(127.0.0.1)/8080
	return fmt.Sprintf("%s%d", ROOT+constant.PATH_SEPARATOR+instance.GetServiceName()+constant.PATH_SEPARATOR+instance.GetHost()+constant.KEY_SEPARATOR, instance.GetPort())
}

// to dubbo service path
func toParentPath(serviceName string) string {
	return ROOT + constant.PATH_SEPARATOR + serviceName
}

// register service watcher
func (e *etcdV3ServiceDiscovery) registerSreviceWatcher(serviceName string) error {

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

// when child data change should DispatchEventByServiceName
func (e *etcdV3ServiceDiscovery) DataChange(eventType remoting.Event) bool {

	if eventType.Action == remoting.EventTypeUpdate {
		instance := &registry.DefaultServiceInstance{}
		err := jsonutil.DecodeJSON([]byte(eventType.Content), &instance)
		if err != nil {
			instance.ServiceName = ""
		}

		if err := e.DispatchEventByServiceName(instance.ServiceName); err != nil {
			return false
		}
	}

	return true
}

// netEcdv3ServiceDiscovery
func newEtcdV3ServiceDiscovery(name string) (registry.ServiceDiscovery, error) {

	initLock.Lock()
	defer initLock.Unlock()

	sdc, ok := config.GetBaseConfig().GetServiceDiscoveries(name)
	if !ok || len(sdc.RemoteRef) == 0 {
		return nil, perrors.New("could not init the etcd service instance because the config is invalid")
	}

	remoteConfig, ok := config.GetBaseConfig().GetRemoteConfig(sdc.RemoteRef)
	if !ok {
		return nil, perrors.New("could not find the remote config for name: " + sdc.RemoteRef)
	}

	// init etcdv3 client
	timeout, err := time.ParseDuration(remoteConfig.TimeoutStr)
	if err != nil {
		logger.Errorf("timeout config %v is invalid,err is %v", remoteConfig.TimeoutStr, err.Error())
		return nil, perrors.WithMessagef(err, "new etcd service discovery(address:%v)", remoteConfig.Address)
	}

	logger.Infof("etcd address is: %v,timeout is:%s", remoteConfig.Address, timeout.String())

	client := etcdv3.NewServiceDiscoveryClient(
		etcdv3.WithName(etcdv3.RegistryETCDV3Client),
		etcdv3.WithTimeout(timeout),
		etcdv3.WithEndpoints(strings.Split(remoteConfig.Address, ",")...),
	)

	descriptor := fmt.Sprintf("etcd-service-discovery[%s]", remoteConfig.Address)

	return &etcdV3ServiceDiscovery{descriptor, client, nil, gxset.NewSet(), make(map[string]*etcdv3.EventListener, 0)}, nil
}
