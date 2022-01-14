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

package polaris

import (
	"context"
	"strconv"
	"sync"
	"time"
)

import (
	perrors "github.com/pkg/errors"

	"github.com/polarismesh/polaris-go/api"
	"github.com/polarismesh/polaris-go/pkg/model"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/common/extension"
	"dubbo.apache.org/dubbo-go/v3/common/logger"
	"dubbo.apache.org/dubbo-go/v3/config_center"
	"dubbo.apache.org/dubbo-go/v3/registry"
	"dubbo.apache.org/dubbo-go/v3/remoting"
	"dubbo.apache.org/dubbo-go/v3/remoting/polaris"
)

var localIP = ""

const (
	// RegistryConnDelay registry connection delay
	RegistryConnDelay           = 3
	defaultHeartbeatIntervalSec = 5
)

func init() {
	localIP = common.GetLocalIp()
	extension.SetRegistry(constant.PolarisKey, newPolarisRegistry)
}

// newPolarisRegistry will create new instance
func newPolarisRegistry(url *common.URL) (registry.Registry, error) {
	sdkCtx, _, err := polaris.GetPolarisConfig(url)
	if err != nil {
		return &polarisRegistry{}, err
	}
	pRegistry := &polarisRegistry{
		consumer:     api.NewConsumerAPIByContext(sdkCtx),
		provider:     api.NewProviderAPIByContext(sdkCtx),
		lock:         &sync.RWMutex{},
		registryUrls: make(map[string]*PolarisHeartbeat),
		listenerLock: &sync.RWMutex{},
		watchers:     make(map[string]*PolarisServiceWatcher),
	}

	return pRegistry, nil
}

type polarisRegistry struct {
	url          *common.URL
	consumer     api.ConsumerAPI
	provider     api.ProviderAPI
	lock         *sync.RWMutex
	registryUrls map[string]*PolarisHeartbeat

	listenerLock *sync.RWMutex
	watchers     map[string]*PolarisServiceWatcher
}

// Register
func (pr *polarisRegistry) Register(url *common.URL) error {
	serviceName := getServiceName(url)
	param := createRegisterParam(url, serviceName)
	resp, err := pr.provider.Register(param)
	if err != nil {
		return err
	}

	if resp.Existed {
		logger.Warnf("instance already regist, namespace:%+v, service:%+v, host:%+v, port:%+v",
			param.Namespace, param.Service, param.Host, param.Port)
	}

	pr.lock.Lock()
	defer pr.lock.Unlock()

	url.SetParam(constant.PolarisInstanceID, resp.InstanceID)

	ctx, cancel := context.WithCancel(context.Background())
	go pr.doHeartbeat(ctx, param)

	pr.registryUrls[url.Key()] = &PolarisHeartbeat{
		url:    url,
		cancel: cancel,
	}
	return nil
}

// UnRegister
func (pr *polarisRegistry) UnRegister(conf *common.URL) error {
	var (
		ok     bool
		err    error
		oldVal *PolarisHeartbeat
	)

	func() {
		pr.lock.Lock()
		defer pr.lock.Unlock()

		oldVal, ok = pr.registryUrls[conf.Key()]

		if !ok {
			err = perrors.Errorf("Path{%s} has not registered", conf.Key())
			return
		}

		oldVal.cancel()
		delete(pr.registryUrls, oldVal.url.Key())
	}()

	if err != nil {
		return err
	}

	request := createDeregisterParam(conf, getServiceName(conf))

	err = pr.provider.Deregister(request)
	if err != nil {
		func() {
			pr.lock.Lock()
			defer pr.lock.Unlock()
			pr.registryUrls[conf.Key()] = oldVal
		}()
		return perrors.WithMessagef(err, "register(conf:%+v)", conf)
	}
	return nil
}

// Subscribe
func (pr *polarisRegistry) Subscribe(url *common.URL, notifyListener registry.NotifyListener) error {
	role, _ := strconv.Atoi(url.GetParam(constant.RegistryRoleKey, ""))
	if role != common.CONSUMER {
		return nil
	}

	watcher, err := pr.createPolarisWatcherIfAbsent(url)

	if err != nil {
		return err
	}

	for {
		listener, err := NewPolarisListener(url)
		if err != nil {
			logger.Warnf("getListener() = err:%v", perrors.WithStack(err))
			<-time.After(time.Duration(RegistryConnDelay) * time.Second)
			continue
		}

		watcher.AddSubscriber(func(et remoting.EventType, instances []model.Instance) {
			for i := range instances {
				instance := instances[i]
				listener.events.In() <- &config_center.ConfigChangeEvent{ConfigType: et, Value: instance}
			}
		})

		for {
			serviceEvent, err := listener.Next()
			if err != nil {
				logger.Warnf("Selector.watch() = error{%v}", perrors.WithStack(err))
				listener.Close()
				return err
			}
			logger.Infof("update begin, service event: %v", serviceEvent.String())
			notifyListener.Notify(serviceEvent)
		}
	}
}

func (pr *polarisRegistry) createPolarisWatcherIfAbsent(url *common.URL) (*PolarisServiceWatcher, error) {

	pr.listenerLock.Lock()
	defer pr.listenerLock.Unlock()

	serviceName := getSubscribeName(url)

	if _, exist := pr.watchers[serviceName]; !exist {
		subscribeParam := &api.WatchServiceRequest{
			WatchServiceRequest: model.WatchServiceRequest{
				Key: model.ServiceKey{
					Namespace: url.GetParam(constant.PolarisNamespace, constant.PolarisDefaultNamespace),
					Service:   serviceName,
				},
			},
		}

		watcher, err := newPolarisWatcher(subscribeParam, pr.consumer)
		if err != nil {
			return nil, err
		}
		pr.watchers[serviceName] = watcher
	}

	return pr.watchers[serviceName], nil
}

// UnSubscribe
func (pr *polarisRegistry) UnSubscribe(url *common.URL, notifyListener registry.NotifyListener) error {
	// TODO wait polaris support it
	return perrors.New("UnSubscribe not support in polarisRegistry")
}

// GetURL
func (pr *polarisRegistry) GetURL() *common.URL {
	return pr.url
}

// Destroy
func (pr *polarisRegistry) Destroy() {
	for _, val := range pr.registryUrls {
		val.cancel()
		err := pr.UnRegister(val.url)
		logger.Infof("DeRegister Polaris URL:%+v", val.url)
		if err != nil {
			logger.Errorf("Deregister URL:%+v err:%v", val.url, err.Error())
		}
	}
	return
}

// IsAvailable always return true when use polaris
func (pr *polarisRegistry) IsAvailable() bool {
	return true
}

// doHeartbeat Since polaris does not support automatic reporting of instance heartbeats, separate logic is
//  needed to implement it
func (pr *polarisRegistry) doHeartbeat(ctx context.Context, ins *api.InstanceRegisterRequest) {
	ticker := time.NewTicker(time.Duration(4) * time.Second)

	heartbeat := &api.InstanceHeartbeatRequest{
		InstanceHeartbeatRequest: model.InstanceHeartbeatRequest{
			Service:   ins.Service,
			Namespace: ins.Namespace,
			Host:      ins.Host,
			Port:      ins.Port,
		},
	}

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			pr.provider.Heartbeat(heartbeat)
		}
	}
}

// createRegisterParam convert dubbo url to polaris instance register request
func createRegisterParam(url *common.URL, serviceName string) *api.InstanceRegisterRequest {
	if len(url.Ip) == 0 {
		url.Ip = localIP
	}
	if len(url.Port) == 0 || url.Port == "0" {
		url.Port = "80"
	}
	port, _ := strconv.Atoi(url.Port)

	metadata := make(map[string]string, len(url.GetParams()))
	url.RangeParams(func(key, value string) bool {
		metadata[key] = value
		return true
	})
	metadata[constant.PolarisDubboPath] = url.Path

	req := &api.InstanceRegisterRequest{
		InstanceRegisterRequest: model.InstanceRegisterRequest{
			Service:   serviceName,
			Namespace: url.GetParam(constant.PolarisNamespace, constant.PolarisDefaultNamespace),
			Host:      url.Ip,
			Port:      port,
			Protocol:  &protocolForDubboGO,
			Metadata:  metadata,
		},
	}

	req.SetTTL(defaultHeartbeatIntervalSec)

	return req
}

// createDeregisterParam convert dubbo url to polaris instance deregister request
func createDeregisterParam(url *common.URL, serviceName string) *api.InstanceDeRegisterRequest {
	if len(url.Ip) == 0 {
		url.Ip = localIP
	}
	if len(url.Port) == 0 || url.Port == "0" {
		url.Port = "80"
	}
	port, _ := strconv.Atoi(url.Port)
	return &api.InstanceDeRegisterRequest{
		InstanceDeRegisterRequest: model.InstanceDeRegisterRequest{
			Service:   serviceName,
			Namespace: url.GetParam(constant.PolarisNamespace, constant.PolarisDefaultNamespace),
			Host:      url.Ip,
			Port:      port,
		},
	}
}
