package script

import (
	"dubbo.apache.org/dubbo-go/v3/cluster/router"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/common/extension"
)

func init() {
	/*
		Tag router is not imported in dubbo-go/imports/imports.go, because it relies on config center,
		and cause warning if config center is empty.
		User can import this package and config config center to use tag router.
	*/
	extension.SetRouterFactory(constant.TagRouterFactoryKey, NewScriptRouterFactory)
}

// RouteFactory router factory
type ScriptRouteFactory struct{}

// NewTagRouterFactory constructs a new PriorityRouterFactory
func NewScriptRouterFactory() router.PriorityRouterFactory {
	return &ScriptRouteFactory{}
}

// NewPriorityRouter construct a new PriorityRouter
func (f *ScriptRouteFactory) NewPriorityRouter() (router.PriorityRouter, error) {
	return NewScriptPriorityRouter(), nil
}
