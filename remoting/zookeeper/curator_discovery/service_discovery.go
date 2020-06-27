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
	"strings"
	"sync"
)

import (
	perrors "github.com/pkg/errors"
)

import (
	"github.com/apache/dubbo-go/common/constant"
	"github.com/apache/dubbo-go/common/logger"
	"github.com/apache/dubbo-go/remoting"
	"github.com/apache/dubbo-go/remoting/zookeeper"
)

type ServiceDiscovery struct {
	client   *zookeeper.ZookeeperClient
	mutex    *sync.Mutex
	basePath string
	services *sync.Map
	listener *zookeeper.ZkEventListener
}

func NewServiceDiscovery(client *zookeeper.ZookeeperClient, basePath string) *ServiceDiscovery {
	return &ServiceDiscovery{
		client:   client,
		mutex:    &sync.Mutex{},
		basePath: basePath,
		services: &sync.Map{},
		listener: zookeeper.NewZkEventListener(client),
	}
}

func (sd *ServiceDiscovery) registerService(instance *ServiceInstance) error {
	path := sd.pathForInstance(instance.Name, instance.Id)
	data, err := json.Marshal(instance)
	if err != nil {
		return err
	}
	err = sd.client.CreateWithValue(path, data)
	if err != nil {
		return err
	}
	return nil
}

func (sd *ServiceDiscovery) RegisterService(instance *ServiceInstance) error {
	_, loaded := sd.services.LoadOrStore(instance.Id, instance)
	err := sd.registerService(instance)
	if err != nil {
		return err
	}
	if !loaded {
		sd.ListenServiceInstanceEvent(instance.Name, instance.Id, sd)
	}
	return nil
}

func (sd *ServiceDiscovery) UpdateService(instance *ServiceInstance) error {
	sd.services.Store(instance.Id, instance)
	path := sd.pathForInstance(instance.Name, instance.Id)
	data, err := json.Marshal(instance)
	if err != nil {
		return err
	}
	_, err = sd.client.SetContent(path, data, -1)
	if err != nil {
		return err
	}
	return nil
}

func (sd *ServiceDiscovery) updateInternalService(name, id string) {
	_, ok := sd.services.Load(id)
	if !ok {
		return
	}
	instance, err := sd.QueryForInstance(name, id)
	if err != nil {
		logger.Infof("[zkServiceDiscovery] UpdateInternalService{%s} error = err{%v}", id, err)
		return
	}
	sd.services.Store(instance.Id, instance)
	return
}

func (sd *ServiceDiscovery) UnregisterService(instance *ServiceInstance) error {
	sd.services.Delete(instance.Id)
	return sd.unregisterService(instance)
}

func (sd *ServiceDiscovery) unregisterService(instance *ServiceInstance) error {
	path := sd.pathForInstance(instance.Name, instance.Id)
	return sd.client.Delete(path)
}

func (sd *ServiceDiscovery) ReRegisterService() {
	sd.services.Range(func(key, value interface{}) bool {
		instance, ok := value.(*ServiceInstance)
		if !ok {

		}
		err := sd.registerService(instance)
		if err != nil {
			logger.Errorf("[zkServiceDiscovery] registerService{%s} error = err{%v}", instance.Id, perrors.WithStack(err))
		}
		sd.ListenServiceInstanceEvent(instance.Name, instance.Id, sd)
		return true
	})
}

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

func (sd *ServiceDiscovery) QueryForNames() ([]string, error) {
	return sd.client.GetChildren(sd.basePath)
}

func (sd *ServiceDiscovery) ListenServiceEvent(name string, listener remoting.DataListener) {
	sd.listener.ListenServiceEvent(nil, sd.pathForName(name), listener)
}

func (sd *ServiceDiscovery) ListenServiceInstanceEvent(name, id string, listener remoting.DataListener) {
	sd.listener.ListenServiceEvent(nil, sd.pathForInstance(name, id), listener)
}

func (sd *ServiceDiscovery) DataChange(eventType remoting.Event) bool {
	path := eventType.Path
	name, id := sd.getNameAndId(path)
	sd.updateInternalService(name, id)
	return true
}

func (sd *ServiceDiscovery) getNameAndId(path string) (string, string) {
	pathSlice := strings.Split(path, constant.PATH_SEPARATOR)
	name := pathSlice[2]
	id := pathSlice[3]
	return name, id
}

func (sd *ServiceDiscovery) pathForInstance(name, id string) string {
	return sd.basePath + constant.PATH_SEPARATOR + name + constant.PATH_SEPARATOR + id
}

func (sd *ServiceDiscovery) pathForName(name string) string {
	return sd.basePath + constant.PATH_SEPARATOR + name
}
