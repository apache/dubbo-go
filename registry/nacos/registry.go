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
	"net"
	"strconv"
	"strings"
	"time"
)

import (
	"github.com/nacos-group/nacos-sdk-go/clients"
	"github.com/nacos-group/nacos-sdk-go/clients/naming_client"
	nacosConstant "github.com/nacos-group/nacos-sdk-go/common/constant"
	"github.com/nacos-group/nacos-sdk-go/vo"
	perrors "github.com/pkg/errors"
)

import (
	"github.com/apache/dubbo-go/common"
	"github.com/apache/dubbo-go/common/constant"
	"github.com/apache/dubbo-go/common/extension"
	"github.com/apache/dubbo-go/common/logger"
	"github.com/apache/dubbo-go/registry"
)

var (
	localIP = ""
)

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
	namingClient naming_client.INamingClient
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
	if strings.TrimSpace(value) != "" {
		target.Write([]byte(constant.NACOS_SERVICE_NAME_SEPARATOR))
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
	isRegistry, err := nr.namingClient.RegisterInstance(param)
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
	isDeRegistry, err := nr.namingClient.DeregisterInstance(param)
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

// GetUrl gets its registration URL
func (nr *nacosRegistry) GetUrl() *common.URL {
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
	nacosConfig, err := getNacosConfig(url)
	if err != nil {
		return &nacosRegistry{}, err
	}
	client, err := clients.CreateNamingClient(nacosConfig)
	if err != nil {
		return &nacosRegistry{}, err
	}
	tmpRegistry := &nacosRegistry{
		URL:          url,
		namingClient: client,
		registryUrls: []*common.URL{},
	}
	return tmpRegistry, nil
}

// getNacosConfig will return the nacos config
// TODO support RemoteRef
func getNacosConfig(url *common.URL) (map[string]interface{}, error) {
	if url == nil {
		return nil, perrors.New("url is empty!")
	}
	if len(url.Location) == 0 {
		return nil, perrors.New("url.location is empty!")
	}
	configMap := make(map[string]interface{}, 2)

	addresses := strings.Split(url.Location, ",")
	serverConfigs := make([]nacosConstant.ServerConfig, 0, len(addresses))
	for _, addr := range addresses {
		ip, portStr, err := net.SplitHostPort(addr)
		if err != nil {
			return nil, perrors.WithMessagef(err, "split [%s] ", addr)
		}
		port, _ := strconv.Atoi(portStr)
		serverConfigs = append(serverConfigs, nacosConstant.ServerConfig{
			IpAddr: ip,
			Port:   uint64(port),
		})
	}
	configMap["serverConfigs"] = serverConfigs

	var clientConfig nacosConstant.ClientConfig
	timeout, err := time.ParseDuration(url.GetParam(constant.REGISTRY_TIMEOUT_KEY, constant.DEFAULT_REG_TIMEOUT))
	if err != nil {
		return nil, err
	}
	clientConfig.TimeoutMs = uint64(timeout.Seconds() * 1000)
	clientConfig.ListenInterval = 2 * clientConfig.TimeoutMs
	clientConfig.CacheDir = url.GetParam(constant.NACOS_CACHE_DIR_KEY, "")
	clientConfig.LogDir = url.GetParam(constant.NACOS_LOG_DIR_KEY, "")
	clientConfig.Endpoint = url.GetParam(constant.NACOS_ENDPOINT, "")
	clientConfig.NamespaceId = url.GetParam(constant.NACOS_NAMESPACE_ID, "")
	clientConfig.NotLoadCacheAtStart = true
	configMap["clientConfig"] = clientConfig

	return configMap, nil
}
