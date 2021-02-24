package config

type MetaDataStruct struct {
	Name string `yaml:"name"`
}

// VirtualService Config Definition

type VirtualServiceConfig struct {
	APIVersion string                  `yaml:"apiVersion"`
	Kind       string                  `yaml:"kind"`
	MetaData   MetaDataStruct          `yaml:"metadata"`
	Spec       UniformRouterConfigSpec `yaml:"spec"`
}

type UniformRouterConfigSpec struct {
	Hosts []string      `yaml:"hosts"`
	Dubbo []*DubboRoute `yaml:"dubbo"`
}

type DubboRoute struct {
	Services     []*StringMatch            `yaml:"service"`
	RouterDetail []*DubboServiceRouterItem `yaml:"routedetail"`
}

type DubboServiceRouterItem struct {
	Name   string               `yaml:"name"`
	Match  []*DubboMatchRequest `yaml:"match"`
	Router []*DubboDestination  `yaml:"route"`
}

type DubboMatchRequest struct {
	Name         string                  `yaml:"name"`
	Method       *DubboMethodMatch       `yaml:"method"`
	SourceLabels map[string]string       `yaml:"sourceLabels"`
	Attachment   *DubboAttachmentMatch   `yaml:"attachments"`
	Header       map[string]*StringMatch `yaml:"headers"`
	Threshold    *DoubleMatch            `yaml:"threshold"`
}

type DoubleMatch struct {
}

type DubboAttachmentMatch struct {
	EagleeyeContext map[string]*StringMatch
	DubboContext    map[string]*StringMatch
}

type DubboMethodMatch struct {
	NameMatch *StringMatch            `yaml:"name_match"`
	Argc      int                     `yaml:"argc"`
	Args      []*DubboMethodArg       `yaml:"args"`
	Argp      []*StringMatch          `yaml:"argp"`
	Headers   map[string]*StringMatch `yaml:"headers"`
}

type DubboMethodArg struct {
	Index     uint32          `yaml:"index"`
	Type      string          `yaml:"type"`
	StrValue  ListStringMatch `yaml:"str_value"`
	NumValue  ListDoubleMatch `yaml:"num_value"`
	BoolValue BoolMatch       `yaml:"bool_value"`
	//todo reserve field
}

type ListStringMatch struct {
}
type ListDoubleMatch struct {
}
type BoolMatch struct {
}

type StringMatch struct {
	Exact   string `yaml:"exact"`
	Prefix  string `yaml:"prefix"`
	Regex   string `yaml:"regex"`
	NoEmpty string `yaml:"noempty"`
	Empty   string `yaml:"empty"`
}

type DubboDestination struct {
	Destination RouterDest `yaml:"destination"`
	//Subset      string            `yaml:"subset"`
	Fallback *DubboDestination `yaml:"fallback"`
}

type RouterDest struct {
	Host   string `yaml:"host"`
	Subset string `yaml:"subset"`
}

// DestinationRule Definition

type DestinationRuleConfig struct {
	APIVersion string              `yaml:"apiVersion"`
	Kind       string              `yaml:"kind"`
	MetaData   MetaDataStruct      `yaml:"metadata"`
	Spec       DestinationRuleSpec `yaml:"spec"`
}

type DestinationRuleSpec struct {
	Host    string                  `yaml:"host"`
	SubSets []DestinationRuleSubSet `yaml:"subsets"`
}
type DestinationRuleSubSet struct {
	Name   string            `yaml:"name"`
	Labels map[string]string `yaml:"labels"`
}
