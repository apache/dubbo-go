package client

import "github.com/dubbo/go-for-apache-dubbo/registry"

type ServiceArrayIf interface {
	GetIdx() *int64
	GetSize() int64
	GetService(i int64) registry.ServiceURL
}
