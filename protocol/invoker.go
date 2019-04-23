package protocol

import "github.com/dubbo/dubbo-go/config"

// Extension - Invoker
type Invoker interface {
	Invoke(Invocation) Result
	GetURL() config.URL
	Destroy()
}
