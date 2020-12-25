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

package file

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"strconv"
)

import (
	gxset "github.com/dubbogo/gost/container/set"
	gxpage "github.com/dubbogo/gost/page"
	perrors "github.com/pkg/errors"
)

import (
	"github.com/apache/dubbo-go/common"
	"github.com/apache/dubbo-go/common/constant"
	"github.com/apache/dubbo-go/common/extension"
	"github.com/apache/dubbo-go/common/logger"
	"github.com/apache/dubbo-go/config"
	"github.com/apache/dubbo-go/config_center"
	"github.com/apache/dubbo-go/config_center/file"
	"github.com/apache/dubbo-go/registry"
)

// init will put the service discovery into extension
func init() {
	extension.SetServiceDiscovery(constant.FILE_KEY, newFileSystemServiceDiscovery)
}

// fileServiceDiscovery is the implementation of service discovery based on file.
type fileSystemServiceDiscovery struct {
	dynamicConfiguration file.FileSystemDynamicConfiguration
	rootPath             string
	fileMap              map[string]string
}

func newFileSystemServiceDiscovery(name string) (registry.ServiceDiscovery, error) {
	sdc, ok := config.GetBaseConfig().GetServiceDiscoveries(name)
	if !ok || sdc.Protocol != constant.FILE_KEY {
		return nil, perrors.New("could not init the instance because the config is invalid")
	}

	rp, err := file.Home()
	if err != nil {
		return nil, perrors.WithStack(err)
	}

	fdcf := extension.GetConfigCenterFactory(constant.FILE_KEY)
	p := path.Join(rp, ".dubbo", constant.REGISTRY_KEY)
	url, _ := common.NewURL("")
	url.AddParamAvoidNil(file.CONFIG_CENTER_DIR_PARAM_NAME, p)
	c, err := fdcf.GetDynamicConfiguration(url)
	if err != nil {
		return nil, perrors.WithStack(err)
	}

	sd := &fileSystemServiceDiscovery{
		dynamicConfiguration: *c.(*file.FileSystemDynamicConfiguration),
		rootPath:             p,
		fileMap:              make(map[string]string),
	}

	extension.AddCustomShutdownCallback(func() {
		sd.Destroy()
	})

	for _, v := range sd.GetServices().Values() {
		for _, i := range sd.GetInstances(v.(string)) {
			// like java do nothing
			l := &RegistryConfigurationListener{}
			sd.dynamicConfiguration.AddListener(getServiceInstanceId(i), l, config_center.WithGroup(getServiceName(i)))
		}
	}

	return sd, nil
}

// nolint
func (fssd *fileSystemServiceDiscovery) String() string {
	return fmt.Sprintf("file-system-service-discovery")
}

// Destroy will destroy the service discovery.
// If the discovery cannot be destroy, it will return an error.
func (fssd *fileSystemServiceDiscovery) Destroy() error {
	fssd.dynamicConfiguration.Close()

	for _, f := range fssd.fileMap {
		fssd.releaseAndRemoveRegistrationFiles(f)
	}

	return nil
}

// nolint
func (fssd *fileSystemServiceDiscovery) releaseAndRemoveRegistrationFiles(file string) {
	os.RemoveAll(file)
}

// ----------------- registration ----------------

// Register will register an instance of ServiceInstance to registry
func (fssd *fileSystemServiceDiscovery) Register(instance registry.ServiceInstance) error {
	id := getServiceInstanceId(instance)
	sn := getServiceName(instance)

	c, err := toJsonString(instance)
	if err != nil {
		return perrors.WithStack(err)
	}

	err = fssd.dynamicConfiguration.PublishConfig(id, sn, c)
	if err != nil {
		return perrors.WithStack(err)
	}

	fssd.fileMap[id] = fssd.dynamicConfiguration.GetPath(id, sn)

	return nil
}

// nolint
func getServiceInstanceId(si registry.ServiceInstance) string {
	if si.GetId() == "" {
		return si.GetHost() + "." + strconv.Itoa(si.GetPort())
	}

	return si.GetId()
}

// nolint
func getServiceName(si registry.ServiceInstance) string {
	return si.GetServiceName()
}

// toJsonString to json string
func toJsonString(si registry.ServiceInstance) (string, error) {
	bytes, err := json.Marshal(si)
	if err != nil {
		return "", perrors.WithStack(err)
	}

	return string(bytes), nil
}

// Update will update the data of the instance in registry
func (fssd *fileSystemServiceDiscovery) Update(instance registry.ServiceInstance) error {
	return fssd.Register(instance)
}

// Unregister will unregister this instance from registry
func (fssd *fileSystemServiceDiscovery) Unregister(instance registry.ServiceInstance) error {
	id := getServiceInstanceId(instance)
	sn := getServiceName(instance)

	err := fssd.dynamicConfiguration.RemoveConfig(id, sn)
	if err != nil {
		return perrors.WithStack(err)
	}

	delete(fssd.fileMap, instance.GetId())
	return nil
}

// ----------------- discovery -------------------
// GetDefaultPageSize will return the default page size
func (fssd *fileSystemServiceDiscovery) GetDefaultPageSize() int {
	return 100
}

// GetServices will return the all service names.
func (fssd *fileSystemServiceDiscovery) GetServices() *gxset.HashSet {
	r := gxset.NewSet()
	// dynamicConfiguration root path is the actual root path
	fileInfo, _ := ioutil.ReadDir(fssd.dynamicConfiguration.RootPath())

	for _, file := range fileInfo {
		if file.IsDir() {
			r.Add(file.Name())
		}
	}

	return r
}

// GetInstances will return all service instances with serviceName
func (fssd *fileSystemServiceDiscovery) GetInstances(serviceName string) []registry.ServiceInstance {
	set, err := fssd.dynamicConfiguration.GetConfigKeysByGroup(serviceName)
	if err != nil {
		logger.Errorf("[FileServiceDiscovery] Could not query the instances for service{%s}, error = err{%v} ",
			serviceName, err)
		return make([]registry.ServiceInstance, 0, 0)
	}

	res := make([]registry.ServiceInstance, 0, set.Size())
	for _, v := range set.Values() {
		id := v.(string)
		p, err := fssd.dynamicConfiguration.GetProperties(id, config_center.WithGroup(serviceName))
		if err != nil {
			logger.Errorf("[FileServiceDiscovery] Could not get the properties for id{%s}, service{%s}, "+
				"error = err{%v} ",
				id, serviceName, err)
			return make([]registry.ServiceInstance, 0, 0)
		}

		dsi := &registry.DefaultServiceInstance{}
		err = json.Unmarshal([]byte(p), dsi)
		if err != nil {
			logger.Errorf("[FileServiceDiscovery] Could not unmarshal the properties for id{%s}, service{%s}, "+
				"error = err{%v} ",
				id, serviceName, err)
			return make([]registry.ServiceInstance, 0, 0)
		}

		res = append(res, dsi)
	}

	return res
}

// GetInstancesByPage will return a page containing instances of ServiceInstance with the serviceName
// the page will start at offset
func (fssd *fileSystemServiceDiscovery) GetInstancesByPage(serviceName string, offset int, pageSize int) gxpage.Pager {
	return nil
}

// GetHealthyInstancesByPage will return a page containing instances of ServiceInstance.
// The param healthy indices that the instance should be healthy or not.
// The page will start at offset
func (fssd *fileSystemServiceDiscovery) GetHealthyInstancesByPage(serviceName string, offset int, pageSize int,
	healthy bool) gxpage.Pager {
	return nil
}

// Batch get all instances by the specified service names
func (fssd *fileSystemServiceDiscovery) GetRequestInstances(serviceNames []string, offset int,
	requestedSize int) map[string]gxpage.Pager {
	return nil
}

// ----------------- event ----------------------
// AddListener adds a new ServiceInstancesChangedListener
// client
func (fssd *fileSystemServiceDiscovery) AddListener(listener *registry.ServiceInstancesChangedListener) error {
	//fssd.dynamicConfiguration.AddListener(listener.ServiceName)
	return nil
}

// DispatchEventByServiceName dispatches the ServiceInstancesChangedEvent to service instance whose name is serviceName
func (fssd *fileSystemServiceDiscovery) DispatchEventByServiceName(serviceName string) error {
	return fssd.DispatchEvent(registry.NewServiceInstancesChangedEvent(serviceName, fssd.GetInstances(serviceName)))
}

// DispatchEventForInstances dispatches the ServiceInstancesChangedEvent to target instances
func (fssd *fileSystemServiceDiscovery) DispatchEventForInstances(serviceName string,
	instances []registry.ServiceInstance) error {
	return fssd.DispatchEvent(registry.NewServiceInstancesChangedEvent(serviceName, instances))
}

// DispatchEvent dispatches the event
func (fssd *fileSystemServiceDiscovery) DispatchEvent(event *registry.ServiceInstancesChangedEvent) error {
	extension.GetGlobalDispatcher().Dispatch(event)
	return nil
}
