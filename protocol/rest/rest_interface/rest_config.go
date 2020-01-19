package rest_interface

type RestConfig struct {
	InterfaceName        string              `required:"true"  yaml:"interface"  json:"interface,omitempty" property:"interface"`
	Url                  string              `yaml:"url"  json:"url,omitempty" property:"url"`
	Path                 string              `yaml:"rest_path"  json:"rest_path,omitempty" property:"rest_path"`
	Produces             string              `yaml:"rest_produces"  json:"rest_produces,omitempty" property:"rest_produces"`
	Consumes             string              `yaml:"rest_consumes"  json:"rest_consumes,omitempty" property:"rest_consumes"`
	MethodType           string              `yaml:"rest_method"  json:"rest_method,omitempty" property:"rest_method"`
	Client               string              `yaml:"rest_client" json:"rest_client,omitempty" property:"rest_client"`
	Server               string              `yaml:"rest_server" json:"rest_server,omitempty" property:"rest_server"`
	RestMethodConfigs    []*RestMethodConfig `yaml:"methods" json:"methods,omitempty" property:"methods"`
	RestMethodConfigsMap map[string]*RestMethodConfig
}

type RestConsumerConfig struct {
	Client        string                 `default:"resty" yaml:"rest_client" json:"rest_client,omitempty" property:"rest_client"`
	Produces      string                 `yaml:"rest_produces"  json:"rest_produces,omitempty" property:"rest_produces"`
	Consumes      string                 `yaml:"rest_consumes"  json:"rest_consumes,omitempty" property:"rest_consumes"`
	RestConfigMap map[string]*RestConfig `yaml:"references" json:"references,omitempty" property:"references"`
}

type RestProviderConfig struct {
	Server        string                 `default:"go-restful" yaml:"rest_server" json:"rest_server,omitempty" property:"rest_server"`
	Produces      string                 `yaml:"rest_produces"  json:"rest_produces,omitempty" property:"rest_produces"`
	Consumes      string                 `yaml:"rest_consumes"  json:"rest_consumes,omitempty" property:"rest_consumes"`
	RestConfigMap map[string]*RestConfig `yaml:"services" json:"services,omitempty" property:"services"`
}
