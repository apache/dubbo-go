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
