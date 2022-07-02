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
	"github.com/dubbogo/gost/log/logger"

	"github.com/nacos-group/nacos-sdk-go/vo"

	perrors "github.com/pkg/errors"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/common/extension"
	"dubbo.apache.org/dubbo-go/v3/registry"
	"dubbo.apache.org/dubbo-go/v3/remoting/nacos"
)

var localIP = ""

const (
	RegistryConnDelay = 3
)

func init() {
	localIP = common.GetLocalIp()
	extension.SetRegistry(constant.NacosKey, newNacosRegistry)
}

type nacosRegistry struct {
	*common.URL
	namingClient *nacosClient.NacosNamingClient
	registryUrls []*common.URL
}

func getCategory(url *common.URL) string {
	role, _ := strconv.Atoi(url.GetParam(constant.RegistryRoleKey, strconv.Itoa(constant.NacosDefaultRoleType)))
	category := common.DubboNodes[role]
	return category
}

func getServiceName(url *common.URL) string {
	var buffer bytes.Buffer

	buffer.Write([]byte(getCategory(url)))
	appendParam(&buffer, url, constant.InterfaceKey)
	appendParam(&buffer, url, constant.VersionKey)
	appendParam(&buffer, url, constant.GroupKey)
	return buffer.String()
}

func appendParam(target *bytes.Buffer, url *common.URL, key string) {
	value := url.GetParam(key, "")
	target.Write([]byte(constant.NacosServiceNameSeparator))
	if strings.TrimSpace(value) != "" {
		target.Write([]byte(value))
	}
}

func createRegisterParam(url *common.URL, serviceName string, groupName string) vo.RegisterInstanceParam {
	category := getCategory(url)
	params := make(map[string]string)

	url.RangeParams(func(key, value string) bool {
		params[key] = value
		return true
	})

	params[constant.NacosCategoryKey] = category
	params[constant.NacosProtocolKey] = url.Protocol
	params[constant.NacosPathKey] = url.Path
	params[constant.MethodsKey] = strings.Join(url.Methods, ",")
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
		GroupName:   groupName,
	}
	return instance
}

// Register will register the service @url to its nacos registry center.
func (nr *nacosRegistry) Register(url *common.URL) error {
	serviceName := getServiceName(url)
	groupName := nr.URL.GetParam(constant.NacosGroupKey, defaultGroup)
	param := createRegisterParam(url, serviceName, groupName)
	logger.Infof("[Nacos Registry] Registry instance with param = %+v", param)
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

func createDeregisterParam(url *common.URL, serviceName string, groupName string) vo.DeregisterInstanceParam {
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
		GroupName:   groupName,
		Ephemeral:   true,
	}
}

// UnRegister returns nil if unregister successfully. If not, returns an error.
func (nr *nacosRegistry) UnRegister(url *common.URL) error {
	serviceName := getServiceName(url)
	groupName := nr.URL.GetParam(constant.NacosGroupKey, defaultGroup)
	param := createDeregisterParam(url, serviceName, groupName)
	isDeRegistry, err := nr.namingClient.Client().DeregisterInstance(param)
	if err != nil {
		return err
	}
	if !isDeRegistry {
		return perrors.New("DeRegistry [" + serviceName + "] to nacos failed")
	}
	return nil
}

func (nr *nacosRegistry) subscribe(conf *common.URL) (registry.Listener, error) {
	return NewNacosListener(conf, nr.URL, nr.namingClient)
}

// Subscribe returns nil if subscribing registry successfully. If not returns an error.
func (nr *nacosRegistry) Subscribe(url *common.URL, notifyListener registry.NotifyListener) error {
	// TODO
	role, _ := strconv.Atoi(url.GetParam(constant.RegistryRoleKey, ""))
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
			logger.Infof("[Nacos Registry] Update begin, service event: %v", serviceEvent.String())
			notifyListener.Notify(serviceEvent)
		}
	}
}

// UnSubscribe :
func (nr *nacosRegistry) UnSubscribe(url *common.URL, _ registry.NotifyListener) error {
	param := createSubscribeParam(url, nr.URL, nil)
	if param == nil {
		return nil
	}
	err := nr.namingClient.Client().Unsubscribe(param)
	if err != nil {
		return perrors.New("UnSubscribe [" + param.ServiceName + "] to nacos failed")
	}
	return nil
}

func createSubscribeParam(url, regUrl *common.URL, cb callback) *vo.SubscribeParam {
	serviceName := getSubscribeName(url)
	groupName := regUrl.GetParam(constant.RegistryGroupKey, defaultGroup)
	if cb == nil {
		v, ok := listenerCache.Load(serviceName + groupName)
		if !ok {
			return nil
		}
		listener, ok := v.(*nacosListener)
		if !ok {
			return nil
		}
		cb = listener.Callback
	}
	return &vo.SubscribeParam{
		ServiceName:       serviceName,
		SubscribeCallback: cb,
		GroupName:         groupName,
	}
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
		err := nr.UnRegister(url)
		logger.Infof("DeRegister Nacos URL:%+v", url)
		if err != nil {
			logger.Errorf("Deregister URL:%+v err:%v", url, err.Error())
		}
	}
	return
}

// newNacosRegistry will create new instance
func newNacosRegistry(url *common.URL) (registry.Registry, error) {
	logger.Infof("[Nacos Registry] New nacos registry with url = %+v", url.ToMap())
	// key transfer: registry -> nacos
	url.SetParam(constant.NacosNamespaceID, url.GetParam(constant.RegistryNamespaceKey, ""))
	url.SetParam(constant.NacosUsername, url.Username)
	url.SetParam(constant.NacosPassword, url.Password)
	url.SetParam(constant.NacosAccessKey, url.GetParam(constant.RegistryAccessKey, ""))
	url.SetParam(constant.NacosSecretKey, url.GetParam(constant.RegistrySecretKey, ""))
	url.SetParam(constant.NacosTimeout, url.GetParam(constant.RegistryTimeoutKey, constant.DefaultRegTimeout))
	url.SetParam(constant.NacosGroupKey, url.GetParam(constant.RegistryGroupKey, defaultGroup))
	namingClient, err := nacos.NewNacosClientByURL(url)
	if err != nil {
		return &nacosRegistry{}, err
	}
	tmpRegistry := &nacosRegistry{
		URL:          url, // registry.group is recorded at this url
		namingClient: namingClient,
		registryUrls: []*common.URL{},
	}
	return tmpRegistry, nil
}
