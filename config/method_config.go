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
	"fmt"
	"strconv"
)

import (
	"github.com/creasty/defaults"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/common/extension"
)

// MethodConfig defines method config
type MethodConfig struct {
	InterfaceId                 string
	InterfaceName               string
	Name                        string `yaml:"name"  json:"name,omitempty" property:"name"`
	Retries                     string `yaml:"retries"  json:"retries,omitempty" property:"retries"`
	LoadBalance                 string `yaml:"loadbalance"  json:"loadbalance,omitempty" property:"loadbalance"`
	Weight                      int64  `yaml:"weight"  json:"weight,omitempty" property:"weight"`
	TpsLimitInterval            string `yaml:"tps.limit.interval" json:"tps.limit.interval,omitempty" property:"tps.limit.interval"`
	TpsLimitRate                string `yaml:"tps.limit.rate" json:"tps.limit.rate,omitempty" property:"tps.limit.rate"`
	TpsLimitStrategy            string `yaml:"tps.limit.strategy" json:"tps.limit.strategy,omitempty" property:"tps.limit.strategy"`
	ExecuteLimit                string `yaml:"execute.limit" json:"execute.limit,omitempty" property:"execute.limit"`
	ExecuteLimitRejectedHandler string `yaml:"execute.limit.rejected.handler" json:"execute.limit.rejected.handler,omitempty" property:"execute.limit.rejected.handler"`
	Sticky                      bool   `yaml:"sticky"   json:"sticky,omitempty" property:"sticky"`
	RequestTimeout              string `yaml:"timeout"  json:"timeout,omitempty" property:"timeout"`
}

// nolint
func (m *MethodConfig) Prefix() string {
	if len(m.InterfaceId) != 0 {
		return constant.Dubbo + "." + m.InterfaceName + "." + m.InterfaceId + "." + m.Name + "."
	}

	return constant.Dubbo + "." + m.InterfaceName + "." + m.Name + "."
}

func (m *MethodConfig) Init() error {
	return m.check()
}

func initProviderMethodConfig(sc *ServiceConfig) error {
	methods := sc.Methods
	if methods == nil {
		return nil
	}
	for _, method := range methods {
		if err := method.check(); err != nil {
			return err
		}
	}
	sc.Methods = methods
	return nil
}

// check set default value and verify
func (m *MethodConfig) check() error {
	qualifieldMethodName := m.InterfaceName + "#" + m.Name
	if m.TpsLimitStrategy != "" {
		_, err := extension.GetTpsLimitStrategyCreator(m.TpsLimitStrategy)
		if err != nil {
			panic(err)
		}
	}

	if m.TpsLimitInterval != "" {
		tpsLimitInterval, err := strconv.ParseInt(m.TpsLimitInterval, 0, 0)
		if err != nil {
			return fmt.Errorf("[MethodConfig] Cannot parse the configuration tps.limit.interval for method %s, please check your configuration", qualifieldMethodName)
		}
		if tpsLimitInterval < 0 {
			return fmt.Errorf("[MethodConfig] The configuration tps.limit.interval for %s must be positive, please check your configuration", qualifieldMethodName)
		}
	}

	if m.TpsLimitRate != "" {
		tpsLimitRate, err := strconv.ParseInt(m.TpsLimitRate, 0, 0)
		if err != nil {
			return fmt.Errorf("[MethodConfig] Cannot parse the configuration tps.limit.rate for method %s, please check your configuration", qualifieldMethodName)
		}
		if tpsLimitRate < 0 {
			return fmt.Errorf("[MethodConfig] The configuration tps.limit.rate for method %s must be positive, please check your configuration", qualifieldMethodName)
		}
	}

	if err := defaults.Set(m); err != nil {
		return err
	}
	return verify(m)
}
