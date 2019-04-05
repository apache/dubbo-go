package client

import (
	"context"
)

import (
	"github.com/dubbo/dubbo-go/registry"
)

type Transport interface {
	Call(ctx context.Context, url *registry.ServiceURL, request Request, resp interface{}) error
	NewRequest(conf registry.DefaultServiceConfig, method string, args interface{}) Request
}

//////////////////////////////////////////////
// Request
//////////////////////////////////////////////

type Request interface {
	ServiceConfig() registry.DefaultServiceConfig
}
