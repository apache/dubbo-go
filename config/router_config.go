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
	"dubbo.apache.org/dubbo-go/v3/cluster/router/chain"
	"dubbo.apache.org/dubbo-go/v3/common/yaml"
)

// LocalRouterRules defines the local router config structure
type LocalRouterRules struct {
	RouterRules []interface{} `yaml:"routerRules"`
}

// RouterInit Set config file to init router config
func RouterInit(vsConfigPath, drConfigPath string) error {
	vsBytes, err := yaml.LoadYMLConfig(vsConfigPath)
	if err != nil {
		return err
	}
	drBytes, err := yaml.LoadYMLConfig(drConfigPath)
	if err != nil {
		return err
	}
	chain.SetVSAndDRConfigByte(vsBytes, drBytes)
	return nil
}
