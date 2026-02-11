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

import (
	"reflect"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
)

type RouterConfig struct {
	Scope      string   `validate:"required" yaml:"scope" json:"scope,omitempty" property:"scope"`
	Key        string   `validate:"required" yaml:"key" json:"key,omitempty" property:"key"`
	Force      *bool    `default:"false" yaml:"force" json:"force,omitempty" property:"force"`
	Runtime    *bool    `default:"false" yaml:"runtime" json:"runtime,omitempty" property:"runtime"`
	Enabled    *bool    `default:"true" yaml:"enabled" json:"enabled,omitempty" property:"enabled"`
	Valid      *bool    `default:"true" yaml:"valid" json:"valid,omitempty" property:"valid"`
	Priority   int      `default:"0" yaml:"priority" json:"priority,omitempty" property:"priority"`
	Conditions []string `yaml:"conditions" json:"conditions,omitempty" property:"conditions"`
	Tags       []Tag    `yaml:"tags" json:"tags,omitempty" property:"tags"`
	ScriptType string   `yaml:"type" json:"type,omitempty" property:"type"`
	Script     string   `yaml:"script" json:"script,omitempty" property:"script"`
}

type Tag struct {
	Name      string               `yaml:"name" json:"name,omitempty" property:"name"`
	Match     []*common.ParamMatch `yaml:"match" json:"match,omitempty" property:"match"`
	Addresses []string             `yaml:"addresses" json:"addresses,omitempty" property:"addresses"`
}

func DefaultRouterConfig() *RouterConfig {
	return &RouterConfig{
		Conditions: make([]string, 0),
		Tags:       make([]Tag, 0),
	}
}

type ConditionRule struct {
	From ConditionRuleFrom `yaml:"from" json:"from,omitempty" property:"from"`
	To   []ConditionRuleTo `yaml:"to" json:"to,omitempty" property:"to"`
}

// Equal checks if two ConditionRule instances are equal.
func (x *ConditionRule) Equal(t *ConditionRule) bool {
	if x == nil || t == nil {
		return x == t
	}
	if !reflect.DeepEqual(x.From, t.From) {
		return false
	}
	if len(x.To) != len(t.To) {
		return false
	}
	if len(x.To) == 0 {
		return true
	}
	for i := range x.To {
		if !reflect.DeepEqual(x.To[i], t.To[i]) {
			return false
		}
	}
	return true
}

type ConditionRuleFrom struct {
	Match string `yaml:"match" json:"match,omitempty" property:"match"`
}

type ConditionRuleTo struct {
	Match  string `yaml:"match" json:"match,omitempty" property:"match"`
	Weight int    `default:"100" yaml:"weight" json:"weight,omitempty" property:"weight"`
}

type ConditionRuleDisable struct {
	Match string `yaml:"match" json:"match,omitempty" property:"match"`
}

type AffinityAware struct {
	Key   string `default:"" yaml:"key" json:"key,omitempty" property:"key"`
	Ratio int32  `default:"0" yaml:"ratio" json:"ratio,omitempty" property:"ratio"`
}

// ConditionRouter -- when RouteConfigVersion == v3.1, decode by this
type ConditionRouter struct {
	Scope      string           `validate:"required" yaml:"scope" json:"scope,omitempty" property:"scope"` // must be chosen from `service` and `application`.
	Key        string           `validate:"required" yaml:"key" json:"key,omitempty" property:"key"`       // specifies which service or application the rule body acts on.
	Force      bool             `default:"false" yaml:"force" json:"force,omitempty" property:"force"`
	Runtime    bool             `default:"false" yaml:"runtime" json:"runtime,omitempty" property:"runtime"`
	Enabled    bool             `default:"true" yaml:"enabled" json:"enabled,omitempty" property:"enabled"`
	Conditions []*ConditionRule `yaml:"conditions" json:"conditions,omitempty" property:"conditions"`
}

// AffinityRouter -- RouteConfigVersion == v3.1
type AffinityRouter struct {
	Scope         string        `validate:"required" yaml:"scope" json:"scope,omitempty" property:"scope"` // must be chosen from `service` and `application`.
	Key           string        `validate:"required" yaml:"key" json:"key,omitempty" property:"key"`       // specifies which service or application the rule body acts on.
	Runtime       bool          `default:"false" yaml:"runtime" json:"runtime,omitempty" property:"runtime"`
	Enabled       bool          `default:"true" yaml:"enabled" json:"enabled,omitempty" property:"enabled"`
	AffinityAware AffinityAware `yaml:"affinityAware" json:"affinityAware,omitempty" property:"affinityAware"`
}

func (c *RouterConfig) Clone() *RouterConfig {
	if c == nil {
		return nil
	}

	var newForce *bool
	if c.Force != nil {
		newForce = new(bool)
		*newForce = *c.Force
	}

	var newRuntime *bool
	if c.Runtime != nil {
		newRuntime = new(bool)
		*newRuntime = *c.Runtime
	}

	var newEnabled *bool
	if c.Enabled != nil {
		newEnabled = new(bool)
		*newEnabled = *c.Enabled
	}

	var newValid *bool
	if c.Valid != nil {
		newValid = new(bool)
		*newValid = *c.Valid
	}

	newConditions := make([]string, len(c.Conditions))
	copy(newConditions, c.Conditions)

	newTags := make([]Tag, len(c.Tags))
	for i := range c.Tags {
		newTags[i] = c.Tags[i]
		if c.Tags[i].Match != nil {
			newTags[i].Match = make([]*common.ParamMatch, len(c.Tags[i].Match))
			for j := range c.Tags[i].Match {
				if c.Tags[i].Match[j] != nil {
					pm := *c.Tags[i].Match[j] // 深拷贝 ParamMatch（包含 Value:StringMatch）
					newTags[i].Match[j] = &pm
				}
			}
		}
		if c.Tags[i].Addresses != nil {
			newTags[i].Addresses = make([]string, len(c.Tags[i].Addresses))
			copy(newTags[i].Addresses, c.Tags[i].Addresses)
		}
	}

	return &RouterConfig{
		Scope:      c.Scope,
		Key:        c.Key,
		Force:      newForce,
		Runtime:    newRuntime,
		Enabled:    newEnabled,
		Valid:      newValid,
		Priority:   c.Priority,
		Conditions: newConditions,
		Tags:       newTags,
		ScriptType: c.ScriptType,
		Script:     c.Script,
	}
}
