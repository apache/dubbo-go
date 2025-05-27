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

type TripleConfig struct {
	// TODO: remove test
	Test              string `yaml:"test" json:"test,omitempty" property:"test"`
	KeepAliveInterval string `yaml:"keep-alive-interval" json:"keep-alive-interval,omitempty" property:"keep-alive-interval"`
	KeepAliveTimeout  string `yaml:"keep-alive-timeout" json:"keep-alive-timeout,omitempty" property:"keep-alive-timeout"`
}

func DefaultTripleConfig() *TripleConfig {
	return &TripleConfig{}
}

// Clone a new TripleConfig
func (t *TripleConfig) Clone() *TripleConfig {
	if t == nil {
		return nil
	}

	return &TripleConfig{
		Test: t.Test,
		KeepAliveInterval: t.KeepAliveInterval,
		KeepAliveTimeout: t.KeepAliveTimeout,
	}
}
