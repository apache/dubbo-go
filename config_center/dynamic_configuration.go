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

package config_center

import (
	gxset "github.com/dubbogo/gost/container/set"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/config_center/parser"
)

const (
	DefaultGroup         = "dubbo"
	DefaultConfigTimeout = "10s"
)

// DynamicConfiguration is the interface which modifys listener and gets properties file.
type DynamicConfiguration interface {
	Parser() parser.ConfigurationParser
	SetParser(parser.ConfigurationParser)
	AddListener(string, ConfigurationListener, ...Option)
	RemoveListener(string, ConfigurationListener, ...Option)
	// GetProperties get properties file
	GetProperties(string, ...Option) (string, error)

	// GetRule get Router rule properties file
	GetRule(string, ...Option) (string, error)

	// GetInternalProperty get value by key in Default properties file(dubbo.properties)
	GetInternalProperty(string, ...Option) (string, error)

	// PublishConfig will publish the config with the (key, group, value) pair
	// for zk: path is /$(group)/config/$(key) -> value
	// for nacos: group, key -> value
	PublishConfig(string, string, string) error

	// RemoveConfig will remove the config white the (key, group) pair
	RemoveConfig(string, string) error

	// GetConfigKeysByGroup will return all keys with the group
	GetConfigKeysByGroup(group string) (*gxset.HashSet, error)
}

// GetRuleKey The format is '{interfaceName}:[version]:[group]'
func GetRuleKey(url *common.URL) string {
	return url.ColonSeparatedKey()
}
