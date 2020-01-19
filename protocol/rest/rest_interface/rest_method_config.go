package rest_interface

type RestMethodConfig struct {
	InterfaceName  string
	MethodName     string `required:"true" yaml:"name"  json:"name,omitempty" property:"name"`
	Url            string `yaml:"url"  json:"url,omitempty" property:"url"`
	Path           string `yaml:"rest_path"  json:"rest_path,omitempty" property:"rest_path"`
	Produces       string `yaml:"rest_produces"  json:"rest_produces,omitempty" property:"rest_produces"`
	Consumes       string `yaml:"rest_consumes"  json:"rest_consumes,omitempty" property:"rest_consumes"`
	MethodType     string `yaml:"rest_method"  json:"rest_method,omitempty" property:"rest_method"`
	PathParams     string `yaml:"rest_path_params"  json:"rest_path_params,omitempty" property:"rest_path_params"`
	PathParamsMap  map[int]string
	QueryParams    string `yaml:"rest_query_params"  json:"rest_query_params,omitempty" property:"rest_query_params"`
	QueryParamsMap map[int]string
	Body           string `yaml:"rest_body"  json:"rest_body,omitempty" property:"rest_body"`
	BodyMap        map[int]string
}
