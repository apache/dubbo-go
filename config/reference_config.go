package config

type ReferenceConfig struct {
	Interface  string                    `required:"true"  yaml:"interface"  json:"interface,omitempty"`
	Registries []referenceConfigRegistry `required:"true"  yaml:"registries"  json:"registries,omitempty"`
	URLs       []ConfigURL               `yaml:"-"`
}
type referenceConfigRegistry struct {
	string
}

func NewReferenceConfig() *ReferenceConfig {
	return &ReferenceConfig{}
}

func (refconfig *ReferenceConfig) CreateProxy() {
	//首先是user specified URL, could be peer-to-peer address, or register center's address.

	//其次是assemble URL from register center's configuration模式

}

func (refconfig *ReferenceConfig) loadRegistries() []ConfigURL {
	for _, registry := range refconfig.Registries {
		for _, registryConf := range consumerConfig.Registries {
			if registry.string == registryConf.Id {

			}
		}

	}
}
