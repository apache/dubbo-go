package config

import (
	"github.com/ghodss/yaml"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

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

//func (vsc *VirtualServiceConfig) ChangeJsonTab2YamlTabAndGetYamlByte() ([]byte, error) {
//	data, err := json.Marshal(vsc)
//	if err != nil {
//		logger.Error("marshal virtual service config err:", err)
//		return []byte{}, err
//	}
//	yaml.Unmarshal(data, vsc)
//
//}

type UniformRouterConfigSpec struct {
	Hosts []string      `yaml:"hosts" json:"hosts"`
	Dubbo []*DubboRoute `yaml:"dubbo" json:"dubbo"`
}

type DubboRoute struct {
	Services     []*StringMatch            `yaml:"service" json:"service"`
	RouterDetail []*DubboServiceRouterItem `yaml:"routedetail" json:"routedetail"`
}

type DubboServiceRouterItem struct {
	Name   string               `yaml:"name" json:"name"`
	Match  []*DubboMatchRequest `yaml:"match" json:"match"`
	Router []*DubboDestination  `yaml:"route" json:"route"`
}

type DubboMatchRequest struct {
	Name         string                  `yaml:"name" json:"name"`
	Method       *DubboMethodMatch       `yaml:"method" json:"method"`
	SourceLabels map[string]string       `yaml:"sourceLabels" json:"sourceLabels"`
	Attachment   *DubboAttachmentMatch   `yaml:"attachments" json:"attachments"`
	Header       map[string]*StringMatch `yaml:"headers" json:"headers"`
	Threshold    *DoubleMatch            `yaml:"threshold" json:"threshold"`
}

type DoubleMatch struct {
	Exact float64           `yaml:"exact" json:"exact"`
	Range *DoubleRangeMatch `yaml:"range" json:"range"`
	Mode  float64           `yaml:"mode" json:"mode"`
}

type DoubleRangeMatch struct {
	Start float64 `yaml:"start" json:"start"`
	End   float64 `yaml:"end" json:"end"`
}

type DubboAttachmentMatch struct {
	EagleeyeContext map[string]*StringMatch `yaml:"eagleeyecontext" json:"eagleeyecontext"`
	DubboContext    map[string]*StringMatch `yaml:"dubbocontext" json:"dubbocontext"`
}

type DubboMethodMatch struct {
	NameMatch *StringMatch            `yaml:"name_match" json:"name_match"`
	Argc      int                     `yaml:"argc" json:"argc"`
	Args      []*DubboMethodArg       `yaml:"args" json:"args"`
	Argp      []*StringMatch          `yaml:"argp" json:"argp"`
	Headers   map[string]*StringMatch `yaml:"headers" json:"headers"`
}

type DubboMethodArg struct {
	Index     uint32           `yaml:"index" json:"index"`
	Type      string           `yaml:"type" json:"type"`
	StrValue  *ListStringMatch `yaml:"str_value" json:"str_value"`
	NumValue  *ListDoubleMatch `yaml:"num_value" json:"num_value"`
	BoolValue *BoolMatch       `yaml:"bool_value" json:"bool_value"`
	//todo reserve field
}

type ListStringMatch struct {
	Oneof []*StringMatch `yaml:"oneof" json:"oneof"`
}
type ListDoubleMatch struct {
	Oneof []*DoubleMatch `yaml:"oneof" json:"oneof"`
}

type BoolMatch struct {
	Exact bool `yaml:"exact" json:"exact"`
}

type StringMatch struct {
	Exact   string `yaml:"exact" json:"exact"`
	Prefix  string `yaml:"prefix" json:"prefix"`
	Regex   string `yaml:"regex" json:"regex"`
	NoEmpty string `yaml:"noempty" json:"noempty"`
	Empty   string `yaml:"empty" json:"empty"`
}

type DubboDestination struct {
	Destination RouterDest `yaml:"destination" json:"destination"`
	//Subset      string            `yaml:"subset"`
	Fallback *DubboDestination `yaml:"fallback" json:"fallback"`
}

type RouterDest struct {
	Host   string `yaml:"host" json:"host"`
	Subset string `yaml:"subset" json:"subset"`
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

func (drc *DestinationRuleConfig) DeepCopyObject() runtime.Object {
	data, _ := yaml.Marshal(drc)
	out := &DestinationRuleConfig{}
	yaml.Unmarshal(data, out)
	return out
}

type DestinationRuleSpec struct {
	Host    string                  `yaml:"host" json:"host"`
	SubSets []DestinationRuleSubSet `yaml:"subsets" json:"subsets"`
}
type DestinationRuleSubSet struct {
	Name   string            `yaml:"name" json:"name"`
	Labels map[string]string `yaml:"labels" json:"labels"`
}

func (urc *VirtualServiceConfig) DeepCopyObject() runtime.Object {
	data, _ := yaml.Marshal(urc)
	out := &VirtualServiceConfig{}
	yaml.Unmarshal(data, out)
	return out
}

//func (drc *DestinationRuleSpec) DeepCopyObject() runtime.Object {
//	data, _ := yaml.Marshal(drc)
//	out := &DestinationRuleSpec{}
//	yaml.Unmarshal(data, out)
//	return out
//}

type DestinationRuleSpecList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []DestinationRuleSpec `json:"items"`
}

//func (urc *DestinationRuleSpecList) DeepCopyObject() runtime.Object {
//	out := &DestinationRuleSpecList{
//		TypeMeta: urc.TypeMeta,
//		ListMeta: urc.ListMeta,
//	}
//	for _, v := range urc.Items {
//		spec := v.DeepCopyObject().(*DestinationRuleSpec)
//		out.Items = append(out.Items, *spec)
//	}
//	return out
//}

type VirtualServiceConfigList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []VirtualServiceConfig `json:"items"`
}

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

type DestinationRuleConfigList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []DestinationRuleConfig `json:"items"`
}

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
