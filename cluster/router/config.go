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

package router

type AffinityAware struct {
	Key   string `default:"" yaml:"key" json:"key,omitempty" property:"key"`
	Ratio int32  `default:"0" yaml:"ratio" json:"ratio,omitempty" property:"ratio"`
}

// AffinityRouter -- RouteConfigVersion == v3.1
type AffinityRouter struct {
	Scope         string        `validate:"required" yaml:"scope" json:"scope,omitempty" property:"scope"` // must be chosen from `service` and `application`.
	Key           string        `validate:"required" yaml:"key" json:"key,omitempty" property:"key"`       // specifies which service or application the rule body acts on.
	Runtime       bool          `default:"false" yaml:"runtime" json:"runtime,omitempty" property:"runtime"`
	Enabled       bool          `default:"true" yaml:"enabled" json:"enabled,omitempty" property:"enabled"`
	AffinityAware AffinityAware `yaml:"affinityAware" json:"affinityAware,omitempty" property:"affinityAware"`
}
