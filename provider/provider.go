package provider

import "context"

type Provider struct {
}

// Provide assemble invoker chains like ProviderConfig.Load, init a service per call
func (pro *Provider) Provide(handler interface{}, info *ServiceInfo, opts ...Option) error {
	// put information from info to url
	// ProviderConfig.Load

	// url
	return nil
}

// meta
type ServiceInfo struct {
	InterfaceName string
	ServiceType   interface{}
	Methods       []MethodInfo
	Meta          map[string]interface{}
}

type MethodInfo struct {
	Name           string
	Type           string
	ReqInitFunc    func() interface{}
	StreamInitFunc func(baseStream interface{}) interface{}
	MethodFunc     func(ctx context.Context, args []interface{}, handler interface{}) (interface{}, error)
	Meta           map[string]interface{}
}

func NewProvider() (*Provider, error) {
	return nil, nil
}
