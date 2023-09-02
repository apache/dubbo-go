package global

// MethodConfig defines method config
type MethodConfig struct {
	InterfaceId                 string
	InterfaceName               string
	Name                        string `yaml:"name"  json:"name,omitempty" property:"name"`
	Retries                     string `yaml:"retries"  json:"retries,omitempty" property:"retries"`
	LoadBalance                 string `yaml:"loadbalance"  json:"loadbalance,omitempty" property:"loadbalance"`
	Weight                      int64  `yaml:"weight"  json:"weight,omitempty" property:"weight"`
	TpsLimitInterval            string `yaml:"tps.limit.interval" json:"tps.limit.interval,omitempty" property:"tps.limit.interval"`
	TpsLimitRate                string `yaml:"tps.limit.rate" json:"tps.limit.rate,omitempty" property:"tps.limit.rate"`
	TpsLimitStrategy            string `yaml:"tps.limit.strategy" json:"tps.limit.strategy,omitempty" property:"tps.limit.strategy"`
	ExecuteLimit                string `yaml:"execute.limit" json:"execute.limit,omitempty" property:"execute.limit"`
	ExecuteLimitRejectedHandler string `yaml:"execute.limit.rejected.handler" json:"execute.limit.rejected.handler,omitempty" property:"execute.limit.rejected.handler"`
	Sticky                      bool   `yaml:"sticky"   json:"sticky,omitempty" property:"sticky"`
	RequestTimeout              string `yaml:"timeout"  json:"timeout,omitempty" property:"timeout"`
}
