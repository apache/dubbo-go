package provider

type Provider struct {
}

func (pro *Provider) Provide(handler interface{}, info *ServiceInfo) error {
	// put information from info to url
}

type ServiceInfo struct {
	InterfaceName string
	Methods       []MethodInfo
}

type MethodInfo struct {
	Request interface{}
	Name    string
	Type    string
}
