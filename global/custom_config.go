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

// CustomConfig
//
// # Experimental
//
// Notice: This struct is EXPERIMENTAL and may be changed or removed in a
// later release.
type CustomConfig struct {
	ConfigMap map[string]interface{} `yaml:"config-map" json:"config-map,omitempty" property:"config-map"`
}

func DefaultCustomConfig() *CustomConfig {
	return &CustomConfig{
		ConfigMap: make(map[string]interface{}),
	}
}

// Clone a new CustomConfig
func (c *CustomConfig) Clone() *CustomConfig {
	if c == nil {
		return nil
	}

	newConfigMap := make(map[string]interface{}, len(c.ConfigMap))
	for k, v := range c.ConfigMap {
		newConfigMap[k] = v
	}

	return &CustomConfig{
		ConfigMap: newConfigMap,
	}
}
