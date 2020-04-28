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

package eureka

import (
	"fmt"
	"net"
	"strings"
)

import (
	gxset "github.com/dubbogo/gost/container/set"
	gxpage "github.com/dubbogo/gost/page"
	"github.com/hudl/fargo"
	perrors "github.com/pkg/errors"
)

import (
	"github.com/apache/dubbo-go/common"
	"github.com/apache/dubbo-go/common/constant"
	"github.com/apache/dubbo-go/common/extension"
	"github.com/apache/dubbo-go/common/logger"
	"github.com/apache/dubbo-go/registry"
)

// init will put the service discovery into extension
func init() {
	extension.SetServiceDiscovery(constant.EUREKA_KEY, newEurekaServiceDiscovery)
}

// eurekaServiceDiscovery is the implementation of service discovery based on eureka.
type eurekaServiceDiscovery struct {
	eurekaConnection *fargo.EurekaConnection
	group            string
	*common.URL
}

// toDeregisterInstance will convert the ServiceInstance to DeregisterInstanceParam
func (e *eurekaServiceDiscovery) toDeregisterInstance(instance registry.ServiceInstance) *fargo.Instance {
	return &fargo.Instance{
		HomePageUrl: instance.GetServiceName(),
		IPAddr:      instance.GetHost(),
		Port:        instance.GetPort(),
		App:         e.group,
	}
}

// Destroy will close the service discovery.
// Actually, it only marks the eurekaConnection as null and then return
func (e *eurekaServiceDiscovery) Destroy() error {
	e.eurekaConnection = nil
	return nil
}

func (e *eurekaServiceDiscovery) Register(instance registry.ServiceInstance) error {
	err := e.eurekaConnection.RegisterInstance(e.toDeregisterInstance(instance))
	if err != nil {
		return perrors.WithMessage(err, "Could not unregister the instance. "+instance.GetServiceName())
	}
	return nil
}

func (e *eurekaServiceDiscovery) Update(instance registry.ServiceInstance) error {
	err := e.Unregister(instance)
	if err != nil {
		return perrors.WithStack(err)
	}
	return e.Register(instance)
}

// Unregister will unregister the instance
func (e *eurekaServiceDiscovery) Unregister(instance registry.ServiceInstance) error {
	err := e.eurekaConnection.DeregisterInstance(e.toDeregisterInstance(instance))
	if err != nil {
		return perrors.WithMessage(err, "Could not unregister the instance. "+instance.GetServiceName())
	}
	return nil
}

func (e *eurekaServiceDiscovery) GetDefaultPageSize() int {
	return registry.DefaultPageSize
}

func (e *eurekaServiceDiscovery) GetServices() *gxset.HashSet {
	res := gxset.NewSet()
	apps, err := e.eurekaConnection.GetApps()
	if err != nil {
		logger.Errorf("GetServices getApps fail error: %s", err)
		return res
	}
	for _, application := range apps {
		if len(application.Instances) == 0 {
			continue
		}
		res.Add(strings.ToLower(application.Name))
	}

	return res
}

func parseMetadata(meta *fargo.InstanceMetadata) map[string]string {
	returnMap := make(map[string]string, len(meta.GetMap()))
	for key := range meta.GetMap() {
		s, err := meta.GetString(key)
		if err != nil {
			logger.Errorf("Could not parse to string key: " + key)
			continue
		}
		returnMap[key] = s
	}
	return returnMap
}

func (e *eurekaServiceDiscovery) GetInstances(serviceName string) []registry.ServiceInstance {
	application, err := e.eurekaConnection.GetApp(serviceName)
	if err != nil {
		logger.Errorf("Could not query the instances for service: " + serviceName + ", group: " + e.group)
		return make([]registry.ServiceInstance, 0, 0)
	}

	res := make([]registry.ServiceInstance, 0, len(application.Instances))

	for _, instance := range application.Instances {
		d := &registry.DefaultServiceInstance{
			Id:          instance.Id(),
			ServiceName: instance.App,
			Host:        instance.IPAddr,
			Enable:      instance.PortEnabled,
			Healthy:     true,
			Metadata:    parseMetadata(&instance.Metadata),
		}
		if fargo.UP != instance.Status {
			d.Healthy = false
		}
		if instance.SecurePortEnabled {
			d.Port = instance.SecurePort
		} else {
			d.Port = instance.Port
		}

		res = append(res, d)
	}

	return res
}

func (e *eurekaServiceDiscovery) GetInstancesByPage(serviceName string, offset int, pageSize int) gxpage.Pager {
	all := e.GetInstances(serviceName)
	res := make([]interface{}, 0, pageSize)
	// could not use res = all[a:b] here because the res should be []interface{}, not []ServiceInstance
	for i := offset; i < len(all) && i < offset+pageSize; i++ {
		res = append(res, all[i])
	}
	return gxpage.New(offset, pageSize, res, len(all))
}

func (e *eurekaServiceDiscovery) GetHealthyInstancesByPage(serviceName string, offset int, pageSize int, healthy bool) gxpage.Pager {
	all := e.GetInstances(serviceName)
	res := make([]interface{}, 0, pageSize)

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

func (e *eurekaServiceDiscovery) GetRequestInstances(serviceNames []string, offset int, requestedSize int) map[string]gxpage.Pager {
	res := make(map[string]gxpage.Pager, len(serviceNames))
	for _, name := range serviceNames {
		res[name] = e.GetInstancesByPage(name, offset, requestedSize)
	}
	return res
}

func (e *eurekaServiceDiscovery) AddListener(listener *registry.ServiceInstancesChangedListener) error {
	return nil
}

func (e *eurekaServiceDiscovery) DispatchEventByServiceName(serviceName string) error {
	return e.DispatchEventForInstances(serviceName, e.GetInstances(serviceName))
}

func (e *eurekaServiceDiscovery) DispatchEventForInstances(serviceName string, instances []registry.ServiceInstance) error {
	return e.DispatchEvent(registry.NewServiceInstancesChangedEvent(serviceName, instances))
}

func (e *eurekaServiceDiscovery) DispatchEvent(event *registry.ServiceInstancesChangedEvent) error {
	return nil
}

// toDeregisterInstance will create new service discovery instance
func newEurekaServiceDiscovery(url *common.URL) (registry.ServiceDiscovery, error) {
	addresses := strings.Split(url.Location, ",")
	serviceURLs := make([]string, 0, len(addresses))
	for _, addr := range addresses {
		ip, portStr, err := net.SplitHostPort(addr)
		if err != nil {
			return nil, perrors.WithMessagef(err, "split [%s] ", addr)
		}
		serviceURLs = append(serviceURLs, fmt.Sprintf("http://%s:%s/eureka/v2", ip, portStr))
	}
	config := fargo.Config{}
	config.Eureka.ServiceUrls = serviceURLs
	eurekaConnection := fargo.NewConnFromConfig(config)
	return &eurekaServiceDiscovery{
		eurekaConnection: &eurekaConnection,
		URL:              url,
	}, nil
}
