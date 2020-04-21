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
	"github.com/apache/dubbo-go/common"
	"github.com/apache/dubbo-go/common/constant"
	"github.com/apache/dubbo-go/common/extension"
	"github.com/apache/dubbo-go/registry"
	perrors "github.com/apache/dubbo-go/vendor/github.com/pkg/errors"
	gxset "github.com/dubbogo/gost/container/set"
	gxpage "github.com/dubbogo/gost/page"
	"github.com/hudl/fargo"
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
	panic("implement me")
}

func (e *eurekaServiceDiscovery) Update(instance registry.ServiceInstance) error {
	panic("implement me")
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
	panic("implement me")
}

func (e *eurekaServiceDiscovery) GetServices() *gxset.HashSet {
	panic("implement me")
}

func (e *eurekaServiceDiscovery) GetInstances(serviceName string) []registry.ServiceInstance {
	panic("implement me")
}

func (e *eurekaServiceDiscovery) GetInstancesByPage(serviceName string, offset int, pageSize int) gxpage.Pager {
	panic("implement me")
}

func (e *eurekaServiceDiscovery) GetHealthyInstancesByPage(serviceName string, offset int, pageSize int, healthy bool) gxpage.Pager {
	panic("implement me")
}

func (e *eurekaServiceDiscovery) GetRequestInstances(serviceNames []string, offset int, requestedSize int) map[string]gxpage.Pager {
	panic("implement me")
}

func (e *eurekaServiceDiscovery) AddListener(listener *registry.ServiceInstancesChangedListener) error {
	panic("implement me")
}

func (e *eurekaServiceDiscovery) DispatchEventByServiceName(serviceName string) error {
	panic("implement me")
}

func (e *eurekaServiceDiscovery) DispatchEventForInstances(serviceName string, instances []registry.ServiceInstance) error {
	panic("implement me")
}

func (e *eurekaServiceDiscovery) DispatchEvent(event *registry.ServiceInstancesChangedEvent) error {
	panic("implement me")
}

// toDeregisterInstance will create new service discovery instance
func newEurekaServiceDiscovery(url *common.URL) (registry.ServiceDiscovery, error) {
	config := fargo.Config{}
	config.Eureka.ServerPort = 1999
	eurekaConnection := fargo.NewConnFromConfig(config)
	return &eurekaServiceDiscovery{
		eurekaConnection: &eurekaConnection,
		URL:              url,
		//group:            url.GetParam(constant.NACOS_GROUP, defaultGroup),
	}, nil
}
