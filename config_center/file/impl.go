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
	"github.com/apache/dubbo-go/common"
	"github.com/apache/dubbo-go/common/constant"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

import (
	"github.com/apache/dubbo-go/config_center"
	"github.com/apache/dubbo-go/config_center/parser"
)

import (
	gxset "github.com/dubbogo/gost/container/set"
)

type fileSystemDynamicConfiguration struct {
	url           *common.URL
	rootDirectory os.File // the root directory for config center
	keyListeners  sync.Map
	encoding      string
	rootPath      string
}

func newFileSystemDynamicConfiguration(url *common.URL) (*fileSystemDynamicConfiguration, error) {
	c := &fileSystemDynamicConfiguration{
		url:      url,
		rootPath: "/" + url.GetParam(constant.CONFIG_NAMESPACE_KEY, config_center.DEFAULT_GROUP) + "/config",
	}

	return c, nil
}

func (fsdc *fileSystemDynamicConfiguration) Parser() parser.ConfigurationParser {
	return nil
}
func (fsdc *fileSystemDynamicConfiguration) SetParser(parser.ConfigurationParser) {

}
func (fsdc *fileSystemDynamicConfiguration) AddListener(key string, listener config_center.ConfigurationListener, opions ...config_center.Option) {
	fsdc.addListener(key, listener)
}
func (fsdc *fileSystemDynamicConfiguration) RemoveListener(key string, listener config_center.ConfigurationListener, opions ...config_center.Option) {

}

// GetProperties get properties file
func (fsdc *fileSystemDynamicConfiguration) GetProperties(string, ...config_center.Option) (string, error) {
	return "", nil
}

// GetRule get Router rule properties file
func (fsdc *fileSystemDynamicConfiguration) GetRule(string, ...config_center.Option) (string, error) {
	return "", nil
}

// GetInternalProperty get value by key in Default properties file(dubbo.properties)
func (fsdc *fileSystemDynamicConfiguration) GetInternalProperty(string, ...config_center.Option) (string, error) {
	return "", nil
}

// PublishConfig will publish the config with the (key, group, value) pair
func (fsdc *fileSystemDynamicConfiguration) PublishConfig(key string, group string, value string) error {
	path := fsdc.buildPath(key, group)
	return write(path, value)
}

func (fsdc *fileSystemDynamicConfiguration) buildPath(key string, group string) string {
	return strings.Join([]string{fsdc.rootPath, group, key}, "/")
}

func write(path string, value string) error {
	return nil
}

// GetConfigKeysByGroup will return all keys with the group
func (fsdc *fileSystemDynamicConfiguration) GetConfigKeysByGroup(group string) (*gxset.HashSet, error) {
	return nil, nil
}

// RemoveConfig will remove the config whit hte (key, group)
func (fsdc *fileSystemDynamicConfiguration) RemoveConfig(string, string) error {
	return nil
}

func (fsdc *fileSystemDynamicConfiguration) GetConfigGroups() *gxset.HashSet {
	var cg []string
	filepath.Walk(fsdc.rootDirectory.Name(), func(path string, info os.FileInfo, err error) error {
		if info.IsDir() {
			cg = append(cg, info.Name())
		}

		return nil
	})

	return gxset.NewSet(cg)
}

func (fsdc *fileSystemDynamicConfiguration) getRootDirectory() os.File {
	return fsdc.rootDirectory
}

func (fsdc *fileSystemDynamicConfiguration) ConfigFile(key string, group string) *os.File {
	return nil
}
