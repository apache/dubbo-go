package protocol

import "github.com/dubbo/dubbo-go/config"

// Extension - Protocol
type Protocol interface {
	Export()
	Refer(url config.ConfigURL)
	Destroy()
}
