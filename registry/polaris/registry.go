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
	"github.com/dubbogo/gost/log/logger"

	perrors "github.com/pkg/errors"

	"github.com/polarismesh/polaris-go/api"
	"github.com/polarismesh/polaris-go/pkg/model"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/common/extension"
	"dubbo.apache.org/dubbo-go/v3/registry"
	"dubbo.apache.org/dubbo-go/v3/remoting/polaris"
)

const (
	RegistryConnDelay           = 3
	defaultHeartbeatIntervalSec = 5
)

func init() {
	extension.SetRegistry(constant.PolarisKey, newPolarisRegistry)
}

// newPolarisRegistry will create new instance
func newPolarisRegistry(url *common.URL) (registry.Registry, error) {
	sdkCtx, ns, err := polaris.GetPolarisConfig(url)
	if err != nil {
		return &polarisRegistry{}, err
	}
	pRegistry := &polarisRegistry{
		namespace:    ns,
		provider:     api.NewProviderAPIByContext(sdkCtx),
		consumer:     api.NewConsumerAPIByContext(sdkCtx),
		lock:         &sync.RWMutex{},
		registryUrls: make(map[string]*PolarisHeartbeat),
		listenerLock: &sync.RWMutex{},
	}

	return pRegistry, nil
}

type polarisRegistry struct {
	consumer     api.ConsumerAPI
	namespace    string
	url          *common.URL
	provider     api.ProviderAPI
	lock         *sync.RWMutex
	registryUrls map[string]*PolarisHeartbeat
	watchers     map[string]*PolarisServiceWatcher
	listenerLock *sync.RWMutex
}

// Register will register the service @url to its polaris registry center.
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

// UnRegister returns nil if unregister successfully. If not, returns an error.
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

// Subscribe returns nil if subscribing registry successfully. If not returns an error.
func (pr *polarisRegistry) Subscribe(url *common.URL, notifyListener registry.NotifyListener) error {

	role, _ := strconv.Atoi(url.GetParam(constant.RegistryRoleKey, ""))
	if role != common.CONSUMER {
		return nil
	}
	timer := time.NewTimer(time.Duration(RegistryConnDelay) * time.Second)
	defer timer.Stop()

	req := api.WatchServiceRequest{
		WatchServiceRequest: model.WatchServiceRequest{
			Key: model.ServiceKey{
				Service:   common.GetSubscribeName(url),
				Namespace: pr.namespace,
			},
		},
	}

	for {
		watcher, err := newPolarisWatcher(&req, pr.consumer)

		if err != nil {
			logger.Warnf("getwatcher() = err:%v", perrors.WithStack(err))
			<-timer.C
			timer.Reset(time.Duration(RegistryConnDelay) * time.Second)
			continue
		}
		listener, err := NewPolarisListener(watcher)

		if err != nil {
			logger.Warnf("getListener() = err:%v", perrors.WithStack(err))
			<-timer.C
			timer.Reset(time.Duration(RegistryConnDelay) * time.Second)
			continue
		}

		if err != nil {
			logger.Warnf("getwatcher() = err:%v", perrors.WithStack(err))
			timer := time.NewTimer(time.Duration(RegistryConnDelay) * time.Second)
			timer.Reset(time.Duration(RegistryConnDelay) * time.Second)
			continue
		}

		for {
			serviceEvent, err := listener.Next()

			if err != nil {
				logger.Warnf("Selector.watch() = error{%v}", perrors.WithStack(err))
				listener.Close()
				return err
			}
			logger.Infof("update begin, service event: %v", serviceEvent.String())
			notifyListener.Notify(serviceEvent)
			watcher.startWatch()
		}
	}
}

// UnSubscribe returns nil if unsubscribing registry successfully. If not returns an error.
func (pr *polarisRegistry) UnSubscribe(url *common.URL, notifyListener registry.NotifyListener) error {
	// TODO wait polaris support it
	return perrors.New("UnSubscribe not support in polarisRegistry")
}

// GetURL returns polaris registry's url.
func (pr *polarisRegistry) GetURL() *common.URL {
	return pr.url
}

func (pr *polarisRegistry) createPolarisWatcher(serviceName string) (*PolarisServiceWatcher, error) {

	pr.listenerLock.Lock()
	defer pr.listenerLock.Unlock()

	if _, exist := pr.watchers[serviceName]; !exist {
		subscribeParam := &api.WatchServiceRequest{
			WatchServiceRequest: model.WatchServiceRequest{
				Key: model.ServiceKey{
					Namespace: pr.namespace,
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

// Destroy stop polaris registry.
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
//
//	needed to implement it
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
	common.HandleRegisterIPAndPort(url)
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
	common.HandleRegisterIPAndPort(url)
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
