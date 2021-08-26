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
	"github.com/creasty/defaults"
)

import (
	_ "dubbo.apache.org/dubbo-go/v3/cluster/router/chain"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
)

// RouterConfig is the configuration of the router.
type RouterConfig struct {
	// Scope must be chosen from `service` and `application`.
	Scope string `validate:"required" yaml:"scope" json:"scope,omitempty" property:"scope"`
	// Key specifies which service or application the rule body acts on.
	Key        string   `validate:"required" yaml:"key" json:"key,omitempty" property:"key"`
	Force      bool     `default:"false" yaml:"force" json:"force,omitempty" property:"force"`
	Runtime    bool     `default:"false" yaml:"runtime" json:"runtime,omitempty" property:"runtime"`
	Enable     bool     `default:"true" yaml:"enable" json:"enable,omitempty" property:"enable"`
	Valid      bool     `default:"true" yaml:"valid" json:"valid,omitempty" property:"valid"`
	Priority   int      `default:"0" yaml:"priority" json:"priority,omitempty" property:"priority"`
	Conditions []string `yaml:"conditions" json:"conditions,omitempty" property:"conditions"`
	Tags       []Tag    `yaml:"tags" json:"tags,omitempty" property:"tags"`
}

type Tag struct {
	Name      string   `yaml:"name" json:"name,omitempty" property:"name"`
	Addresses []string `yaml:"addresses" json:"addresses,omitempty" property:"addresses"`
}

// Prefix dubbo.router
func (RouterConfig) Prefix() string {
	return constant.RouterConfigPrefix
}

func (c *RouterConfig) check() error {
	if err := defaults.Set(c); err != nil {
		return err
	}
	return verify(c)
}

func initRouterConfig(rc *RootConfig) error {
	routers := rc.Router
	if len(routers) > 0 {
		for _, r := range routers {
			if err := r.check(); err != nil {
				return err
			}
		}
		rc.Router = routers
	}

	//chain.SetVSAndDRConfigByte(vsBytes, drBytes)
	return nil
}

//// LocalRouterRules defines the local router config structure
//type LocalRouterRules struct {
//	RouterRules []interface{} `yaml:"routerRules"`
//}
//
//// RouterInit Set config file to init router config
//func RouterInit(vsConfigPath, drConfigPath string) error {
//	vsBytes, err := yaml.LoadYMLConfig(vsConfigPath)
//	if err != nil {
//		return err
//	}
//	drBytes, err := yaml.LoadYMLConfig(drConfigPath)
//	if err != nil {
//		return err
//	}
//	chain.SetVSAndDRConfigByte(vsBytes, drBytes)
//	return nil
//}
