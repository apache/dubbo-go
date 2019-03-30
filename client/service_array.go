package client

import "github.com/dubbo/dubbo-go/service"

type ServiceArrayIf interface{
	GetIdx()*int64
	GetSize()int
	GetService(i int)*service.ServiceURL
}
