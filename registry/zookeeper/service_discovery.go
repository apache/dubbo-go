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
	"net/url"
	"strconv"
	"strings"
	"sync"
)

import (
	"github.com/dubbogo/gost/container/set"
	"github.com/dubbogo/gost/page"
	perrors "github.com/pkg/errors"
)

import (
	"github.com/apache/dubbo-go/common"
	"github.com/apache/dubbo-go/common/constant"
	"github.com/apache/dubbo-go/common/extension"
	"github.com/apache/dubbo-go/common/logger"
	"github.com/apache/dubbo-go/config"
	"github.com/apache/dubbo-go/registry"
	"github.com/apache/dubbo-go/remoting"
	"github.com/apache/dubbo-go/remoting/zookeeper"
	"github.com/apache/dubbo-go/remoting/zookeeper/curator_discovery"
)

const (
	// RegistryZkClient zk client name
	ServiceDiscoveryZkClient = "zk service discovery"
)

var (
	// 16 would be enough. We won't use concurrentMap because in most cases, there are not race condition
	instanceMap = make(map[string]registry.ServiceDiscovery, 16)
	initLock    sync.Mutex
)

// init will put the service discovery into extension
func init() {
	extension.SetServiceDiscovery(constant.ZOOKEEPER_KEY, newZookeeperServiceDiscovery)
}

type zookeeperServiceDiscovery struct {
	client      *zookeeper.ZookeeperClient
	csd         *curator_discovery.ServiceDiscovery
	listener    *zookeeper.ZkEventListener
	url         *common.URL
	wg          sync.WaitGroup
	cltLock     sync.Mutex
	listenLock  sync.Mutex
	done        chan struct{}
	rootPath    string
	listenNames []string
}

// newZookeeperServiceDiscovery the constructor of newZookeeperServiceDiscovery
func newZookeeperServiceDiscovery(name string) (registry.ServiceDiscovery, error) {
	instance, ok := instanceMap[name]
	if ok {
		return instance, nil
	}

	initLock.Lock()
	defer initLock.Unlock()

	// double check
	instance, ok = instanceMap[name]
	if ok {
		return instance, nil
	}

	sdc, ok := config.GetBaseConfig().GetServiceDiscoveries(name)
	if !ok || len(sdc.RemoteRef) == 0 {
		return nil, perrors.New("could not init the instance because the config is invalid")
	}
	remoteConfig, ok := config.GetBaseConfig().GetRemoteConfig(sdc.RemoteRef)
	if !ok {
		return nil, perrors.New("could not find the remote config for name: " + sdc.RemoteRef)
	}
	rootPath := remoteConfig.GetParam("rootPath", "/services")
	url := common.NewURLWithOptions(
		common.WithParams(make(url.Values)),
		common.WithPassword(remoteConfig.Password),
		common.WithUsername(remoteConfig.Username),
		common.WithParamsValue(constant.REGISTRY_TIMEOUT_KEY, remoteConfig.TimeoutStr))
	url.Location = remoteConfig.Address
	zksd := &zookeeperServiceDiscovery{
		url:      url,
		rootPath: rootPath,
	}
	err := zookeeper.ValidateZookeeperClient(zksd, zookeeper.WithZkName(ServiceDiscoveryZkClient))
	if err != nil {
		return nil, err
	}
	go zookeeper.HandleClientRestart(zksd)
	zksd.csd = curator_discovery.NewServiceDiscovery(zksd.client, rootPath)
	return zksd, nil
}

// nolint
func (zksd *zookeeperServiceDiscovery) ZkClient() *zookeeper.ZookeeperClient {
	return zksd.client
}

// nolint
func (zksd *zookeeperServiceDiscovery) SetZkClient(client *zookeeper.ZookeeperClient) {
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
func (zksd *zookeeperServiceDiscovery) GetUrl() *common.URL {
	return zksd.url
}

// nolint
func (zksd *zookeeperServiceDiscovery) String() string {
	return fmt.Sprintf("zookeeper-service-discovery[%s]", zksd.url)
}

// Close client be closed
func (zksd *zookeeperServiceDiscovery) Destroy() error {
	zksd.client.Close()
	return nil
}

// Register will register service in zookeeper, instance convert to curator's service instance
// which define in curator-x-discovery.
func (zksd *zookeeperServiceDiscovery) Register(instance registry.ServiceInstance) error {
	cris := zksd.toCuratorInstance(instance)
	return zksd.csd.RegisterService(cris)
}

// Register will update service in zookeeper, instance convert to curator's service instance
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
		return make([]registry.ServiceInstance, 0, 0)
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
	return gxpage.New(offset, pageSize, res, len(all))
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
	return gxpage.New(offset, pageSize, res, len(all))
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
func (zksd *zookeeperServiceDiscovery) AddListener(listener *registry.ServiceInstancesChangedListener) error {
	zksd.listenLock.Lock()
	defer zksd.listenLock.Unlock()
	zksd.listenNames = append(zksd.listenNames, listener.ServiceName)
	zksd.csd.ListenServiceEvent(listener.ServiceName, zksd)
	return nil
}

func (zksd *zookeeperServiceDiscovery) DispatchEventByServiceName(serviceName string) error {
	return zksd.DispatchEventForInstances(serviceName, zksd.GetInstances(serviceName))
}

// DispatchEventForInstances dispatch ServiceInstancesChangedEvent
func (zksd *zookeeperServiceDiscovery) DispatchEventForInstances(serviceName string, instances []registry.ServiceInstance) error {
	return zksd.DispatchEvent(registry.NewServiceInstancesChangedEvent(serviceName, instances))
}

// nolint
func (zksd *zookeeperServiceDiscovery) DispatchEvent(event *registry.ServiceInstancesChangedEvent) error {
	extension.GetGlobalDispatcher().Dispatch(event)
	return nil
}

// DataChange implement DataListener's DataChange function
// to resolve event to do DispatchEventByServiceName
func (zksd *zookeeperServiceDiscovery) DataChange(eventType remoting.Event) bool {
	path := strings.TrimPrefix(eventType.Path, zksd.rootPath)
	path = strings.TrimPrefix(path, constant.PATH_SEPARATOR)
	// get service name in zk path
	serviceName := strings.Split(path, constant.PATH_SEPARATOR)[0]
	err := zksd.DispatchEventByServiceName(serviceName)
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
	cuis := &curator_discovery.ServiceInstance{
		Name:                instance.GetServiceName(),
		Id:                  id,
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
		logger.Errorf("[zkServiceDiscovery] toZookeeperInstance{%s} payload is not map[string]interface{}", cris.Id)
		return nil
	}
	mdi, ok := pl["metadata"].(map[string]interface{})
	if !ok {
		logger.Errorf("[zkServiceDiscovery] toZookeeperInstance{%s} metadata is not map[string]interface{}", cris.Id)
		return nil
	}
	md := make(map[string]string, len(mdi))
	for k, v := range mdi {
		md[k] = fmt.Sprint(v)
	}
	return &registry.DefaultServiceInstance{
		Id:          cris.Id,
		ServiceName: cris.Name,
		Host:        cris.Address,
		Port:        cris.Port,
		Enable:      true,
		Healthy:     true,
		Metadata:    md,
	}
}
