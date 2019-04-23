package protocol

import "github.com/dubbo/dubbo-go/config"

// Extension - Protocol
type Protocol interface {
	Export(invoker Invoker) Exporter
	Refer(url config.URL) Invoker
	Destroy()
}

type Exporter interface {
	GetInvoker() Invoker
	Unexport()
}
