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
	"github.com/ghodss/yaml"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

// nolint
type MetaDataStruct struct {
	Name string `yaml:"name"`
}

// VirtualService Config Definition
type VirtualServiceConfig struct {
	YamlAPIVersion    string `yaml:"apiVersion"`
	YamlKind          string `yaml:"kind"`
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	MetaData          MetaDataStruct          `yaml:"metadata"`
	Spec              UniformRouterConfigSpec `yaml:"spec" json:"spec"`
}

// nolint
type UniformRouterConfigSpec struct {
	Hosts []string      `yaml:"hosts" json:"hosts"`
	Dubbo []*DubboRoute `yaml:"dubbo" json:"dubbo"`
}

// nolint
type DubboRoute struct {
	Services     []*StringMatch            `yaml:"services" json:"service"`
	RouterDetail []*DubboServiceRouterItem `yaml:"routedetail" json:"routedetail"`
}

// nolint
type DubboServiceRouterItem struct {
	Name   string               `yaml:"name" json:"name"`
	Match  []*DubboMatchRequest `yaml:"match" json:"match"`
	Router []*DubboDestination  `yaml:"route" json:"route"`
	// todo mirror/retries/timeout
}

// nolint
type DubboMatchRequest struct {
	Name         string                  `yaml:"name" json:"name"`
	Method       *DubboMethodMatch       `yaml:"method" json:"method"`
	SourceLabels map[string]string       `yaml:"sourceLabels" json:"sourceLabels"`
	Attachment   *DubboAttachmentMatch   `yaml:"attachments" json:"attachments"`
	Header       map[string]*StringMatch `yaml:"headers" json:"headers"`
	Threshold    *DoubleMatch            `yaml:"threshold" json:"threshold"`
}

// nolint
type DoubleMatch struct {
	Exact float64           `yaml:"exact" json:"exact"`
	Range *DoubleRangeMatch `yaml:"range" json:"range"`
	Mode  float64           `yaml:"mode" json:"mode"`
}

// nolint
type DoubleRangeMatch struct {
	Start float64 `yaml:"start" json:"start"`
	End   float64 `yaml:"end" json:"end"`
}

// nolint
type DubboAttachmentMatch struct {
	EagleeyeContext map[string]*StringMatch `yaml:"eagleeyecontext" json:"eagleeyecontext"`
	DubboContext    map[string]*StringMatch `yaml:"dubbocontext" json:"dubbocontext"`
}

// nolint
type DubboMethodMatch struct {
	NameMatch *StringMatch            `yaml:"name_match" json:"name_match"`
	Argc      int                     `yaml:"argc" json:"argc"`
	Args      []*DubboMethodArg       `yaml:"args" json:"args"`
	Argp      []*StringMatch          `yaml:"argp" json:"argp"`
	Headers   map[string]*StringMatch `yaml:"headers" json:"headers"`
}

// nolint
type DubboMethodArg struct {
	Index     uint32           `yaml:"index" json:"index"`
	Type      string           `yaml:"type" json:"type"`
	StrValue  *ListStringMatch `yaml:"str_value" json:"str_value"`
	NumValue  *ListDoubleMatch `yaml:"num_value" json:"num_value"`
	BoolValue *BoolMatch       `yaml:"bool_value" json:"bool_value"`
	//todo reserve field
}

// nolint
type ListStringMatch struct {
	Oneof []*StringMatch `yaml:"oneof" json:"oneof"`
}

// nolint
type ListDoubleMatch struct {
	Oneof []*DoubleMatch `yaml:"oneof" json:"oneof"`
}

// nolint
type BoolMatch struct {
	Exact bool `yaml:"exact" json:"exact"`
}

// nolint
type StringMatch struct {
	Exact   string `yaml:"exact" json:"exact"`
	Prefix  string `yaml:"prefix" json:"prefix"`
	Regex   string `yaml:"regex" json:"regex"`
	NoEmpty string `yaml:"noempty" json:"noempty"`
	Empty   string `yaml:"empty" json:"empty"`
}

// nolint
type DubboDestination struct {
	Destination RouterDest `yaml:"destination" json:"destination"`
	//Subset      string            `yaml:"subset"`
}

// nolint
type RouterDest struct {
	Host     string            `yaml:"host" json:"host"`
	Subset   string            `yaml:"subset" json:"subset"`
	Weight   int               `yaml:"weight" json:"weight"`
	Fallback *DubboDestination `yaml:"fallback" json:"fallback"`
	// todo port
}

// DestinationRule Definition
type DestinationRuleConfig struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	YamlAPIVersion    string              `yaml:"apiVersion" `
	YamlKind          string              `yaml:"kind" `
	MetaData          MetaDataStruct      `yaml:"metadata"`
	Spec              DestinationRuleSpec `yaml:"spec" json:"spec"`
}

// nolint
func (drc *DestinationRuleConfig) DeepCopyObject() runtime.Object {
	data, _ := yaml.Marshal(drc)
	out := &DestinationRuleConfig{}
	yaml.Unmarshal(data, out)
	return out
}

// nolint
type DestinationRuleSpec struct {
	Host    string                  `yaml:"host" json:"host"`
	SubSets []DestinationRuleSubSet `yaml:"subsets" json:"subsets"`
}

// nolint
type DestinationRuleSubSet struct {
	Name   string            `yaml:"name" json:"name"`
	Labels map[string]string `yaml:"labels" json:"labels"`
}

// nolint
func (urc *VirtualServiceConfig) DeepCopyObject() runtime.Object {
	data, _ := yaml.Marshal(urc)
	out := &VirtualServiceConfig{}
	yaml.Unmarshal(data, out)
	return out
}

// nolint
type DestinationRuleSpecList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []DestinationRuleSpec `json:"items"`
}

// nolint
type VirtualServiceConfigList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []VirtualServiceConfig `json:"items"`
}

// nolint
func (drc *VirtualServiceConfigList) DeepCopyObject() runtime.Object {
	out := &VirtualServiceConfigList{
		TypeMeta: drc.TypeMeta,
		ListMeta: drc.ListMeta,
	}
	for _, v := range drc.Items {
		spec := v.DeepCopyObject().(*VirtualServiceConfig)
		out.Items = append(out.Items, *spec)
	}
	return out
}

// nolint
type DestinationRuleConfigList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []DestinationRuleConfig `json:"items"`
}

// nolint
func (drc *DestinationRuleConfigList) DeepCopyObject() runtime.Object {
	out := &DestinationRuleConfigList{
		TypeMeta: drc.TypeMeta,
		ListMeta: drc.ListMeta,
	}
	for _, v := range drc.Items {
		spec := v.DeepCopyObject().(*DestinationRuleConfig)
		out.Items = append(out.Items, *spec)
	}
	return out
}
