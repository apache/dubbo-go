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
	gxnet "github.com/dubbogo/gost/net"
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
	//RegistryConnDelay registry connection delay
	RegistryConnDelay = 3
)

func init() {
	localIP, _ = gxnet.GetLocalIP()
	extension.SetRegistry(constant.NACOS_KEY, newNacosRegistry)
}

type nacosRegistry struct {
	baseRegistry
}

func newNacosRegistry(url *common.URL) (registry.Registry, error) {
	base, err := newBaseRegistry(url)
	if err != nil {
		return nil, perrors.WithStack(err)
	}
	return &nacosRegistry{
		base,
	}, nil
}

func getCategory(url common.URL) string {
	role, _ := strconv.Atoi(url.GetParam(constant.ROLE_KEY, strconv.Itoa(constant.NACOS_DEFAULT_ROLETYPE)))
	category := common.DubboNodes[role]
	return category
}

func getServiceName(url common.URL) string {
	var buffer bytes.Buffer

	buffer.Write([]byte(getCategory(url)))
	appendParam(&buffer, url, constant.INTERFACE_KEY)
	appendParam(&buffer, url, constant.VERSION_KEY)
	appendParam(&buffer, url, constant.GROUP_KEY)
	return buffer.String()
}

func appendParam(target *bytes.Buffer, url common.URL, key string) {
	value := url.GetParam(key, "")
	if strings.TrimSpace(value) != "" {
		target.Write([]byte(constant.NACOS_SERVICE_NAME_SEPARATOR))
		target.Write([]byte(value))
	}
}

func createRegisterParam(url common.URL, serviceName string) vo.RegisterInstanceParam {
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

func (nr *nacosRegistry) Register(url common.URL) error {
	serviceName := getServiceName(url)
	param := createRegisterParam(url, serviceName)
	isRegistry, err := nr.namingClient.RegisterInstance(param)
	if err != nil {
		return err
	}
	if !isRegistry {
		return perrors.New("registry [" + serviceName + "] to  nacos failed")
	}
	return nil
}

func (nr *nacosRegistry) subscribe(conf *common.URL) (registry.Listener, error) {
	return NewNacosListener(*conf, nr.namingClient)
}

//subscribe from registry
func (nr *nacosRegistry) Subscribe(url *common.URL, notifyListener registry.NotifyListener) {
	for {
		if !nr.IsAvailable() {
			logger.Warnf("event listener game over.")
			return
		}

		listener, err := nr.subscribe(url)
		if err != nil {
			if !nr.IsAvailable() {
				logger.Warnf("event listener game over.")
				return
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
				return
			}

			logger.Infof("update begin, service event: %v", serviceEvent.String())
			notifyListener.Notify(serviceEvent)
		}

	}
}

func (nr *nacosRegistry) GetUrl() common.URL {
	return *nr.URL
}

func (nr *nacosRegistry) IsAvailable() bool {
	return true
}

func (nr *nacosRegistry) Destroy() {
	return
}
