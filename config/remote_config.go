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
	"time"
)

type RemoteConfig struct {
	Address string            `yaml:"address" json:"address,omitempty"`
	Timeout time.Duration     `default:"10s" yaml:"timeout" json:"timeout,omitempty"`
	Params  map[string]string `yaml:"params" json:"address,omitempty"`
}

// GetParam will return the value of the key. If not found, def will be return;
// def => default value
func (rc *RemoteConfig) GetParam(key string, def string) string {
	param, ok := rc.Params[key]
	if !ok {
		return def
	}
	return param
}
