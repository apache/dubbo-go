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

package consul

import (
	"encoding/base64"
	"fmt"
	"strconv"
	"sync"
	"time"
)

import (
	"github.com/dubbogo/gost/container/set"
	"github.com/dubbogo/gost/page"
	consul "github.com/hashicorp/consul/api"
	"github.com/hashicorp/consul/api/watch"
	perrors "github.com/pkg/errors"
)

import (
	"github.com/apache/dubbo-go/common/constant"
	"github.com/apache/dubbo-go/common/extension"
	"github.com/apache/dubbo-go/common/logger"
	"github.com/apache/dubbo-go/config"
	"github.com/apache/dubbo-go/registry"
)

const (
	enable                 = "enable"
	watch_type             = "type"
	watch_type_service     = "service"
	watch_service          = "service"
	watch_passingonly      = "passingonly"
	watch_passingonly_true = true
)

var (
	errConsulClientClosed = perrors.New("consul client is closed")
)

// init will put the service discovery into extension
func init() {
	extension.SetServiceDiscovery(constant.CONSUL_KEY, newConsulServiceDiscovery)
}

// consulServiceDiscovery is the implementation of service discovery based on consul.
type consulServiceDiscovery struct {
	// descriptor is a short string about the basic information of this instance
	descriptor string
	clientLock sync.RWMutex
	// Consul client.
	consulClient                   *consul.Client
	checkPassInterval              int64
	tag                            string
	address                        string
	deregisterCriticalServiceAfter string
	ttl                            sync.Map
	*consul.Config
}

// newConsulServiceDiscovery will create new service discovery instance
// use double-check pattern to reduce race condition
func newConsulServiceDiscovery(name string) (registry.ServiceDiscovery, error) {
	sdc, ok := config.GetBaseConfig().GetServiceDiscoveries(name)
	if !ok || len(sdc.RemoteRef) == 0 {
		return nil, perrors.New("could not init the instance because the config is invalid")
	}

	remoteConfig, ok := config.GetBaseConfig().GetRemoteConfig(sdc.RemoteRef)
	if !ok {
		return nil, perrors.New("could not find the remote config for name: " + sdc.RemoteRef)
	}

	descriptor := fmt.Sprintf("consul-service-discovery[%s]", remoteConfig.Address)

	config := &consul.Config{Address: remoteConfig.Address, Token: remoteConfig.Params[constant.ACL_TOKEN]}
	client, err := consul.NewClient(config)
	if err != nil {
		return nil, perrors.WithMessage(err, "create consul client failed.")
	}

	return &consulServiceDiscovery{
		address:                        remoteConfig.Address,
		descriptor:                     descriptor,
		checkPassInterval:              getCheckPassInterval(remoteConfig.Params),
		Config:                         config,
		tag:                            remoteConfig.Params[constant.QUERY_TAG],
		consulClient:                   client,
		deregisterCriticalServiceAfter: getDeregisterAfter(remoteConfig.Params),
		clientLock:                     sync.RWMutex{},
	}, nil
}

func (csd *consulServiceDiscovery) String() string {
	return csd.descriptor
}

// nolint
func (csd *consulServiceDiscovery) getConsulClient() *consul.Client {
	csd.clientLock.RLock()
	defer csd.clientLock.RUnlock()
	return csd.consulClient
}

// nolint
func (csd *consulServiceDiscovery) setConsulClient(consulClient *consul.Client) {
	csd.clientLock.Lock()
	defer csd.clientLock.Unlock()
	csd.consulClient = consulClient
}

func (csd *consulServiceDiscovery) Destroy() error {
	csd.setConsulClient(nil)
	csd.ttl.Range(func(key, t interface{}) bool {
		close(t.(chan struct{}))
		csd.ttl.Delete(key)
		return true
	})
	return nil
}

func (csd *consulServiceDiscovery) Register(instance registry.ServiceInstance) error {
	var (
		err          error
		consulClient *consul.Client
	)
	ins, _ := csd.buildRegisterInstance(instance)
	if consulClient = csd.getConsulClient(); consulClient == nil {
		return errConsulClientClosed
	}
	err = consulClient.Agent().ServiceRegister(ins)
	if err != nil {
		logger.Errorf("consul register the instance %s fail:%v", instance.GetServiceName(), err)
		return perrors.WithMessage(err, "consul could not register the instance. "+instance.GetServiceName())
	}

	return csd.registerTtl(instance)
}

func (csd *consulServiceDiscovery) registerTtl(instance registry.ServiceInstance) error {
	var (
		err          error
		consulClient *consul.Client
	)

	checkID := buildID(instance)

	stopChan := make(chan struct{})
	csd.ttl.LoadOrStore(buildID(instance), stopChan)

	period := time.Duration(csd.checkPassInterval/8) * time.Millisecond
	timer := time.NewTicker(period)
	go func() {
		defer timer.Stop()
		for {
			select {
			case <-timer.C:
				if consulClient = csd.getConsulClient(); consulClient == nil {
					logger.Debugf("consul client is closed!")
					return
				}
				err = consulClient.Agent().PassTTL(fmt.Sprintf("service:%s", checkID), "")
				if err != nil {
					logger.Warnf("pass ttl heartbeat fail:%v", err)
					break
				}
				logger.Debugf("passed ttl heartbeat for %s", checkID)
				break
			case <-stopChan:
				logger.Info("ttl %s for service %s is stopped", checkID, instance.GetServiceName())
				return
			}
		}
	}()
	return nil
}

func (csd *consulServiceDiscovery) Update(instance registry.ServiceInstance) error {
	var (
		err          error
		consulClient *consul.Client
	)
	ins, _ := csd.buildRegisterInstance(instance)
	consulClient = csd.getConsulClient()
	if consulClient == nil {
		return errConsulClientClosed
	}
	err = consulClient.Agent().ServiceDeregister(buildID(instance))
	if err != nil {
		logger.Warnf("unregister instance %s fail:%v", instance.GetServiceName(), err)
	}
	return consulClient.Agent().ServiceRegister(ins)
}

func (csd *consulServiceDiscovery) Unregister(instance registry.ServiceInstance) error {
	var (
		err          error
		consulClient *consul.Client
	)
	if consulClient = csd.getConsulClient(); consulClient == nil {
		return errConsulClientClosed
	}
	err = consulClient.Agent().ServiceDeregister(buildID(instance))
	if err != nil {
		logger.Errorf("unregister service instance %s,error: %v", instance.GetId(), err)
		return err
	}
	stopChanel, ok := csd.ttl.Load(buildID(instance))
	if !ok {
		logger.Warnf("ttl for service instance %s didn't exist", instance.GetId())
		return nil
	}
	close(stopChanel.(chan struct{}))
	csd.ttl.Delete(buildID(instance))
	return nil
}

func (csd *consulServiceDiscovery) GetDefaultPageSize() int {
	return registry.DefaultPageSize
}

func (csd *consulServiceDiscovery) GetServices() *gxset.HashSet {
	var (
		err          error
		consulClient *consul.Client
		services     map[string][]string
	)
	var res = gxset.NewSet()
	if consulClient = csd.getConsulClient(); consulClient == nil {
		logger.Warnf("consul client is closed!")
		return res
	}
	services, _, err = consulClient.Catalog().Services(nil)
	if err != nil {
		logger.Errorf("get services,error: %v", err)
		return res
	}

	for service, _ := range services {
		res.Add(service)
	}
	return res

}

// encodeConsulMetadata because consul validate key strictly.
func encodeConsulMetadata(metadata map[string]string) map[string]string {
	consulMetadata := make(map[string]string, len(metadata))
	encoder := base64.RawStdEncoding
	for k, v := range metadata {
		consulMetadata[encoder.EncodeToString([]byte(k))] = v
	}
	return consulMetadata
}

// nolint
func decodeConsulMetadata(metadata map[string]string) map[string]string {
	dubboMetadata := make(map[string]string, len(metadata))
	encoder := base64.RawStdEncoding
	for k, v := range metadata {
		kBytes, err := encoder.DecodeString(k)
		if err != nil {
			logger.Warnf("can not decoded consul metadata key %s", k)
			continue
		}
		dubboMetadata[string(kBytes)] = v
	}
	return dubboMetadata
}

func (csd *consulServiceDiscovery) GetInstances(serviceName string) []registry.ServiceInstance {
	var (
		err          error
		consulClient *consul.Client
		instances    []*consul.ServiceEntry
	)
	if consulClient = csd.getConsulClient(); consulClient == nil {
		logger.Warn("consul client is closed!")
		return nil
	}
	instances, _, err = consulClient.Health().Service(serviceName, csd.tag, true, &consul.QueryOptions{
		WaitTime: time.Duration(csd.checkPassInterval),
	})

	if err != nil {
		logger.Errorf("get instances for service %s,error: %v", serviceName, err)
		return nil
	}

	res := make([]registry.ServiceInstance, 0, len(instances))
	for _, ins := range instances {
		metadata := ins.Service.Meta

		// enable status
		enableStr := metadata[enable]
		delete(metadata, enable)
		enable, _ := strconv.ParseBool(enableStr)
		metadata = decodeConsulMetadata(metadata)

		// health status
		status := ins.Checks.AggregatedStatus()
		healthy := false
		if status == consul.HealthPassing {
			healthy = true
		}
		res = append(res, &registry.DefaultServiceInstance{
			Id:          ins.Service.ID,
			ServiceName: ins.Service.Service,
			Host:        ins.Service.Address,
			Port:        ins.Service.Port,
			Enable:      enable,
			Healthy:     healthy,
			Metadata:    metadata,
		})
	}

	return res
}

func (csd *consulServiceDiscovery) GetInstancesByPage(serviceName string, offset int, pageSize int) gxpage.Pager {
	all := csd.GetInstances(serviceName)
	res := make([]interface{}, 0, pageSize)
	for i := offset; i < len(all) && i < offset+pageSize; i++ {
		res = append(res, all[i])
	}
	return gxpage.New(offset, pageSize, res, len(all))
}

func (csd *consulServiceDiscovery) GetHealthyInstancesByPage(serviceName string, offset int, pageSize int, healthy bool) gxpage.Pager {
	all := csd.GetInstances(serviceName)
	res := make([]interface{}, 0, pageSize)
	// could not use res = all[a:b] here because the res should be []interface{}, not []ServiceInstance
	var (
		i     = offset
		count = 0
	)
	for i < len(all) && count < pageSize {
		ins := all[i]
		if ins.IsHealthy() == healthy {
			res = append(res, all[i])
			count++
		}
		i++
	}
	return gxpage.New(offset, pageSize, res, len(all))
}

func (csd *consulServiceDiscovery) GetRequestInstances(serviceNames []string, offset int, requestedSize int) map[string]gxpage.Pager {
	res := make(map[string]gxpage.Pager, len(serviceNames))
	for _, name := range serviceNames {
		res[name] = csd.GetInstancesByPage(name, offset, requestedSize)
	}
	return res
}

func (csd *consulServiceDiscovery) AddListener(listener *registry.ServiceInstancesChangedListener) error {

	params := make(map[string]interface{}, 8)
	params[watch_type] = watch_type_service
	params[watch_service] = listener.ServiceName
	params[watch_passingonly] = watch_passingonly_true
	plan, err := watch.Parse(params)
	if err != nil {
		logger.Errorf("add listener for service %s,error:%v", listener.ServiceName, err)
		return err
	}

	plan.Handler = func(idx uint64, raw interface{}) {
		services, ok := raw.([]*consul.ServiceEntry)
		if !ok {
			err = perrors.New("handler get non ServiceEntry type parameter")
			return
		}
		instances := make([]registry.ServiceInstance, 0, len(services))
		for _, ins := range services {
			metadata := ins.Service.Meta

			// enable status
			enableStr := metadata[enable]
			delete(metadata, enable)
			enable, _ := strconv.ParseBool(enableStr)

			// health status
			status := ins.Checks.AggregatedStatus()
			healthy := false
			if status == consul.HealthPassing {
				healthy = true
			}
			instances = append(instances, &registry.DefaultServiceInstance{
				Id:          ins.Service.ID,
				ServiceName: ins.Service.Service,
				Host:        ins.Service.Address,
				Port:        ins.Service.Port,
				Enable:      enable,
				Healthy:     healthy,
				Metadata:    metadata,
			})
		}
		e := csd.DispatchEventForInstances(listener.ServiceName, instances)
		if e != nil {
			logger.Errorf("Dispatching event got exception, service name: %s, err: %v", listener.ServiceName, err)
		}
	}
	go func() {
		err = plan.RunWithConfig(csd.Config.Address, csd.Config)
		if err != nil {
			logger.Error("consul plan run failure!error:%v", err)
		}
	}()
	return nil
}

func (csd *consulServiceDiscovery) DispatchEventByServiceName(serviceName string) error {
	return csd.DispatchEventForInstances(serviceName, csd.GetInstances(serviceName))
}

func (csd *consulServiceDiscovery) DispatchEventForInstances(serviceName string, instances []registry.ServiceInstance) error {
	return csd.DispatchEvent(registry.NewServiceInstancesChangedEvent(serviceName, instances))
}

func (csd *consulServiceDiscovery) DispatchEvent(event *registry.ServiceInstancesChangedEvent) error {
	extension.GetGlobalDispatcher().Dispatch(event)
	return nil
}

func (csd *consulServiceDiscovery) buildRegisterInstance(instance registry.ServiceInstance) (*consul.AgentServiceRegistration, error) {
	metadata := instance.GetMetadata()
	metadata = encodeConsulMetadata(metadata)
	metadata[enable] = strconv.FormatBool(instance.IsEnable())
	// check
	check := csd.buildCheck(instance)

	return &consul.AgentServiceRegistration{
		ID:      buildID(instance),
		Name:    instance.GetServiceName(),
		Port:    instance.GetPort(),
		Address: instance.GetHost(),
		Meta:    metadata,
		Check:   &check,
	}, nil
}

func (csd *consulServiceDiscovery) buildCheck(instance registry.ServiceInstance) consul.AgentServiceCheck {

	deregister, ok := instance.GetMetadata()[constant.DEREGISTER_AFTER]
	if !ok || len(deregister) == 0 {
		deregister = constant.DEFAULT_DEREGISTER_TIME
	}
	return consul.AgentServiceCheck{
		TTL:                            strconv.FormatInt(csd.checkPassInterval/1000, 10) + "s",
		DeregisterCriticalServiceAfter: csd.deregisterCriticalServiceAfter,
	}
}

// nolint
func getCheckPassInterval(params map[string]string) int64 {
	checkPassIntervalStr, ok := params[constant.CHECK_PASS_INTERVAL]
	if !ok {
		return constant.DEFAULT_CHECK_PASS_INTERVAL
	}
	checkPassInterval, err := strconv.ParseInt(checkPassIntervalStr, 10, 64)
	if err != nil {
		logger.Warnf("consul service discovery remote config error:%s", checkPassIntervalStr)
		return constant.DEFAULT_CHECK_PASS_INTERVAL
	}
	return checkPassInterval
}

// nolint
func getDeregisterAfter(metadata map[string]string) string {
	deregister, ok := metadata[constant.DEREGISTER_AFTER]
	if !ok || len(deregister) == 0 {
		deregister = constant.DEFAULT_DEREGISTER_TIME
	}
	return deregister
}

// nolint
func buildID(instance registry.ServiceInstance) string {
	id := fmt.Sprintf("id:%s,serviceName:%s,host:%s,port:%d", instance.GetId(), instance.GetServiceName(), instance.GetHost(), instance.GetPort())
	return id
}
