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

package global

// CenterConfig is configuration for config center
//
// ConfigCenter also introduced concepts of namespace and group to better manage Key-Value pairs by group,
// those configs are already built-in in many professional third-party configuration centers.
// In most cases, namespace is used to isolate different tenants, while group is used to divide the key set from one tenant into groups.
//
// CenterConfig has currently supported Zookeeper, Nacos, Etcd, Consul, Apollo
type CenterConfig struct {
	Protocol  string            `validate:"required" yaml:"protocol"  json:"protocol,omitempty"`
	Address   string            `validate:"required" yaml:"address" json:"address,omitempty"`
	DataId    string            `yaml:"data-id" json:"data-id,omitempty"`
	Cluster   string            `yaml:"cluster" json:"cluster,omitempty"`
	Group     string            `yaml:"group" json:"group,omitempty"`
	Username  string            `yaml:"username" json:"username,omitempty"`
	Password  string            `yaml:"password" json:"password,omitempty"`
	Namespace string            `yaml:"namespace"  json:"namespace,omitempty"`
	AppID     string            `default:"dubbo" yaml:"app-id"  json:"app-id,omitempty"`
	Timeout   string            `default:"10s" yaml:"timeout"  json:"timeout,omitempty"`
	Params    map[string]string `yaml:"params"  json:"parameters,omitempty"`

	//FileExtension the suffix of config dataId, also the file extension of config content
	FileExtension string `default:"yaml" yaml:"file-extension" json:"file-extension" `
}

func DefaultCenterConfig() *CenterConfig {
	return &CenterConfig{
		Params: make(map[string]string),
	}
}

// Clone a new CenterConfig
func (c *CenterConfig) Clone() *CenterConfig {
	newParams := make(map[string]string, len(c.Params))
	for k, v := range c.Params {
		newParams[k] = v
	}

	return &CenterConfig{
		Protocol:      c.Protocol,
		Address:       c.Address,
		DataId:        c.DataId,
		Cluster:       c.Cluster,
		Group:         c.Group,
		Username:      c.Username,
		Password:      c.Password,
		Namespace:     c.Namespace,
		AppID:         c.AppID,
		Timeout:       c.Timeout,
		Params:        newParams,
		FileExtension: c.FileExtension,
	}
}
