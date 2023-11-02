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

package extension

import (
	"dubbo.apache.org/dubbo-go/v3/cluster/directory"
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/registry"
	"github.com/dubbogo/gost/log/logger"
)

type registryDirectory func(url *common.URL, registry registry.Registry) (directory.Directory, error)

var directories = make(map[string]registryDirectory)
var defaultDirectory registryDirectory

// SetDefaultRegistryDirectory sets the default registryDirectory
func SetDefaultRegistryDirectory(v registryDirectory) {
	defaultDirectory = v
}

// SetDirectory sets the default registryDirectory
func SetDirectory(key string, v registryDirectory) {
	directories[key] = v
}

// GetDefaultRegistryDirectory finds the registryDirectory with url and registry
func GetDefaultRegistryDirectory(config *common.URL, registry registry.Registry) (directory.Directory, error) {
	if defaultDirectory == nil {
		panic("registry directory is not existing, make sure you have import the package.")
	}
	return defaultDirectory(config, registry)
}

// GetDirectoryInstance finds the registryDirectory with url and registry
func GetDirectoryInstance(config *common.URL, registry registry.Registry) (directory.Directory, error) {
	key := config.Protocol
	if key == "" {
		return GetDefaultRegistryDirectory(config, registry)
	}
	if directories[key] == nil {
		logger.Warn("registry directory " + key + " does not exist, make sure you have import the package, will use the default directory type.")
		return GetDefaultRegistryDirectory(config, registry)
	}
	return directories[key](config, registry)
}
