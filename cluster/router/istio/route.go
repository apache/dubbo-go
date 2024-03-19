package istio

import (
	"dubbo.apache.org/dubbo-go/v3/cluster/router"
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/config_center"
	"dubbo.apache.org/dubbo-go/v3/protocol"
)

const (
	name = "xds-router"
)

// XdsRouter have
type XdsRouter struct {
}

// NewXdsRouter construct an NewConnCheckRouter via url
func NewXdsRouter() (router.PriorityRouter, error) {
	return &XdsRouter{}, nil
}

// Route gets a list of routed invoker
func (r *XdsRouter) Route(invokers []protocol.Invoker, url *common.URL, invocation protocol.Invocation) []protocol.Invoker {
	return nil
}

// Process there is no process needs for uniform Router, as it upper struct RouterChain has done it
func (r *XdsRouter) Process(event *config_center.ConfigChangeEvent) {
}

// Name get name of ConnCheckerRouter
func (r *XdsRouter) Name() string {
	return name
}

// Priority get Router priority level
func (r *XdsRouter) Priority() int64 {
	return 0
}

// URL Return URL in router
func (r *XdsRouter) URL() *common.URL {
	return nil
}

// Notify the router the invoker list
func (r *XdsRouter) Notify(invokers []protocol.Invoker) {
}
