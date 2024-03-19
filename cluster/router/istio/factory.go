package istio

import (
	"dubbo.apache.org/dubbo-go/v3/cluster/router"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/common/extension"
)

func init() {
	extension.SetRouterFactory(constant.XdsRouterFactoryKey, NewXdsRouterFactory)
}

// MeshRouterFactory is mesh router's factory
type XdsRouterFactory struct{}

// NewMeshRouterFactory constructs a new PriorityRouterFactory
func NewXdsRouterFactory() router.PriorityRouterFactory {
	return &XdsRouterFactory{}
}

// NewPriorityRouter construct a new UniformRouteFactory as PriorityRouter
func (f *XdsRouterFactory) NewPriorityRouter() (router.PriorityRouter, error) {
	return NewXdsRouter()
}
