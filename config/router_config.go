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
	"reflect"
)

import (
	"github.com/creasty/defaults"
)

import (
	_ "dubbo.apache.org/dubbo-go/v3/cluster/router/chain"
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	_ "dubbo.apache.org/dubbo-go/v3/metrics/prometheus"
)

// RouterConfig is the configuration of the router.
type RouterConfig struct {
	Scope      string   `validate:"required" yaml:"scope" json:"scope,omitempty" property:"scope"` // must be chosen from `service` and `application`.
	Key        string   `validate:"required" yaml:"key" json:"key,omitempty" property:"key"`       // specifies which service or application the rule body acts on.
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

type ConditionRule struct {
	From ConditionRuleFrom `yaml:"from" json:"from,omitempty" property:"from"`
	To   []ConditionRuleTo `yaml:"to" json:"to,omitempty" property:"to"`
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

// Prefix dubbo.router
func (RouterConfig) Prefix() string {
	return constant.RouterConfigPrefix
}

func (c *RouterConfig) Init() error {
	if err := defaults.Set(c); err != nil {
		return err
	}
	return verify(c)
}

func initRouterConfig(rc *RootConfig) error {
	routers := rc.Router
	if len(routers) > 0 {
		for _, r := range routers {
			if err := r.Init(); err != nil {
				return err
			}
		}
		rc.Router = routers
	}

	//chain.SetVSAndDRConfigByte(vsBytes, drBytes)
	return nil
}

type RouterConfigBuilder struct {
	routerConfig *RouterConfig
}

func NewRouterConfigBuilder() *RouterConfigBuilder {
	return &RouterConfigBuilder{routerConfig: &RouterConfig{}}
}

func (rcb *RouterConfigBuilder) SetScope(scope string) *RouterConfigBuilder {
	rcb.routerConfig.Scope = scope
	return rcb
}

func (rcb *RouterConfigBuilder) SetKey(key string) *RouterConfigBuilder {
	rcb.routerConfig.Key = key
	return rcb
}

func (rcb *RouterConfigBuilder) SetForce(force bool) *RouterConfigBuilder {
	rcb.routerConfig.Force = &force
	return rcb
}

func (rcb *RouterConfigBuilder) SetRuntime(runtime bool) *RouterConfigBuilder {
	rcb.routerConfig.Runtime = &runtime
	return rcb
}

func (rcb *RouterConfigBuilder) SetEnabled(enabled bool) *RouterConfigBuilder {
	rcb.routerConfig.Enabled = &enabled
	return rcb
}

func (rcb *RouterConfigBuilder) SetValid(valid bool) *RouterConfigBuilder {
	rcb.routerConfig.Valid = &valid
	return rcb
}

func (rcb *RouterConfigBuilder) SetPriority(priority int) *RouterConfigBuilder {
	rcb.routerConfig.Priority = priority
	return rcb
}

func (rcb *RouterConfigBuilder) SetConditions(conditions []string) *RouterConfigBuilder {
	rcb.routerConfig.Conditions = conditions
	return rcb
}

func (rcb *RouterConfigBuilder) AddCondition(condition string) *RouterConfigBuilder {
	if rcb.routerConfig.Conditions == nil {
		rcb.routerConfig.Conditions = make([]string, 0)
	}
	rcb.routerConfig.Conditions = append(rcb.routerConfig.Conditions, condition)
	return rcb
}

func (rcb *RouterConfigBuilder) SetTags(tags []Tag) *RouterConfigBuilder {
	rcb.routerConfig.Tags = tags
	return rcb
}

func (rcb *RouterConfigBuilder) AddTag(tag Tag) *RouterConfigBuilder {
	if rcb.routerConfig.Tags == nil {
		rcb.routerConfig.Tags = make([]Tag, 0)
	}
	rcb.routerConfig.Tags = append(rcb.routerConfig.Tags, tag)
	return rcb
}

func (rcb *RouterConfigBuilder) Build() *RouterConfig {
	if err := rcb.routerConfig.Init(); err != nil {
		panic(err)
	}
	return rcb.routerConfig
}

func (x *ConditionRule) Equal(t *ConditionRule) bool {
	if !reflect.DeepEqual(x.From, t.From) {
		return false
	}
	if len(x.To) != len(t.To) {
		return false
	}
	for i := range x.To {
		if !reflect.DeepEqual(x.To[i], t.To[i]) {
			return false
		}
	}
	return true
}
