package client

import (
	"context"
	"github.com/dubbo/dubbo-go/service"
)

type Transport interface {
	Call(ctx context.Context, url *service.ServiceURL, request Request, resp interface{}) error
	NewRequest(conf service.ServiceConfig, method string, args interface{}) Request
}

//////////////////////////////////////////////
// Request
//////////////////////////////////////////////

type Request struct {
	ID          int64
	Group       string
	Protocol    string
	Version     string
	Service     string
	Method      string
	Args        interface{}
	ContentType string
}

func (r *Request) ServiceConfig() service.ServiceConfigIf {
	return &service.ServiceConfig{
		Protocol: r.Protocol,
		Service:  r.Service,
		Group:    r.Group,
		Version:  r.Version,
	}
}
