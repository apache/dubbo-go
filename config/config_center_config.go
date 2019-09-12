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

package config

import (
	"context"
	"time"
)

import (
	"github.com/creasty/defaults"
)

type ConfigCenterConfig struct {
	context       context.Context
	Protocol      string `required:"true"  yaml:"protocol"  json:"protocol,omitempty"`
	Address       string `yaml:"address" json:"address,omitempty"`
	Cluster       string `yaml:"cluster" json:"cluster,omitempty"`
	Group         string `default:"dubbo" yaml:"group" json:"group,omitempty"`
	Username      string `yaml:"username" json:"username,omitempty"`
	Password      string `yaml:"password" json:"password,omitempty"`
	ConfigFile    string `default:"dubbo.properties" yaml:"config_file"  json:"config_file,omitempty"`
	AppConfigFile string `default:"dubbo.properties" yaml:"app_config_file"  json:"app_config_file,omitempty"`
	TimeoutStr    string `yaml:"timeout"  json:"timeout,omitempty"`
	timeout       time.Duration
}

func (c *ConfigCenterConfig) UnmarshalYAML(unmarshal func(interface{}) error) error {
	if err := defaults.Set(c); err != nil {
		return err
	}
	type plain ConfigCenterConfig
	if err := unmarshal((*plain)(c)); err != nil {
		return err
	}
	return nil
}
