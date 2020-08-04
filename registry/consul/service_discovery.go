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
	"fmt"
	"github.com/hashicorp/consul/api/watch"
	"github.com/hashicorp/go-hclog"
	"strconv"
	"sync"
)

import (
	"github.com/dubbogo/gost/container/set"
	"github.com/dubbogo/gost/page"
	consul "github.com/hashicorp/consul/api"
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
	PageSize = "pageSize"
	Enable   = "enable"
)

var (
	// 16 would be enough. We won't use concurrentMap because in most cases, there are not race condition
	instanceMap = make(map[string]registry.ServiceDiscovery, 16)
	initLock    sync.Mutex
)

// init will put the service discovery into extension
func init() {
	extension.SetServiceDiscovery(constant.CONSUL_KEY, newConsulServiceDiscovery)
}

// newConsulServiceDiscovery will create new service discovery instance
// use double-check pattern to reduce race condition
func newConsulServiceDiscovery(name string) (registry.ServiceDiscovery, error) {

	instance, ok := instanceMap[name]
	if ok {
		return instance, nil
	}

	initLock.Lock()
	defer initLock.Unlock()

	// double check
	instance, ok = instanceMap[name]
	if ok {
		return instance, nil
	}

	sdc, ok := config.GetBaseConfig().GetServiceDiscoveries(name)
	if !ok || len(sdc.RemoteRef) == 0 {
		return nil, perrors.New("could not init the instance because the config is invalid")
	}

	remoteConfig, ok := config.GetBaseConfig().GetRemoteConfig(sdc.RemoteRef)
	if !ok {
		return nil, perrors.New("could not find the remote config for name: " + sdc.RemoteRef)
	}

	config := &consul.Config{Address: remoteConfig.Address}
	client, err := consul.NewClient(config)
	if err != nil {
		return nil, perrors.WithMessage(err, "create consul client failed.")
	}

	descriptor := fmt.Sprintf("consul-service-discovery[%s]", remoteConfig.Address)

	pageSize := 20
	if remoteConfig.Params != nil {
		if tmp, OK := remoteConfig.Params[PageSize]; OK {
			intTmp, err := strconv.Atoi(tmp)
			if err == nil && intTmp > 20 {
				pageSize = intTmp
			}
		}
	}
	return &consulServiceDiscovery{
		consulClient: client,
		descriptor:   descriptor,
		PageSize:     pageSize,
	}, nil
}

// nacosServiceDiscovery is the implementation of service discovery based on nacos.
// There is a problem, the go client for nacos does not support the id field.
// we will use the metadata to store the id of ServiceInstance
type consulServiceDiscovery struct {
	group string
	// descriptor is a short string about the basic information of this instance
	descriptor string
	// Consul client.
	consulClient *consul.Client
	PageSize     int
}

func (csd consulServiceDiscovery) String() string {
	return csd.descriptor
}

func (csd consulServiceDiscovery) Destroy() error {
	csd.consulClient = nil
	return nil
}

func (csd consulServiceDiscovery) Register(instance registry.ServiceInstance) error {
	ins, err := csd.buildRegisterInstance(instance)
	if err != nil {
		panic(err)
	}
	err = csd.consulClient.Agent().ServiceRegister(ins)
	if err != nil {
		return perrors.WithMessage(err, "consul could not register the instance. "+instance.GetServiceName())
	}
	return nil

}

func (csd consulServiceDiscovery) Update(instance registry.ServiceInstance) error {
	ins, err := csd.buildRegisterInstance(instance)
	if err != nil {
		panic(err)
	}
	return csd.consulClient.Agent().ServiceRegisterOpts(ins, consul.ServiceRegisterOpts{ReplaceExistingChecks: true})
}

func (csd consulServiceDiscovery) Unregister(instance registry.ServiceInstance) error {
	return csd.consulClient.Agent().ServiceDeregister(instance.GetId())
}

func (csd consulServiceDiscovery) GetDefaultPageSize() int {
	return csd.PageSize
}

func (csd consulServiceDiscovery) GetServices() *gxset.HashSet {

	var res = gxset.NewSet()
	services, err := csd.consulClient.Agent().Services()
	if err != nil {
		return res
	}

	for _, service := range services {
		res.Add(service.Service)
	}
	return res

}

func (csd consulServiceDiscovery) GetInstances(serviceName string) []registry.ServiceInstance {
	_, instances, err := csd.consulClient.Agent().AgentHealthServiceByName(serviceName)
	if err != nil {
		return nil
	}

	res := make([]registry.ServiceInstance, 0, len(instances))
	for _, ins := range instances {
		metadata := ins.Service.Meta

		// enable status
		enableStr := metadata[Enable]
		delete(metadata, Enable)
		enable, _ := strconv.ParseBool(enableStr)

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

func (csd consulServiceDiscovery) GetInstancesByPage(serviceName string, offset int, pageSize int) gxpage.Pager {
	all := csd.GetInstances(serviceName)
	res := make([]interface{}, 0, pageSize)
	for i := offset; i < len(all) && i < offset+pageSize; i++ {
		res = append(res, all[i])
	}
	return gxpage.New(offset, pageSize, res, len(all))
}

func (csd consulServiceDiscovery) GetHealthyInstancesByPage(serviceName string, offset int, pageSize int, healthy bool) gxpage.Pager {
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

func (csd consulServiceDiscovery) GetRequestInstances(serviceNames []string, offset int, requestedSize int) map[string]gxpage.Pager {
	res := make(map[string]gxpage.Pager, len(serviceNames))
	for _, name := range serviceNames {
		res[name] = csd.GetInstancesByPage(name, offset, requestedSize)
	}
	return res
}

func (csd consulServiceDiscovery) AddListener(listener *registry.ServiceInstancesChangedListener) error {

	params := make(map[string]interface{}, 8)
	params["type"] = "service"
	params["service"] = listener.ServiceName
	//params["tag"] = "dubbo"
	//params["passingonly"] = true
	plan, err := watch.Parse(params)
	if err != nil {
		return err
	}

	hcLogger := hclog.New(&hclog.LoggerOptions{
		Name:   "watch",
		Output: plan.LogOutput,
	})
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
			enableStr := metadata[Enable]
			delete(metadata, Enable)
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
		if len(instances) < 1 {
			return
		}
		e := csd.DispatchEventForInstances(listener.ServiceName, instances)
		if e != nil {
			logger.Errorf("Dispatching event got exception, service name: %s, err: %v", listener.ServiceName, err)
		}
	}
	err = plan.RunWithClientAndHclog(csd.consulClient, hcLogger)
	if err != nil {
		logger.Error("consul plan run failure!error:%v", err)
		return err
	}
	return nil
}

func (csd consulServiceDiscovery) DispatchEventByServiceName(serviceName string) error {
	return csd.DispatchEventForInstances(serviceName, csd.GetInstances(serviceName))
}

func (csd consulServiceDiscovery) DispatchEventForInstances(serviceName string, instances []registry.ServiceInstance) error {
	return csd.DispatchEvent(registry.NewServiceInstancesChangedEvent(serviceName, instances))
}

func (csd consulServiceDiscovery) DispatchEvent(event *registry.ServiceInstancesChangedEvent) error {
	extension.GetGlobalDispatcher().Dispatch(event)
	return nil
}

func (csd consulServiceDiscovery) buildRegisterInstance(instance registry.ServiceInstance) (*consul.AgentServiceRegistration, error) {
	metadata := instance.GetMetadata()
	if metadata == nil {
		metadata = make(map[string]string, 1)
	}
	metadata[Enable] = strconv.FormatBool(instance.IsEnable())

	// tcp
	tcp := fmt.Sprintf("%s:%d", instance.GetHost(), instance.GetPort())

	// check
	check := &consul.AgentServiceCheck{
		TCP: tcp,
		//Interval:                       url.GetParam("consul-check-interval", "10s"),
		//Timeout:                        url.GetParam("consul-check-timeout", "1s"),
		//DeregisterCriticalServiceAfter: url.GetParam("consul-deregister-critical-service-after", "20s"),
	}

	return &consul.AgentServiceRegistration{
		ID:      instance.GetId(),
		Name:    instance.GetServiceName(),
		Port:    instance.GetPort(),
		Address: instance.GetHost(),
		Meta:    metadata,
		Check:   check,
	}, nil
}
