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

package filesystem

import (
	"bytes"
	"github.com/dubbogo/gost/container/set"
	"github.com/dubbogo/gost/page"
	perrors "github.com/pkg/errors"
	"os"
	"os/exec"
	"os/user"
	"runtime"
	"strings"
)

import (
	"github.com/apache/dubbo-go/common"
	"github.com/apache/dubbo-go/common/extension"
	"github.com/apache/dubbo-go/registry"
)

const (
	name = "file-system"
	paramNamePrefix = "dubbo.config-center."
	configCenterDirParamName = paramNamePrefix + "dir"
	configCenterEncodingParamName = paramNamePrefix + "encoding"
	defaultConfigCenterEncoding = "UTF-8"
)

func init() {
	extension.SetServiceDiscovery(name, newFileSystemServiceDiscovery)
}

func newFileSystemServiceDiscovery(url *common.URL) (registry.ServiceDiscovery, error){
	instance := &FilleSystemServiceDiscovery{
		fileLocksCache: make(map[*File]*Filelock, 4),
		listeners: make([]*registry.ServiceInstancesChangedListener, 0, 2),
	}
	return instance,nil
}

func createFile(url *common.URL) (*File, string, error) {
	userHome, _ := home()
	path := userHome + "/" + ".dubbo" + "/" + "registry"
	url.AddParam(configCenterDirParamName, path)
	file, error := initDirectory(url, userHome)
	if error != nil {
		return nil, "", error
	}
	encoding :=url.GetParam(configCenterEncodingParamName, defaultConfigCenterEncoding)
	return file, encoding, nil
}

func initDirectory(url *common.URL, userHome string) (*File, error) {
	directoryPath := url.GetParam(configCenterDirParamName, url.Path)
	var rootDirectory string
	if directoryPath == "" || len(directoryPath) == 0 {
		rootDirectory = string(os.PathSeparator) + directoryPath
	}

	if directoryPath == "" {
		if result, _ := exists(directoryPath); !result {
			// If the directory does not exist
			rootDirectory = userHome + "/" + ".dubbo" + "/" + "config-center"
		}
	}

	if result,_ := exists(rootDirectory); !result {
		if _, err :=os.Create(rootDirectory); err != nil {
			return nil, perrors.WithStack(err)
		}
	}
	return &File{Path: rootDirectory}, nil
}

func exists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil { return true, nil }
	if os.IsNotExist(err) { return false, nil }
	return true, err
}

// FilleSystemServiceDiscovery is an implementation based on memory.
// Usually you will not use this implementation except for tests.
type FilleSystemServiceDiscovery struct {
	fileLocksCache map[*File]*Filelock
	listeners      []*registry.ServiceInstancesChangedListener
	encoding       string
	rootDirectory  *File
}

func (i *FilleSystemServiceDiscovery) String() string {
	return name
}

// Destroy doesn't destroy the instance, it just clear the instances
func (i *FilleSystemServiceDiscovery) Destroy() error {
	// reset to empty
	i.fileLocksCache = nil
	i.listeners = nil
	return nil
}

// Register will store the instance using its id as key
func (i *FilleSystemServiceDiscovery) Register(instance registry.ServiceInstance) error {
	return nil
}

// Update will act like register
func (i *FilleSystemServiceDiscovery) Update(instance registry.ServiceInstance) error {
	return i.Register(instance)
}

// Unregister will remove the instance
func (i *FilleSystemServiceDiscovery) Unregister(instance registry.ServiceInstance) error {
	delete(i.fileLocksCache,  i.rootDirectory)
	return nil
}

// GetDefaultPageSize will return the default page size
func (i *FilleSystemServiceDiscovery) GetDefaultPageSize() int {
	return registry.DefaultPageSize
}

// GetServices will return all service names
func (i *FilleSystemServiceDiscovery) GetServices() *gxset.HashSet {
	result := gxset.NewSet()
	return result
}

// GetInstances will find out all instances with serviceName
func (i *FilleSystemServiceDiscovery) GetInstances(serviceName string) []registry.ServiceInstance {
	return nil
}

// GetInstancesByPage will return the part of instances
func (i *FilleSystemServiceDiscovery) GetInstancesByPage(serviceName string, offset int, pageSize int) gxpage.Pager {
	instances := i.GetInstances(serviceName)
	// we can not use []registry.ServiceInstance since New(...) received []interface{} as parameter
	result := make([]interface{}, 0, pageSize)
	for i := offset; i < len(instances) && i < offset+pageSize; i++ {
		result = append(result, instances[i])
	}
	return gxpage.New(offset, pageSize, result, len(instances))
}

// GetHealthyInstancesByPage will return the instances
func (i *FilleSystemServiceDiscovery) GetHealthyInstancesByPage(serviceName string, offset int, pageSize int, healthy bool) gxpage.Pager {
	instances := i.GetInstances(serviceName)
	// we can not use []registry.ServiceInstance since New(...) received []interface{} as parameter
	result := make([]interface{}, 0, pageSize)
	count := 0
	for i := offset; i < len(instances) && count < pageSize; i++ {
		if instances[i].IsHealthy() == healthy {
			result = append(result, instances[i])
			count++
		}
	}
	return gxpage.New(offset, pageSize, result, len(instances))
}

// GetRequestInstances will iterate the serviceName and aggregate them
func (i *FilleSystemServiceDiscovery) GetRequestInstances(serviceNames []string, offset int, requestedSize int) map[string]gxpage.Pager {
	res := make(map[string]gxpage.Pager, len(serviceNames))
	for _, name := range serviceNames {
		res[name] = i.GetInstancesByPage(name, offset, requestedSize)
	}
	return res
}

// AddListener will save the listener inside the memory
func (i *FilleSystemServiceDiscovery) AddListener(listener *registry.ServiceInstancesChangedListener) error {
	i.listeners = append(i.listeners, listener)
	return nil
}

// DispatchEventByServiceName will do nothing
func (i *FilleSystemServiceDiscovery) DispatchEventByServiceName(serviceName string) error {
	return nil
}

// DispatchEventForInstances will do nothing
func (i *FilleSystemServiceDiscovery) DispatchEventForInstances(serviceName string, instances []registry.ServiceInstance) error {
	return nil
}

// DispatchEvent will do nothing
func (i *FilleSystemServiceDiscovery) DispatchEvent(event *registry.ServiceInstancesChangedEvent) error {
	return nil
}

func home() (string, error) {
	user, err := user.Current()
	if nil == err {
		return user.HomeDir, nil
	}

	// cross compile support
	if "windows" == runtime.GOOS {
		return homeWindows()
	}

	// Unix-like system, so just assume Unix
	return homeUnix()
}

func homeUnix() (string, error) {
	// First prefer the HOME environmental variable
	if home := os.Getenv("HOME"); home != "" {
		return home, nil
	}

	// If that fails, try the shell
	var stdout bytes.Buffer
	cmd := exec.Command("sh", "-c", "eval echo ~$USER")
	cmd.Stdout = &stdout
	if err := cmd.Run(); err != nil {
		return "", err
	}

	result := strings.TrimSpace(stdout.String())
	if result == "" {
		return "", perrors.Errorf("blank output when reading home directory")
	}

	return result, nil
}

func homeWindows() (string, error) {
	drive := os.Getenv("HOMEDRIVE")
	path := os.Getenv("HOMEPATH")
	home := drive + path
	if drive == "" || path == "" {
		home = os.Getenv("USERPROFILE")
	}
	if home == "" {
		return "", perrors.Errorf("HOMEDRIVE, HOMEPATH, and USERPROFILE are blank")
	}

	return home, nil
}
