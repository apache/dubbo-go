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
	"strconv"
	"strings"
	"time"
)

import (
	nacosClient "github.com/dubbogo/gost/database/kv/nacos"
	"github.com/nacos-group/nacos-sdk-go/vo"
	perrors "github.com/pkg/errors"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/common/extension"
	"dubbo.apache.org/dubbo-go/v3/common/logger"
	"dubbo.apache.org/dubbo-go/v3/registry"
	"dubbo.apache.org/dubbo-go/v3/remoting/nacos"
)

var localIP = ""

const (
	// RegistryConnDelay registry connection delay
	RegistryConnDelay = 3
)

func init() {
	localIP = common.GetLocalIp()
	extension.SetRegistry(constant.NACOS_KEY, newNacosRegistry)
}

type nacosRegistry struct {
	*common.URL
	namingClient *nacosClient.NacosNamingClient
	registryUrls []*common.URL
}

func getCategory(url *common.URL) string {
	role, _ := strconv.Atoi(url.GetParam(constant.ROLE_KEY, strconv.Itoa(constant.NACOS_DEFAULT_ROLETYPE)))
	category := common.DubboNodes[role]
	return category
}

func getServiceName(url *common.URL) string {
	var buffer bytes.Buffer

	buffer.Write([]byte(getCategory(url)))
	appendParam(&buffer, url, constant.INTERFACE_KEY)
	appendParam(&buffer, url, constant.VERSION_KEY)
	appendParam(&buffer, url, constant.GROUP_KEY)
	return buffer.String()
}

func appendParam(target *bytes.Buffer, url *common.URL, key string) {
	value := url.GetParam(key, "")
	target.Write([]byte(constant.NACOS_SERVICE_NAME_SEPARATOR))
	if strings.TrimSpace(value) != "" {
		target.Write([]byte(value))
	}
}

func createRegisterParam(url *common.URL, serviceName string) vo.RegisterInstanceParam {
	category := getCategory(url)
	params := make(map[string]string)

	url.RangeParams(func(key, value string) bool {
		params[key] = value
		return true
	})

	params[constant.NACOS_CATEGORY_KEY] = category
	params[constant.NACOS_PROTOCOL_KEY] = url.Protocol
	params[constant.NACOS_PATH_KEY] = url.Path
	if len(url.Ip) == 0 {
		url.Ip = localIP
	}
	if len(url.Port) == 0 || url.Port == "0" {
		url.Port = "80"
	}
	port, _ := strconv.Atoi(url.Port)
	instance := vo.RegisterInstanceParam{
		Ip:          url.Ip,
		Port:        uint64(port),
		Metadata:    params,
		Weight:      1,
		Enable:      true,
		Healthy:     true,
		Ephemeral:   true,
		ServiceName: serviceName,
	}
	return instance
}

// Register will register the service @url to its nacos registry center
func (nr *nacosRegistry) Register(url *common.URL) error {
	serviceName := getServiceName(url)
	param := createRegisterParam(url, serviceName)
	isRegistry, err := nr.namingClient.Client().RegisterInstance(param)
	if err != nil {
		return err
	}
	if !isRegistry {
		return perrors.New("registry [" + serviceName + "] to  nacos failed")
	}
	nr.registryUrls = append(nr.registryUrls, url)
	return nil
}

func createDeregisterParam(url *common.URL, serviceName string) vo.DeregisterInstanceParam {
	if len(url.Ip) == 0 {
		url.Ip = localIP
	}
	if len(url.Port) == 0 || url.Port == "0" {
		url.Port = "80"
	}
	port, _ := strconv.Atoi(url.Port)
	return vo.DeregisterInstanceParam{
		Ip:          url.Ip,
		Port:        uint64(port),
		ServiceName: serviceName,
		Ephemeral:   true,
	}
}

func (nr *nacosRegistry) DeRegister(url *common.URL) error {
	serviceName := getServiceName(url)
	param := createDeregisterParam(url, serviceName)
	isDeRegistry, err := nr.namingClient.Client().DeregisterInstance(param)
	if err != nil {
		return err
	}
	if !isDeRegistry {
		return perrors.New("DeRegistry [" + serviceName + "] to nacos failed")
	}
	return nil
}

// UnRegister
func (nr *nacosRegistry) UnRegister(conf *common.URL) error {
	return perrors.New("UnRegister is not support in nacosRegistry")
}

func (nr *nacosRegistry) subscribe(conf *common.URL) (registry.Listener, error) {
	return NewNacosListener(conf, nr.namingClient)
}

// subscribe from registry
func (nr *nacosRegistry) Subscribe(url *common.URL, notifyListener registry.NotifyListener) error {
	role, _ := strconv.Atoi(nr.URL.GetParam(constant.ROLE_KEY, ""))
	if role != common.CONSUMER {
		return nil
	}

	for {
		if !nr.IsAvailable() {
			logger.Warnf("event listener game over.")
			return perrors.New("nacosRegistry is not available.")
		}

		listener, err := nr.subscribe(url)
		if err != nil {
			if !nr.IsAvailable() {
				logger.Warnf("event listener game over.")
				return err
			}
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

// UnSubscribe :
func (nr *nacosRegistry) UnSubscribe(url *common.URL, notifyListener registry.NotifyListener) error {
	return perrors.New("UnSubscribe not support in nacosRegistry")
}

// GetURL gets its registration URL
func (nr *nacosRegistry) GetURL() *common.URL {
	return nr.URL
}

// IsAvailable determines nacos registry center whether it is available
func (nr *nacosRegistry) IsAvailable() bool {
	// TODO
	return true
}

// nolint
func (nr *nacosRegistry) Destroy() {
	for _, url := range nr.registryUrls {
		err := nr.DeRegister(url)
		logger.Infof("DeRegister Nacos URL:%+v", url)
		if err != nil {
			logger.Errorf("Deregister URL:%+v err:%v", url, err.Error())
		}
	}
	return
}

// newNacosRegistry will create new instance
func newNacosRegistry(url *common.URL) (registry.Registry, error) {
	namingClient, err := nacos.NewNacosClientByUrl(url)
	if err != nil {
		return &nacosRegistry{}, err
	}
	tmpRegistry := &nacosRegistry{
		URL:          url,
		namingClient: namingClient,
		registryUrls: []*common.URL{},
	}
	return tmpRegistry, nil
}
