package protocol

import "github.com/dubbo/dubbo-go/config"

// Extension - Invoker
type Invoker interface {
	Invoke()
	GetURL() config.URL
	Destroy()
}
