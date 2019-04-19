package client

import (
	"context"
)

import (
	"github.com/dubbo/dubbo-go/registry"
)

type Transport interface {
	Call(ctx context.Context, url config.ConfigURL, request Request, resp interface{}) error
	NewRequest(conf registry.ReferenceConfig, method string, args interface{}) (Request, error)
}

//////////////////////////////////////////////
// Request
//////////////////////////////////////////////

type Request interface {
	ServiceConfig() registry.ReferenceConfig
}
