package service

import (
	"context"
)

type HelloService struct {
	// say hello
	Say func(ctx context.Context, req []interface{}) error
}

func (HelloService) Name() string {
	return "helloService"
}

func (HelloService) Reference() string {
	return "org.github.dubbo.HelloService"
}
