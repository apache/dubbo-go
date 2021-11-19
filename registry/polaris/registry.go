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
	"strconv"
	"sync"
	"time"

	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/common/extension"
	"dubbo.apache.org/dubbo-go/v3/common/logger"
	"dubbo.apache.org/dubbo-go/v3/registry"
	"dubbo.apache.org/dubbo-go/v3/remoting/polaris"
	perrors "github.com/pkg/errors"
	"github.com/polarismesh/polaris-go/api"
	"github.com/polarismesh/polaris-go/pkg/model"
)

var localIP = ""

const (
	// RegistryConnDelay registry connection delay
	RegistryConnDelay = 3
)

func init() {
	localIP = common.GetLocalIp()
	extension.SetRegistry(constant.NACOS_KEY, newPolarisRegistry)
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
		registryUrls: make(map[string]*common.URL),
	}

	return pRegistry, nil
}

type polarisRegistry struct {
	url          *common.URL
	consumer     api.ConsumerAPI
	provider     api.ProviderAPI
	lock         *sync.RWMutex
	registryUrls map[string]*common.URL
}

// Register
//  @receiver pr
//  @param url
//  @return error
func (pr *polarisRegistry) Register(url *common.URL) error {
	serviceName := getServiceName(url)
	param := createRegisterParam(url, serviceName)
	resp, err := pr.provider.Register(param)
	if err != nil {
		return err
	}

	pr.lock.Lock()
	defer pr.lock.Unlock()
	url.SetParam(constant.POLARIS_INSTANCE_ID, resp.InstanceID)
	pr.registryUrls[url.Key()] = url
	return nil
}

// UnRegister
//  @receiver pr
//  @param conf
//  @return error
func (pr *polarisRegistry) UnRegister(conf *common.URL) error {
	var (
		ok     bool
		err    error
		oldURL *common.URL
	)

	func() {
		pr.lock.Lock()
		defer pr.lock.Unlock()

		oldURL, ok = pr.registryUrls[conf.Key()]

		if !ok {
			err = perrors.Errorf("Path{%s} has not registered", conf.Key())
		}
		delete(pr.registryUrls, oldURL.Key())
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
			pr.registryUrls[conf.Key()] = oldURL
		}()
		return perrors.WithMessagef(err, "register(conf:%+v)", conf)
	}
	return nil
}

// Subscribe
func (pr *polarisRegistry) Subscribe(url *common.URL, notifyListener registry.NotifyListener) error {
	role, _ := strconv.Atoi(url.GetParam(constant.ROLE_KEY, ""))
	if role != common.CONSUMER {
		return nil
	}

	for {
		listener, err := NewPolarisListener(url, pr.consumer)
		if err != nil {
			logger.Warnf("getListener() = err:%v", perrors.WithStack(err))
			time.Sleep(time.Duration(RegistryConnDelay) * time.Second)
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
		}
	}
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
	for _, url := range pr.registryUrls {
		err := pr.UnRegister(url)
		logger.Infof("DeRegister Polaris URL:%+v", url)
		if err != nil {
			logger.Errorf("Deregister URL:%+v err:%v", url, err.Error())
		}
	}
	return
}

// IsAvailable always return true when use polaris
func (pr *polarisRegistry) IsAvailable() bool {
	return true
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

	metadata := make(map[string]string)
	url.RangeParams(func(key, value string) bool {
		metadata[key] = value
		return true
	})
	metadata[constant.POLARIS_DUBBO_PATH] = url.Path

	return &api.InstanceRegisterRequest{
		InstanceRegisterRequest: model.InstanceRegisterRequest{
			Service:      serviceName,
			ServiceToken: url.GetParam(constant.POLARIS_SERVICE_TOKEN, ""),
			Namespace:    url.GetParam(constant.POLARIS_NAMESPACE, defaultNamespace),
			Host:         url.Ip,
			Port:         port,
			Metadata:     metadata,
		},
	}
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
			Service:      serviceName,
			ServiceToken: url.GetParam(constant.POLARIS_SERVICE_TOKEN, ""),
			Namespace:    url.GetParam(constant.POLARIS_NAMESPACE, defaultNamespace),
			Host:         url.Ip,
			Port:         port,
		},
	}
}
