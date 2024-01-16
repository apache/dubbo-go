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

// todo(DMwangnima): finish refactoring and move related logic from config package to this file.
// This RegistryConfig is a copy of /config/RegistryConfig right now.
// Please refer to issue(https://github.com/apache/dubbo-go/issues/2377).

// RegistryConfig is the configuration of the registry center
type RegistryConfig struct {
	Protocol          string            `validate:"required" yaml:"protocol"  json:"protocol,omitempty" property:"protocol"`
	Timeout           string            `default:"5s" validate:"required" yaml:"timeout" json:"timeout,omitempty" property:"timeout"` // unit: second
	Group             string            `yaml:"group" json:"group,omitempty" property:"group"`
	Namespace         string            `yaml:"namespace" json:"namespace,omitempty" property:"namespace"`
	TTL               string            `default:"15m" yaml:"ttl" json:"ttl,omitempty" property:"ttl"` // unit: minute
	Address           string            `validate:"required" yaml:"address" json:"address,omitempty" property:"address"`
	Username          string            `yaml:"username" json:"username,omitempty" property:"username"`
	Password          string            `yaml:"password" json:"password,omitempty"  property:"password"`
	Simplified        bool              `yaml:"simplified" json:"simplified,omitempty"  property:"simplified"`
	Preferred         bool              `yaml:"preferred" json:"preferred,omitempty" property:"preferred"` // Always use this registry first if set to true, useful when subscribe to multiple registriesConfig
	Zone              string            `yaml:"zone" json:"zone,omitempty" property:"zone"`                // The region where the registry belongs, usually used to isolate traffics
	Weight            int64             `yaml:"weight" json:"weight,omitempty" property:"weight"`          // Affects traffic distribution among registriesConfig, useful when subscribe to multiple registriesConfig Take effect only when no preferred registry is specified.
	Params            map[string]string `yaml:"params" json:"params,omitempty" property:"params"`
	RegistryType      string            `yaml:"registry-type"`
	UseAsMetaReport   bool              `default:"true" yaml:"use-as-meta-report" json:"use-as-meta-report,omitempty" property:"use-as-meta-report"`
	UseAsConfigCenter bool              `default:"true" yaml:"use-as-config-center" json:"use-as-config-center,omitempty" property:"use-as-config-center"`
}

func DefaultRegistryConfig() *RegistryConfig {
	return &RegistryConfig{UseAsMetaReport: true, UseAsConfigCenter: true, Timeout: "5s", TTL: "15m"}
}
