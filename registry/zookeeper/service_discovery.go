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

package zookeeper

import (
	"fmt"
	"strconv"
	"strings"
	"sync"
)

import (
	gxset "github.com/dubbogo/gost/container/set"
	gxzookeeper "github.com/dubbogo/gost/database/kv/zk"
	gxpage "github.com/dubbogo/gost/hash/page"
	"github.com/dubbogo/gost/log/logger"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/common/extension"
	"dubbo.apache.org/dubbo-go/v3/registry"
	"dubbo.apache.org/dubbo-go/v3/remoting"
	"dubbo.apache.org/dubbo-go/v3/remoting/zookeeper"
	"dubbo.apache.org/dubbo-go/v3/remoting/zookeeper/curator_discovery"
)

const (
	rootPath = "/services"
)

func init() {
	extension.SetServiceDiscovery(constant.ZookeeperKey, newZookeeperServiceDiscovery)
}

type zookeeperServiceDiscovery struct {
	client              *gxzookeeper.ZookeeperClient
	csd                 *curator_discovery.ServiceDiscovery
	url                 *common.URL
	wg                  sync.WaitGroup
	cltLock             sync.Mutex
	listenLock          sync.Mutex
	done                chan struct{}
	rootPath            string
	listenNames         []string
	instanceListenerMap map[string]*gxset.HashSet
}

// newZookeeperServiceDiscovery the constructor of newZookeeperServiceDiscovery
func newZookeeperServiceDiscovery(url *common.URL) (registry.ServiceDiscovery, error) {
	zksd := &zookeeperServiceDiscovery{
		url:                 url,
		rootPath:            rootPath,
		instanceListenerMap: make(map[string]*gxset.HashSet),
	}
	if err := zookeeper.ValidateZookeeperClient(zksd, url.Location); err != nil {
		return nil, err
	}
	zksd.WaitGroup().Add(1) // zk client start successful, then wg +1
	go zookeeper.HandleClientRestart(zksd)
	zksd.csd = curator_discovery.NewServiceDiscovery(zksd.client, rootPath)
	return zksd, nil
}

// nolint
func (zksd *zookeeperServiceDiscovery) ZkClient() *gxzookeeper.ZookeeperClient {
	return zksd.client
}

// nolint
func (zksd *zookeeperServiceDiscovery) SetZkClient(client *gxzookeeper.ZookeeperClient) {
	zksd.client = client
}

// nolint
func (zksd *zookeeperServiceDiscovery) ZkClientLock() *sync.Mutex {
	return &zksd.cltLock
}

// nolint
func (zksd *zookeeperServiceDiscovery) WaitGroup() *sync.WaitGroup {
	return &zksd.wg
}

// nolint
func (zksd *zookeeperServiceDiscovery) Done() chan struct{} {
	return zksd.done
}

// RestartCallBack when zookeeper connection reconnect this function will be invoked.
// try to re-register service, and listen services
func (zksd *zookeeperServiceDiscovery) RestartCallBack() bool {
	zksd.csd.ReRegisterServices()
	zksd.listenLock.Lock()
	defer zksd.listenLock.Unlock()
	for _, name := range zksd.listenNames {
		zksd.csd.ListenServiceEvent(name, zksd)
	}
	return true
}

// nolint
func (zksd *zookeeperServiceDiscovery) GetURL() *common.URL {
	return zksd.url
}

// nolint
func (zksd *zookeeperServiceDiscovery) String() string {
	return fmt.Sprintf("zookeeper-service-discovery[%s]", zksd.url)
}

// Destroy will destroy the clinet.
func (zksd *zookeeperServiceDiscovery) Destroy() error {
	zksd.csd.Close()
	return nil
}

// Register will register service in zookeeper, instance convert to curator's service instance
// which define in curator-x-discovery.
func (zksd *zookeeperServiceDiscovery) Register(instance registry.ServiceInstance) error {
	cris := zksd.toCuratorInstance(instance)
	return zksd.csd.RegisterService(cris)
}

// Update will update service in zookeeper, instance convert to curator's service instance
// which define in curator-x-discovery, please refer to https://github.com/apache/curator.
func (zksd *zookeeperServiceDiscovery) Update(instance registry.ServiceInstance) error {
	cris := zksd.toCuratorInstance(instance)
	return zksd.csd.UpdateService(cris)
}

// Unregister will unregister the instance in zookeeper
func (zksd *zookeeperServiceDiscovery) Unregister(instance registry.ServiceInstance) error {
	cris := zksd.toCuratorInstance(instance)
	return zksd.csd.UnregisterService(cris)
}

// GetDefaultPageSize will return the constant registry.DefaultPageSize
func (zksd *zookeeperServiceDiscovery) GetDefaultPageSize() int {
	return registry.DefaultPageSize
}

// GetServices will return the all services in zookeeper
func (zksd *zookeeperServiceDiscovery) GetServices() *gxset.HashSet {
	services, err := zksd.csd.QueryForNames()
	res := gxset.NewSet()
	if err != nil {
		logger.Errorf("[zkServiceDiscovery] Could not query the services: %v", err)
		return res
	}
	for _, service := range services {
		res.Add(service)
	}
	return res
}

// GetInstances will return the instances in a service
func (zksd *zookeeperServiceDiscovery) GetInstances(serviceName string) []registry.ServiceInstance {
	criss, err := zksd.csd.QueryForInstances(serviceName)
	if err != nil {
		logger.Errorf("[zkServiceDiscovery] Could not query the instances for service{%s}, error = err{%v} ",
			serviceName, err)
		return make([]registry.ServiceInstance, 0)
	}
	iss := make([]registry.ServiceInstance, 0, len(criss))
	for _, cris := range criss {
		iss = append(iss, zksd.toZookeeperInstance(cris))
	}
	return iss
}

// GetInstancesByPage will return the instances
func (zksd *zookeeperServiceDiscovery) GetInstancesByPage(serviceName string, offset int, pageSize int) gxpage.Pager {
	all := zksd.GetInstances(serviceName)
	res := make([]interface{}, 0, pageSize)
	// could not use res = all[a:b] here because the res should be []interface{}, not []ServiceInstance
	for i := offset; i < len(all) && i < offset+pageSize; i++ {
		res = append(res, all[i])
	}
	return gxpage.NewPage(offset, pageSize, res, len(all))
}

// GetHealthyInstancesByPage will return the instance
// In zookeeper, all service instance's is healthy.
// However, the healthy parameter in this method maybe false. So we can not use that API.
// Thus, we must query all instances and then do filter
func (zksd *zookeeperServiceDiscovery) GetHealthyInstancesByPage(serviceName string, offset int, pageSize int, healthy bool) gxpage.Pager {
	all := zksd.GetInstances(serviceName)
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
func (zksd *zookeeperServiceDiscovery) GetRequestInstances(serviceNames []string, offset int, requestedSize int) map[string]gxpage.Pager {
	res := make(map[string]gxpage.Pager, len(serviceNames))
	for _, name := range serviceNames {
		res[name] = zksd.GetInstancesByPage(name, offset, requestedSize)
	}
	return res
}

// AddListener ListenServiceEvent will add a data listener in service
func (zksd *zookeeperServiceDiscovery) AddListener(listener registry.ServiceInstancesChangedListener) error {
	zksd.listenLock.Lock()
	defer zksd.listenLock.Unlock()

	for _, t := range listener.GetServiceNames().Values() {
		serviceName, ok := t.(string)
		if !ok {
			logger.Errorf("service name error %s", t)
			continue
		}
		zksd.listenNames = append(zksd.listenNames, serviceName)
		listenerSet, found := zksd.instanceListenerMap[serviceName]
		if !found {
			listenerSet = gxset.NewSet(listener)
			listenerSet.Add(listener)
			zksd.instanceListenerMap[serviceName] = listenerSet
		} else {
			listenerSet.Add(listener)
		}
	}

	for _, t := range listener.GetServiceNames().Values() {
		serviceName, ok := t.(string)
		if !ok {
			logger.Errorf("service name error %s", t)
			continue
		}
		zksd.csd.ListenServiceEvent(serviceName, zksd)
	}
	return nil
}

// DataChange implement DataListener's DataChange function
// to resolve event to do DispatchEventByServiceName
func (zksd *zookeeperServiceDiscovery) DataChange(eventType remoting.Event) bool {
	path := strings.TrimPrefix(eventType.Path, zksd.rootPath)
	path = strings.TrimPrefix(path, constant.PathSeparator)
	// get service name in zk path
	serviceName := strings.Split(path, constant.PathSeparator)[0]

	var err error
	instances := zksd.GetInstances(serviceName)
	for _, lis := range zksd.instanceListenerMap[serviceName].Values() {
		var instanceListener registry.ServiceInstancesChangedListener
		instanceListener = lis.(registry.ServiceInstancesChangedListener)
		err = instanceListener.OnEvent(registry.NewServiceInstancesChangedEvent(serviceName, instances))
	}

	if err != nil {
		logger.Errorf("[zkServiceDiscovery] DispatchEventByServiceName{%s} error = err{%v}", serviceName, err)
		return false
	}
	return true
}

// toCuratorInstance convert to curator's service instance
func (zksd *zookeeperServiceDiscovery) toCuratorInstance(instance registry.ServiceInstance) *curator_discovery.ServiceInstance {
	id := instance.GetHost() + ":" + strconv.Itoa(instance.GetPort())
	pl := make(map[string]interface{}, 8)
	pl["id"] = id
	pl["name"] = instance.GetServiceName()
	pl["metadata"] = instance.GetMetadata()
	pl["@class"] = "org.apache.dubbo.registry.zookeeper.ZookeeperInstance"
	cuis := &curator_discovery.ServiceInstance{
		Name:                instance.GetServiceName(),
		ID:                  id,
		Address:             instance.GetHost(),
		Port:                instance.GetPort(),
		Payload:             pl,
		RegistrationTimeUTC: 0,
	}
	return cuis
}

// toZookeeperInstance convert to registry's service instance
func (zksd *zookeeperServiceDiscovery) toZookeeperInstance(cris *curator_discovery.ServiceInstance) registry.ServiceInstance {
	pl, ok := cris.Payload.(map[string]interface{})
	if !ok {
		logger.Errorf("[zkServiceDiscovery] toZookeeperInstance{%s} payload is not map[string]interface{}", cris.ID)
		return nil
	}
	mdi, ok := pl["metadata"].(map[string]interface{})
	if !ok {
		logger.Errorf("[zkServiceDiscovery] toZookeeperInstance{%s} metadata is not map[string]interface{}", cris.ID)
		return nil
	}
	md := make(map[string]string, len(mdi))
	for k, v := range mdi {
		md[k] = fmt.Sprint(v)
	}
	return &registry.DefaultServiceInstance{
		ID:          cris.ID,
		ServiceName: cris.Name,
		Host:        cris.Address,
		Port:        cris.Port,
		Enable:      true,
		Healthy:     true,
		Metadata:    md,
	}
}
