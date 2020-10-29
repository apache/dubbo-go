package common

import (
	"github.com/apache/dubbo-go/tools/cli/protocol"
)

var (
	protocols = make(map[string]func() protocol.Protocol)
)

// SetProtocol sets the protocol extension with @name
func SetProtocol(name string, v func() protocol.Protocol) {
	protocols[name] = v
}

// GetProtocol finds the protocol extension with @name
func GetProtocol(name string) protocol.Protocol {
	if protocols[name] == nil {
		panic("protocol for " + name + " is not existing, make sure you have import the package.")
	}
	return protocols[name]()
}
