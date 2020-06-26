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
	client   *zookeeper.ZookeeperClient
	csd      *curator_discovery.ServiceDiscovery
	listener *zookeeper.ZkEventListener
	url      *common.URL
	wg       sync.WaitGroup
	cltLock  sync.Mutex
	done     chan struct{}
	rootPath string
}

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
	basePath := remoteConfig.GetParam("rootPath", "/services")
	url := common.NewURLWithOptions(
		common.WithLocation(remoteConfig.Address),
		common.WithPassword(remoteConfig.Password),
		common.WithUsername(remoteConfig.Username),
		common.WithParamsValue(constant.REGISTRY_TIMEOUT_KEY, remoteConfig.TimeoutStr))
	zksd := &zookeeperServiceDiscovery{
		url:      url,
		rootPath: basePath,
	}
	err := zookeeper.ValidateZookeeperClient(zksd, zookeeper.WithZkName(ServiceDiscoveryZkClient))
	if err != nil {
		return nil, err
	}
	go zookeeper.HandleClientRestart(zksd)
	zksd.listener = zookeeper.NewZkEventListener(zksd.client)
	zksd.listener.ListenServiceEvent(nil, basePath, zksd)
	zksd.csd = curator_discovery.NewServiceDiscovery(zksd.client, basePath)
	return zksd, nil
}

func (zksd zookeeperServiceDiscovery) ZkClient() *zookeeper.ZookeeperClient {
	return zksd.client
}

func (zksd zookeeperServiceDiscovery) SetZkClient(client *zookeeper.ZookeeperClient) {
	zksd.client = client
}

func (zksd zookeeperServiceDiscovery) ZkClientLock() *sync.Mutex {
	return &zksd.cltLock
}

func (zksd zookeeperServiceDiscovery) WaitGroup() *sync.WaitGroup {
	return &zksd.wg
}

func (zksd zookeeperServiceDiscovery) Done() chan struct{} {
	return zksd.done
}

func (zksd zookeeperServiceDiscovery) RestartCallBack() bool {
	zksd.csd.ReRegisterService()
	zksd.listener.ListenServiceEvent(nil, zksd.rootPath, zksd)
	return true
}

func (zksd zookeeperServiceDiscovery) GetUrl() common.URL {
	return *zksd.url
}

func (zksd zookeeperServiceDiscovery) String() string {
	return fmt.Sprintf("zookeeper-service-discovery[%s]", zksd.url)
}

func (zksd zookeeperServiceDiscovery) Destroy() error {
	zksd.client.Close()
	return nil
}

func (zksd zookeeperServiceDiscovery) Register(instance registry.ServiceInstance) error {
	cris := zksd.toCuratorInstance(instance)
	return zksd.csd.RegisterService(cris)
}

func (zksd zookeeperServiceDiscovery) Update(instance registry.ServiceInstance) error {
	cris := zksd.toCuratorInstance(instance)
	return zksd.csd.UpdateService(cris)
}

func (zksd zookeeperServiceDiscovery) Unregister(instance registry.ServiceInstance) error {
	cris := zksd.toCuratorInstance(instance)
	return zksd.csd.UnregisterService(cris)
}

func (zksd zookeeperServiceDiscovery) GetDefaultPageSize() int {
	return registry.DefaultPageSize
}

func (zksd zookeeperServiceDiscovery) GetServices() *gxset.HashSet {
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

func (zksd zookeeperServiceDiscovery) GetInstances(serviceName string) []registry.ServiceInstance {
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

func (zksd zookeeperServiceDiscovery) GetInstancesByPage(serviceName string, offset int, pageSize int) gxpage.Pager {
	all := zksd.GetInstances(serviceName)
	res := make([]interface{}, 0, pageSize)
	// could not use res = all[a:b] here because the res should be []interface{}, not []ServiceInstance
	for i := offset; i < len(all) && i < offset+pageSize; i++ {
		res = append(res, all[i])
	}
	return gxpage.New(offset, pageSize, res, len(all))
}

func (zksd zookeeperServiceDiscovery) GetHealthyInstancesByPage(serviceName string, offset int, pageSize int, _ bool) gxpage.Pager {
	return zksd.GetInstancesByPage(serviceName, offset, pageSize)
}

func (zksd zookeeperServiceDiscovery) GetRequestInstances(serviceNames []string, offset int, requestedSize int) map[string]gxpage.Pager {
	res := make(map[string]gxpage.Pager, len(serviceNames))
	for _, name := range serviceNames {
		res[name] = zksd.GetInstancesByPage(name, offset, requestedSize)
	}
	return res
}

func (zksd zookeeperServiceDiscovery) AddListener(listener *registry.ServiceInstancesChangedListener) error {
	return nil
}

func (zksd zookeeperServiceDiscovery) DispatchEventByServiceName(serviceName string) error {
	return zksd.DispatchEventForInstances(serviceName, zksd.GetInstances(serviceName))
}

func (zksd zookeeperServiceDiscovery) DispatchEventForInstances(serviceName string, instances []registry.ServiceInstance) error {
	return zksd.DispatchEvent(registry.NewServiceInstancesChangedEvent(serviceName, instances))
}

func (zksd zookeeperServiceDiscovery) DispatchEvent(event *registry.ServiceInstancesChangedEvent) error {
	extension.GetGlobalDispatcher().Dispatch(event)
	return nil
}

func (zksd zookeeperServiceDiscovery) DataChange(eventType remoting.Event) bool {
	path := eventType.Path
	name := strings.Split(path, "/")[1]
	id := strings.Split(path, "/")[2]
	zksd.csd.UpdateInternalService(name, id)
	err := zksd.DispatchEventByServiceName(name)
	if err != nil {
		logger.Errorf("[zkServiceDiscovery] DispatchEventByServiceName{%s} error = err{%v}", name, err)
	}
	return true
}

func (zksd zookeeperServiceDiscovery) toCuratorInstance(instance registry.ServiceInstance) *curator_discovery.ServiceInstance {
	id := instance.GetHost() + ":" + strconv.Itoa(instance.GetPort())
	pl := make(map[string]interface{})
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

func (zksd zookeeperServiceDiscovery) toZookeeperInstance(cris *curator_discovery.ServiceInstance) registry.ServiceInstance {
	pl, ok := cris.Payload.(map[string]interface{})
	if !ok {
		logger.Errorf("[zkServiceDiscovery] toZookeeperInstance{%s} payload is not map", cris.Id)
	}
	md, ok := pl["metadata"].(map[string]string)
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
