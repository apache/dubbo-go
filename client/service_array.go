package client

import "github.com/dubbo/dubbo-go/registry"

type ServiceArrayIf interface {
	GetIdx() *int64
	GetSize() int64
	GetService(i int64) config.ConfigURL
}
