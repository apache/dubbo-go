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
	"bytes"
	"net/url"
	"reflect"
	"strconv"
	"sync"
)

import (
	gxchan "github.com/dubbogo/gost/container/chan"
	nacosClient "github.com/dubbogo/gost/database/kv/nacos"
	"github.com/dubbogo/gost/log/logger"

	"github.com/nacos-group/nacos-sdk-go/model"
	"github.com/nacos-group/nacos-sdk-go/vo"

	perrors "github.com/pkg/errors"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/config_center"
	"dubbo.apache.org/dubbo-go/v3/registry"
	"dubbo.apache.org/dubbo-go/v3/remoting"
)

var (
	listenerCache sync.Map
)

type callback func(services []model.SubscribeService, err error)

type nacosListener struct {
	namingClient   *nacosClient.NacosNamingClient
	listenURL      *common.URL
	regURL         *common.URL
	events         *gxchan.UnboundedChan
	instanceMap    map[string]model.Instance
	cacheLock      sync.Mutex
	done           chan struct{}
	subscribeParam *vo.SubscribeParam
}

// NewNacosListener creates a data listener for nacos
func NewNacosListener(url, regURL *common.URL, namingClient *nacosClient.NacosNamingClient) (*nacosListener, error) {
	listener := &nacosListener{
		namingClient: namingClient,
		listenURL:    url,
		regURL:       regURL,
		events:       gxchan.NewUnboundedChan(32),
		instanceMap:  map[string]model.Instance{},
		done:         make(chan struct{}),
	}
	err := listener.startListen()
	return listener, err
}

func generateInstance(ss model.SubscribeService) model.Instance {
	return model.Instance{
		InstanceId:  ss.InstanceId,
		Ip:          ss.Ip,
		Port:        ss.Port,
		ServiceName: ss.ServiceName,
		Valid:       ss.Valid,
		Enable:      ss.Enable,
		Weight:      ss.Weight,
		Metadata:    ss.Metadata,
		ClusterName: ss.ClusterName,
		Healthy:     ss.Healthy,
	}
}

func generateUrl(instance model.Instance) *common.URL {
	if instance.Metadata == nil {
		logger.Errorf("nacos instance metadata is empty,instance:%+v", instance)
		return nil
	}
	path := instance.Metadata["path"]
	myInterface := instance.Metadata["interface"]
	if len(path) == 0 && len(myInterface) == 0 {
		logger.Errorf("nacos instance metadata does not have  both path key and interface key,instance:%+v", instance)
		return nil
	}
	if len(path) == 0 && len(myInterface) != 0 {
		path = "/" + myInterface
	}
	protocol := instance.Metadata["protocol"]
	if len(protocol) == 0 {
		logger.Errorf("nacos instance metadata does not have protocol key,instance:%+v", instance)
		return nil
	}
	urlMap := url.Values{}
	for k, v := range instance.Metadata {
		urlMap.Set(k, v)
	}
	return common.NewURLWithOptions(
		common.WithIp(instance.Ip),
		common.WithPort(strconv.Itoa(int(instance.Port))),
		common.WithProtocol(protocol),
		common.WithParams(urlMap),
		common.WithPath(path),
	)
}

// Callback will be invoked when got subscribed events.
func (nl *nacosListener) Callback(services []model.SubscribeService, err error) {
	if err != nil {
		logger.Errorf("nacos subscribe callback error:%s , subscribe:%+v ", err.Error(), nl.subscribeParam)
		return
	}

	addInstances := make([]model.Instance, 0, len(services))
	delInstances := make([]model.Instance, 0, len(services))
	updateInstances := make([]model.Instance, 0, len(services))
	newInstanceMap := make(map[string]model.Instance, len(services))

	nl.cacheLock.Lock()
	defer nl.cacheLock.Unlock()
	for i := range services {
		if !services[i].Enable {
			// instance is not available,so ignore it
			continue
		}
		host := services[i].Ip + ":" + strconv.Itoa(int(services[i].Port))
		instance := generateInstance(services[i])
		newInstanceMap[host] = instance
		if old, ok := nl.instanceMap[host]; !ok && instance.Healthy {
			// instance does not exist in cache, add it to cache
			addInstances = append(addInstances, instance)
		} else if !reflect.DeepEqual(old, instance) && instance.Healthy {
			// instance is not different from cache, update it to cache
			updateInstances = append(updateInstances, instance)
		}
	}

	for host, inst := range nl.instanceMap {
		if newInstance, ok := newInstanceMap[host]; !ok || !newInstance.Healthy {
			// cache instance does not exist in new instance list, remove it from cache
			delInstances = append(delInstances, inst)
		}
	}

	nl.instanceMap = newInstanceMap
	for i := range addInstances {
		if newUrl := generateUrl(addInstances[i]); newUrl != nil {
			nl.process(&config_center.ConfigChangeEvent{Value: newUrl, ConfigType: remoting.EventTypeAdd})
		}
	}
	for i := range delInstances {
		if newUrl := generateUrl(delInstances[i]); newUrl != nil {
			nl.process(&config_center.ConfigChangeEvent{Value: newUrl, ConfigType: remoting.EventTypeDel})
		}
	}

	for i := range updateInstances {
		if newUrl := generateUrl(updateInstances[i]); newUrl != nil {
			nl.process(&config_center.ConfigChangeEvent{Value: newUrl, ConfigType: remoting.EventTypeUpdate})
		}
	}
}

func getSubscribeName(url *common.URL) string {
	var buffer bytes.Buffer

	buffer.Write([]byte(common.DubboNodes[common.PROVIDER]))
	appendParam(&buffer, url, constant.InterfaceKey)
	appendParam(&buffer, url, constant.VersionKey)
	appendParam(&buffer, url, constant.GroupKey)
	return buffer.String()
}

func (nl *nacosListener) startListen() error {
	if nl.namingClient == nil {
		return perrors.New("nacos naming namingClient stopped")
	}
	nl.subscribeParam = createSubscribeParam(nl.listenURL, nl.regURL, nl.Callback)
	if nl.subscribeParam == nil {
		return perrors.New("create nacos subscribeParam failed")
	}
	go func() {
		err := nl.namingClient.Client().Subscribe(nl.subscribeParam)
		if err == nil {
			listenerCache.Store(nl.subscribeParam.ServiceName+nl.subscribeParam.GroupName, nl)
		}
	}()
	return nil
}

func (nl *nacosListener) stopListen() error {
	return nl.namingClient.Client().Unsubscribe(nl.subscribeParam)
}

func (nl *nacosListener) process(configType *config_center.ConfigChangeEvent) {
	nl.events.In() <- configType
}

// Next returns the service event from nacos.
func (nl *nacosListener) Next() (*registry.ServiceEvent, error) {
	for {
		select {
		case <-nl.done:
			logger.Warnf("nacos listener is close!listenUrl:%+v", nl.listenURL)
			return nil, perrors.New("listener stopped")

		case val := <-nl.events.Out():
			e, _ := val.(*config_center.ConfigChangeEvent)
			logger.Debugf("got nacos event %s", e)
			return &registry.ServiceEvent{Action: e.ConfigType, Service: e.Value.(*common.URL)}, nil
		}
	}
}

// nolint
func (nl *nacosListener) Close() {
	_ = nl.stopListen()
	close(nl.done)
}
