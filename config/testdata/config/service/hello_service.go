package service

import (
	"context"
)

type HelloService struct {
	// say hello
	Say func(ctx context.Context, req []interface{}) error
}

func (HelloService) Reference() string {
	return "helloService"
}
