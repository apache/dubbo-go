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
	"os"
	"strconv"
	"strings"
)

import (
	"github.com/dubbogo/gost/log/logger"

	perrors "github.com/pkg/errors"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/common/extension"
	"dubbo.apache.org/dubbo-go/v3/registry"
	"dubbo.apache.org/dubbo-go/v3/remoting/xds"
	common2 "dubbo.apache.org/dubbo-go/v3/remoting/xds/common"
)

var localIP = ""
var DefaultXDSSniffingTimeoutStr = "5s"

func init() {
	localIP = common.GetLocalIp()
	extension.SetRegistry(constant.XDSRegistryKey, newXDSRegistry)
}

type xdsRegistry struct {
	xdsWrappedClient xds.XDSWrapperClient
	registryURL      *common.URL
}

func isProvider(url *common.URL) bool {
	return getCategory(url) == constant.ProviderCategory
}

func getCategory(url *common.URL) string {
	role, _ := strconv.Atoi(url.GetParam(constant.RegistryRoleKey, strconv.Itoa(constant.NacosDefaultRoleType)))
	category := common.DubboNodes[role]
	return category
}

// getServiceName return serviceName $(providers_or_consumers):$(interfaceName):$(version):$(group)
func getServiceName(url *common.URL) string {
	var buffer bytes.Buffer

	buffer.Write([]byte(getCategory(url)))
	appendParam(&buffer, url, constant.InterfaceKey)
	appendParam(&buffer, url, constant.VersionKey)
	appendParam(&buffer, url, constant.GroupKey)
	return buffer.String()
}

// getSubscribeName returns subscribeName is providers:$(interfaceName):$(version):$(group)
func getSubscribeName(url *common.URL) string {
	var buffer bytes.Buffer

	buffer.Write([]byte(common.DubboNodes[common.PROVIDER]))
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
	if !isProvider(url) {
		return nil
	}
	return nr.xdsWrappedClient.ChangeInterfaceMap(getServiceName(url), true)
}

// UnRegister
func (nr *xdsRegistry) UnRegister(url *common.URL) error {
	return nr.xdsWrappedClient.ChangeInterfaceMap(getServiceName(url), false)
}

// Subscribe from xds client
func (nr *xdsRegistry) Subscribe(url *common.URL, notifyListener registry.NotifyListener) error {
	if isProvider(url) {
		return nil
	}
	hostAddr, svcAddr, err := nr.getHostAddrFromURL(url)
	if err != nil {
		return err
	}
	return nr.xdsWrappedClient.Subscribe(svcAddr, url.GetParam(constant.InterfaceKey, ""), hostAddr, notifyListener)
}

// UnSubscribe from xds client
func (nr *xdsRegistry) UnSubscribe(url *common.URL, _ registry.NotifyListener) error {
	_, svcAddr, err := nr.getHostAddrFromURL(url)
	if err != nil {
		return err
	}
	nr.xdsWrappedClient.UnSubscribe(svcAddr)
	return nil
}

// GetURL gets its registration URL
func (nr *xdsRegistry) GetURL() *common.URL {
	return nr.registryURL
}

func (nr *xdsRegistry) getHostAddrFromURL(url *common.URL) (string, string, error) {
	svcName := getSubscribeName(url)
	hostAddr, err := nr.xdsWrappedClient.GetHostAddrByServiceUniqueKey(svcName)
	return hostAddr, svcName, err
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
	logger.Infof("[XDS Registry] New XDS registry with url = %+v", url.ToMap())
	pn := os.Getenv(constant.PodNameEnvKey)
	ns := os.Getenv(constant.PodNamespaceEnvKey)
	if pn == "" || ns == "" {
		return nil, perrors.Errorf("%s and %s can't be empty when using xds registry",
			constant.PodNameEnvKey, constant.PodNamespaceEnvKey)
	}

	wrappedXDSClient, err := xds.NewXDSWrappedClient(xds.Config{
		PodName:   pn,
		Namespace: ns,
		LocalIP:   localIP,
		IstioAddr: common2.NewHostNameOrIPAddr(url.Ip + ":" + url.Port),
	})
	if err != nil {
		return nil, err
	}

	newRegistry := &xdsRegistry{
		xdsWrappedClient: wrappedXDSClient,
		registryURL:      url,
	}

	return newRegistry, nil
}
