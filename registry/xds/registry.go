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

package xds

import (
	"bytes"
	"dubbo.apache.org/dubbo-go/v3/remoting/xds"
	"strconv"
	"strings"
)

import (
	perrors "github.com/pkg/errors"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/common/extension"
	"dubbo.apache.org/dubbo-go/v3/common/logger"
	"dubbo.apache.org/dubbo-go/v3/registry"
)

var localIP = ""

const (
	// RegistryConnDelay registry connection delay
	RegistryConnDelay = 3
)

func init() {
	localIP = common.GetLocalIp()
	extension.SetRegistry(constant.XDSRegistryKey, newXDSRegistry)
}

type xdsRegistry struct {
	xdsWrappedClient *xds.WrappedClient
	registryURL      *common.URL
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

// Register will register the service @url to its nacos registry center
func (nr *xdsRegistry) Register(url *common.URL) error {
	// todo get hostName of this app
	return nr.xdsWrappedClient.ChangeInterfaceMap(getServiceName(url), "host name of this")
}

// UnRegister
func (nr *xdsRegistry) UnRegister(url *common.URL) error {
	return nr.xdsWrappedClient.ChangeInterfaceMap(getServiceName(url), "")
}

// Subscribe from xds client
func (nr *xdsRegistry) Subscribe(url *common.URL, notifyListener registry.NotifyListener) error {
	hostName, err := nr.getHostNameFromURL(url)
	if err != nil {
		return err
	}
	return nr.xdsWrappedClient.Subscribe(hostName, notifyListener)
}

// UnSubscribe from xds client
func (nr *xdsRegistry) UnSubscribe(url *common.URL, _ registry.NotifyListener) error {
	hostName, err := nr.getHostNameFromURL(url)
	if err != nil {
		return err
	}
	nr.xdsWrappedClient.UnSubscribe(hostName)
	return nil
}

// GetURL gets its registration URL
func (nr *xdsRegistry) GetURL() *common.URL {
	return nr.registryURL
}

func (nr *xdsRegistry) getHostNameFromURL(url *common.URL) (string, error) {
	interfaceAppMap := nr.xdsWrappedClient.GetInterfaceAppNameMapFromPilot()
	svcName := getServiceName(url)
	hostName, ok := interfaceAppMap[svcName]
	if !ok {
		return "", perrors.Errorf("service %s not found", svcName)
	}
	return hostName, nil
}

// IsAvailable determines nacos registry center whether it is available
func (nr *xdsRegistry) IsAvailable() bool {
	// TODO
	return true
}

// nolint
func (nr *xdsRegistry) Destroy() {
	// todo unregistry all
	//for _, url := range nr.registryUrls {
	//	err := nr.UnRegister(url)
	//	logger.Infof("DeRegister Nacos URL:%+v", url)
	//	if err != nil {
	//		logger.Errorf("Deregister URL:%+v err:%v", url, err.Error())
	//	}
	//}
	return
}

// newXDSRegistry will create new instance
func newXDSRegistry(url *common.URL) (registry.Registry, error) {
	logger.Infof("[XDS Registry] New nacos registry with url = %+v", url.ToMap())
	// todo get podName, get Namespace Name

	wrappedXDSClient := xds.NewXDSWrappedClient("", "", localIP, url.Location)
	tmpRegistry := &xdsRegistry{
		xdsWrappedClient: wrappedXDSClient,
		registryURL:      url,
	}
	return tmpRegistry, nil
}
