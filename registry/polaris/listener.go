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

//@Author: chuntaojun <liaochuntao@live.com>
//@Description:
//@Time: 2021/11/19 02:46

package polaris

import (
	"bytes"
	"fmt"
	"net/url"
	"reflect"
	"strconv"
	"sync"

	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/common/logger"
	"dubbo.apache.org/dubbo-go/v3/config_center"
	"dubbo.apache.org/dubbo-go/v3/registry"
	"dubbo.apache.org/dubbo-go/v3/remoting"
	gxchan "github.com/dubbogo/gost/container/chan"
	perrors "github.com/pkg/errors"
	"github.com/polarismesh/polaris-go/api"
	"github.com/polarismesh/polaris-go/pkg/model"
)

type polarisListener struct {
	consumer       api.ConsumerAPI
	listenUrl      *common.URL
	events         *gxchan.UnboundedChan
	instanceMap    map[string]model.Instance
	cacheLock      *sync.RWMutex
	closeCh        chan struct{}
	subscribeParam *api.WatchServiceRequest
}

// NewPolarisListener new polaris listener
func NewPolarisListener(url *common.URL, consumer api.ConsumerAPI) (*polarisListener, error) {
	listener := &polarisListener{
		consumer:    consumer,
		listenUrl:   url,
		events:      gxchan.NewUnboundedChan(32),
		instanceMap: map[string]model.Instance{},
		cacheLock:   &sync.RWMutex{},
		closeCh:     make(chan struct{}),
	}
	return listener, listener.run()
}

func (pl *polarisListener) run() error {
	if pl.consumer == nil {
		return perrors.New("polaris consumer is nil")
	}
	serviceName := getSubscribeName(pl.listenUrl)
	pl.subscribeParam = &api.WatchServiceRequest{
		WatchServiceRequest: model.WatchServiceRequest{
			Key: model.ServiceKey{
				Namespace: pl.listenUrl.GetParam(constant.POLARIS_NAMESPACE, "default"),
				Service:   serviceName,
			},
		},
	}
	go func() {
		resp, err := pl.consumer.WatchService(pl.subscribeParam)
		if err != nil {
		} else {
			pl.handlerWatchResp(resp)
		}
	}()
	return nil
}

// Next returns next service event once received
func (pl *polarisListener) Next() (*registry.ServiceEvent, error) {
	for {
		select {
		case <-pl.closeCh:
			logger.Warnf("polaris listener is close!listenUrl:%+v", pl.listenUrl)
			return nil, perrors.New("listener stopped")

		case val := <-pl.events.Out():
			e, _ := val.(*config_center.ConfigChangeEvent)
			logger.Debugf("got polaris event %s", e)
			return &registry.ServiceEvent{Action: e.ConfigType, Service: e.Value.(*common.URL)}, nil
		}
	}
}

// Close closes this listener
func (pl *polarisListener) Close() {
	// TODO need to add UnWatch in polaris

	close(pl.closeCh)
}

//
func (pl *polarisListener) handlerWatchResp(resp *model.WatchServiceResponse) {

	for {
		select {
		case <-pl.closeCh:
			return
		case _ = <-resp.EventChannel:
			receiveInstances := resp.GetAllInstancesResp.Instances
			addInstances := make([]model.Instance, 0, len(receiveInstances))
			delInstances := make([]model.Instance, 0, len(receiveInstances))
			updateInstances := make([]model.Instance, 0, len(receiveInstances))
			newInstanceMap := make(map[string]model.Instance, len(receiveInstances))

			pl.cacheLock.Lock()
			defer pl.cacheLock.Unlock()
			for i := range receiveInstances {
				instance := receiveInstances[i]

				if !instance.IsIsolated() {
					// instance is isolated,so ignore it
					continue
				}
				host := fmt.Sprintf("%s:%d", instance.GetHost(), instance.GetPort())
				newInstanceMap[host] = instance
				if old, ok := pl.instanceMap[host]; !ok {
					// instance does not exist in cache, add it to cache
					addInstances = append(addInstances, instance)
				} else {
					// instance is not different from cache, update it to cache
					if !reflect.DeepEqual(old, instance) {
						updateInstances = append(updateInstances, instance)
					}
				}
			}

			for host, inst := range pl.instanceMap {
				if _, ok := newInstanceMap[host]; !ok {
					// cache instance does not exist in new instance list, remove it from cache
					delInstances = append(delInstances, inst)
				}
			}

			pl.instanceMap = newInstanceMap

			pl.process(addInstances, remoting.EventTypeAdd)
			pl.process(delInstances, remoting.EventTypeDel)
			pl.process(updateInstances, remoting.EventTypeUpdate)
		}
	}
}

func (pl *polarisListener) process(instances []model.Instance, eventType remoting.EventType) {
	for i := range instances {
		newUrl := generateUrl(instances[i])
		if newUrl != nil {
			pl.events.In() <- &config_center.ConfigChangeEvent{Value: newUrl, ConfigType: eventType}
		}
	}

}

func getSubscribeName(url *common.URL) string {
	var buffer bytes.Buffer
	buffer.Write([]byte(common.DubboNodes[common.PROVIDER]))
	appendParam(&buffer, url, constant.INTERFACE_KEY)
	return buffer.String()
}

func generateUrl(instance model.Instance) *common.URL {
	if instance.GetMetadata() == nil {
		logger.Errorf("polaris instance metadata is empty,instance:%+v", instance)
		return nil
	}
	path := instance.GetMetadata()["path"]
	myInterface := instance.GetMetadata()["interface"]
	if len(path) == 0 && len(myInterface) == 0 {
		logger.Errorf("polaris instance metadata does not have  both path key and interface key,instance:%+v", instance)
		return nil
	}
	if len(path) == 0 && len(myInterface) != 0 {
		path = "/" + myInterface
	}
	protocol := instance.GetProtocol()
	if len(protocol) == 0 {
		logger.Errorf("polaris instance metadata does not have protocol key,instance:%+v", instance)
		return nil
	}
	urlMap := url.Values{}
	for k, v := range instance.GetMetadata() {
		urlMap.Set(k, v)
	}
	return common.NewURLWithOptions(
		common.WithIp(instance.GetHost()),
		common.WithPort(strconv.Itoa(int(instance.GetPort()))),
		common.WithProtocol(protocol),
		common.WithParams(urlMap),
		common.WithPath(path),
	)
}
