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
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/config_center"
)

const (
	// DefaultKey for default Configurator
	DefaultKey = "default"
)

type getConfiguratorFunc func(url *common.URL) config_center.Configurator

var configurator = make(map[string]getConfiguratorFunc)

// SetConfigurator sets the getConfiguratorFunc with @name
func SetConfigurator(name string, v getConfiguratorFunc) {
	configurator[name] = v
}

// GetConfigurator finds the Configurator with @name
func GetConfigurator(name string, url *common.URL) config_center.Configurator {
	if configurator[name] == nil {
		panic("configurator for " + name + " is not existing, make sure you have import the package.")
	}
	return configurator[name](url)
}

// SetDefaultConfigurator sets the default Configurator
func SetDefaultConfigurator(v getConfiguratorFunc) {
	configurator[DefaultKey] = v
}

// GetDefaultConfigurator gets default configurator
func GetDefaultConfigurator(url *common.URL) config_center.Configurator {
	if configurator[DefaultKey] == nil {
		panic("configurator for default is not existing, make sure you have import the package.")
	}
	return configurator[DefaultKey](url)
}

// GetDefaultConfiguratorFunc gets default configurator function
func GetDefaultConfiguratorFunc() getConfiguratorFunc {
	if configurator[DefaultKey] == nil {
		panic("configurator for default is not existing, make sure you have import the package.")
	}
	return configurator[DefaultKey]
}
