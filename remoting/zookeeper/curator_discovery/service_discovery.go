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

package curator_discovery

import (
	"encoding/json"
	"path"
	"strings"
	"sync"
)

import (
	"github.com/dubbogo/go-zookeeper/zk"

	gxzookeeper "github.com/dubbogo/gost/database/kv/zk"
	"github.com/dubbogo/gost/log/logger"

	perrors "github.com/pkg/errors"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/remoting"
	"dubbo.apache.org/dubbo-go/v3/remoting/zookeeper"
)

// Entry contain a service instance
type Entry struct {
	sync.Mutex
	instance *ServiceInstance
}

// ServiceDiscovery which define in curator-x-discovery, please refer to
// https://github.com/apache/curator/blob/master/curator-x-discovery/src/main/java/org/apache/curator/x/discovery/ServiceDiscovery.java
// It's not exactly the same as curator-x-discovery's service discovery
type ServiceDiscovery struct {
	client   *gxzookeeper.ZookeeperClient
	mutex    *sync.Mutex
	basePath string
	services *sync.Map
	listener *zookeeper.ZkEventListener
}

// NewServiceDiscovery the constructor of service discovery
func NewServiceDiscovery(client *gxzookeeper.ZookeeperClient, basePath string) *ServiceDiscovery {
	return &ServiceDiscovery{
		client:   client,
		mutex:    &sync.Mutex{},
		basePath: basePath,
		services: &sync.Map{},
		listener: zookeeper.NewZkEventListener(client),
	}
}

// registerService register service to zookeeper
func (sd *ServiceDiscovery) registerService(instance *ServiceInstance) error {
	path := sd.pathForInstance(instance.Name, instance.ID)
	data, err := json.Marshal(instance)
	if err != nil {
		return err
	}
	err = sd.client.CreateTempWithValue(path, data)
	if err == zk.ErrNodeExists {
		_, state, _ := sd.client.GetContent(path)
		if state != nil {
			_, err = sd.client.SetContent(path, data, state.Version+1)
			if err != nil {
				logger.Debugf("Try to update the node data failed. In most cases, it's not a problem. ")
			}
		}
		return nil
	}
	if err != nil {
		return err
	}
	return nil
}

// RegisterService register service to zookeeper, and ensure cache is consistent with zookeeper
func (sd *ServiceDiscovery) RegisterService(instance *ServiceInstance) error {
	value, loaded := sd.services.LoadOrStore(instance.ID, &Entry{})
	entry, ok := value.(*Entry)
	if !ok {
		return perrors.New("[ServiceDiscovery] services value not entry")
	}
	entry.Lock()
	defer entry.Unlock()
	entry.instance = instance
	err := sd.registerService(instance)
	if err != nil {
		return err
	}
	if !loaded {
		sd.ListenServiceInstanceEvent(instance.Name, instance.ID, sd)
	}
	return nil
}

// UpdateService update service in zookeeper, and ensure cache is consistent with zookeeper
func (sd *ServiceDiscovery) UpdateService(instance *ServiceInstance) error {
	value, ok := sd.services.Load(instance.ID)
	if !ok {
		return perrors.Errorf("[ServiceDiscovery] Service{%s} not registered", instance.ID)
	}
	entry, ok := value.(*Entry)
	if !ok {
		return perrors.New("[ServiceDiscovery] services value not entry")
	}
	data, err := json.Marshal(instance)
	if err != nil {
		return err
	}

	entry.Lock()
	defer entry.Unlock()
	entry.instance = instance
	path := sd.pathForInstance(instance.Name, instance.ID)

	_, err = sd.client.SetContent(path, data, -1)
	if err != nil {
		return err
	}
	return nil
}

// updateInternalService update service in cache
func (sd *ServiceDiscovery) updateInternalService(name, id string) {
	value, ok := sd.services.Load(id)
	if !ok {
		return
	}
	entry, ok := value.(*Entry)
	if !ok {
		return
	}
	entry.Lock()
	defer entry.Unlock()
	instance, err := sd.QueryForInstance(name, id)
	if err != nil {
		logger.Infof("[zkServiceDiscovery] UpdateInternalService{%s} error = err{%v}", id, err)
		return
	}
	entry.instance = instance
}

// UnregisterService un-register service in zookeeper and delete service in cache
func (sd *ServiceDiscovery) UnregisterService(instance *ServiceInstance) error {
	_, ok := sd.services.Load(instance.ID)
	if !ok {
		return nil
	}
	sd.services.Delete(instance.ID)
	return sd.unregisterService(instance)
}

// unregisterService un-register service in zookeeper
func (sd *ServiceDiscovery) unregisterService(instance *ServiceInstance) error {
	path := sd.pathForInstance(instance.Name, instance.ID)
	return sd.client.Delete(path)
}

// ReRegisterServices re-register all cache services to zookeeper
func (sd *ServiceDiscovery) ReRegisterServices() {
	sd.services.Range(func(key, value interface{}) bool {
		entry, ok := value.(*Entry)
		if !ok {
			return true
		}
		entry.Lock()
		defer entry.Unlock()
		instance := entry.instance
		err := sd.registerService(instance)
		if err != nil {
			logger.Errorf("[zkServiceDiscovery] registerService{%s} error = err{%v}", instance.ID, perrors.WithStack(err))
			return true
		}
		sd.ListenServiceInstanceEvent(instance.Name, instance.ID, sd)
		return true
	})
}

// QueryForInstances query instances in zookeeper by name
func (sd *ServiceDiscovery) QueryForInstances(name string) ([]*ServiceInstance, error) {
	ids, err := sd.client.GetChildren(sd.pathForName(name))
	if err != nil {
		return nil, err
	}
	var (
		instance  *ServiceInstance
		instances []*ServiceInstance
	)
	for _, id := range ids {
		instance, err = sd.QueryForInstance(name, id)
		if err != nil {
			return nil, err
		}
		instances = append(instances, instance)
	}
	return instances, nil
}

// QueryForInstance query instances in zookeeper by name and id
func (sd *ServiceDiscovery) QueryForInstance(name string, id string) (*ServiceInstance, error) {
	path := sd.pathForInstance(name, id)
	data, _, err := sd.client.GetContent(path)
	if err != nil {
		return nil, err
	}
	instance := &ServiceInstance{}
	err = json.Unmarshal(data, instance)
	if err != nil {
		return nil, err
	}
	return instance, nil
}

// QueryForNames query all service name in zookeeper
func (sd *ServiceDiscovery) QueryForNames() ([]string, error) {
	return sd.client.GetChildren(sd.basePath)
}

// ListenServiceEvent add a listener in a service
func (sd *ServiceDiscovery) ListenServiceEvent(name string, listener remoting.DataListener) {
	sd.listener.ListenServiceEvent(nil, sd.pathForName(name), listener)
}

// ListenServiceInstanceEvent add a listener in a instance
func (sd *ServiceDiscovery) ListenServiceInstanceEvent(name, id string, listener remoting.DataListener) {
	sd.listener.ListenServiceNodeEvent(sd.pathForInstance(name, id), listener)
}

// DataChange implement DataListener's DataChange function
func (sd *ServiceDiscovery) DataChange(eventType remoting.Event) bool {
	path := eventType.Path
	name, id, err := sd.getNameAndID(path)
	if err != nil {
		logger.Errorf("[ServiceDiscovery] data change error = {%v}", err)
		return true
	}
	sd.updateInternalService(name, id)
	return true
}

// getNameAndID get service name and instance id by path
func (sd *ServiceDiscovery) getNameAndID(path string) (string, string, error) {
	path = strings.TrimPrefix(path, sd.basePath)
	path = strings.TrimPrefix(path, constant.PathSeparator)
	pathSlice := strings.Split(path, constant.PathSeparator)
	if len(pathSlice) < 2 {
		return "", "", perrors.Errorf("[ServiceDiscovery] path{%s} dont contain name and id", path)
	}
	name := pathSlice[0]
	id := pathSlice[1]
	return name, id, nil
}

// nolint
func (sd *ServiceDiscovery) pathForInstance(name, id string) string {
	return path.Join(sd.basePath, name, id)
}

// nolint
func (sd *ServiceDiscovery) pathForName(name string) string {
	return path.Join(sd.basePath, name)
}

func (sd *ServiceDiscovery) Close() {
	if sd.listener != nil {
		sd.listener.Close()
	}
	if sd.client != nil {
		sd.client.Close()
	}
}
