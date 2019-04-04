package client

import (
	"context"
)

import (
	"github.com/dubbo/dubbo-go/registry"
)

type Transport interface {
	Call(ctx context.Context, url *registry.DefaultServiceURL, request Request, resp interface{}) error
	NewRequest(conf registry.ServiceConfig, method string, args interface{}) (Request,error)
}

//////////////////////////////////////////////
// Request
//////////////////////////////////////////////

type Request interface {
	ServiceConfig() registry.ServiceConfig
}
