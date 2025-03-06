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
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"
)

import (
	nacosClient "github.com/dubbogo/gost/database/kv/nacos"
	"github.com/dubbogo/gost/log/logger"

	"github.com/nacos-group/nacos-sdk-go/v2/vo"

	perrors "github.com/pkg/errors"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/common/extension"
	"dubbo.apache.org/dubbo-go/v3/metrics"
	metricsRegistry "dubbo.apache.org/dubbo-go/v3/metrics/registry"
	"dubbo.apache.org/dubbo-go/v3/registry"
	"dubbo.apache.org/dubbo-go/v3/remoting"
	"dubbo.apache.org/dubbo-go/v3/remoting/nacos"
)

const (
	RegistryConnDelay = 3
)

func init() {
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
	common.HandleRegisterIPAndPort(url)
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
	start := time.Now()
	serviceName := getServiceName(url)
	groupName := nr.URL.GetParam(constant.NacosGroupKey, defaultGroup)
	param := createRegisterParam(url, serviceName, groupName)
	logger.Infof("[Nacos Registry] Registry instance with param = %+v", param)
	isRegistry, err := nr.namingClient.Client().RegisterInstance(param)
	metrics.Publish(metricsRegistry.NewRegisterEvent(err == nil && isRegistry, start))
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
	common.HandleRegisterIPAndPort(url)
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

// Subscribe returns nil if subscribing registry successfully. If not returns an error.
func (nr *nacosRegistry) Subscribe(url *common.URL, notifyListener registry.NotifyListener) error {
	role, _ := strconv.Atoi(url.GetParam(constant.RegistryRoleKey, ""))
	if role != common.CONSUMER {
		return nil
	}
	serviceName := url.GetParam(constant.InterfaceKey, "")
	var serviceNames []string
	var err error
	if serviceName == constant.AnyValue {
		serviceNames, err = nr.getAllSubscribeServiceNames(url)
		if err != nil {
			return err
		}
	} else {
		serviceNames = []string{getSubscribeName(url)}
	}
	return nr.subscribe(serviceNames, notifyListener)
}

// subscribe subscribe services
func (nr *nacosRegistry) subscribe(serviceNames []string, notifyListener registry.NotifyListener) error {
	if len(serviceNames) == 0 {
		logger.Warnf("No services to listen to.")
		return nil
	}
	for _, serviceName := range serviceNames {
		serviceName := serviceName
		go func() {
			for {
				if !nr.IsAvailable() {
					logger.Warnf("event listener game over.")
					return
				}
				listener := NewNacosListenerWithServiceName(serviceName, nr.URL, nr.namingClient)
				err := listener.listenService(serviceName)
				metrics.Publish(metricsRegistry.NewSubscribeEvent(err == nil))
				if err != nil {
					logger.Warnf("listen service s% err:%v", serviceName, perrors.WithStack(err))
					time.Sleep(time.Duration(RegistryConnDelay) * time.Second)
				} else {
					// this will block for loop
					nr.handleServiceEvents(listener, notifyListener)
				}
			}
		}()
	}
	return nil
}

// getAllServices retrieves the list of all services from the registry
func (nr *nacosRegistry) getAllSubscribeServiceNames(url *common.URL) ([]string, error) {
	services, err := nr.namingClient.Client().GetAllServicesInfo(vo.GetAllServiceInfoParam{
		GroupName: nr.GetParam(constant.RegistryGroupKey, defaultGroup),
		PageNo:    1,
		PageSize:  math.MaxInt32,
	})
	if err != nil {
		logger.Errorf("query services error: %v", err)
		return nil, err
	}
	var subScribeServiceNames []string
	categories := strings.Split(url.GetParam(constant.CategoryKey, constant.DefaultCategory), constant.CommaSeparator)
	for _, dom := range services.Doms {
		if strings.Contains(dom, constant.NacosServiceNameSeparator) {
			realCategory := strings.Split(dom, constant.NacosServiceNameSeparator)[0]
			for _, item := range categories {
				if item == realCategory {
					subScribeServiceNames = append(subScribeServiceNames, dom)
				}
			}
		}
	}

	return subScribeServiceNames, err
}

// handleServiceEvents receives service events from the listener and notifies the notifyListener
func (nr *nacosRegistry) handleServiceEvents(listener registry.Listener, notifyListener registry.NotifyListener) {
	for {
		serviceEvent, err := listener.Next()
		if err != nil {
			logger.Warnf("Selector.watch() = error{%v}", perrors.WithStack(err))
			listener.Close()
			return
		}
		logger.Infof("[Nacos Registry] Update begin, service event: %v", serviceEvent.String())
		notifyListener.Notify(serviceEvent)
	}
}

// UnSubscribe :
func (nr *nacosRegistry) UnSubscribe(url *common.URL, _ registry.NotifyListener) error {
	param := createSubscribeParam(getSubscribeName(url), nr.URL, nil)
	if param == nil {
		return nil
	}
	err := nr.namingClient.Client().Unsubscribe(param)
	if err != nil {
		return perrors.New("UnSubscribe [" + param.ServiceName + "] to nacos failed")
	}
	return nil
}

// LoadSubscribeInstances load subscribe instance
func (nr *nacosRegistry) LoadSubscribeInstances(url *common.URL, notify registry.NotifyListener) error {
	serviceName := getSubscribeName(url)
	groupName := nr.GetURL().GetParam(constant.RegistryGroupKey, defaultGroup)
	instances, err := nr.namingClient.Client().SelectAllInstances(vo.SelectAllInstancesParam{
		ServiceName: serviceName,
		GroupName:   groupName,
	})
	if err != nil {
		return perrors.New(fmt.Sprintf("could not query the instances for serviceName=%s,groupName=%s,error=%v",
			serviceName, groupName, err))
	}

	for i := range instances {
		if newUrl := generateUrl(instances[i]); newUrl != nil {
			notify.Notify(&registry.ServiceEvent{Action: remoting.EventTypeAdd, Service: newUrl})
		}
	}
	return nil
}

func createSubscribeParam(serviceName string, regUrl *common.URL, cb callback) *vo.SubscribeParam {
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
