package extension

import (
	"github.com/dubbo/go-for-apache-dubbo/protocol"
)

var (
	protocols map[string]func() protocol.Protocol
)

func init() {
	protocols = make(map[string]func() protocol.Protocol)
}

func SetProtocol(name string, v func() protocol.Protocol) {
	protocols[name] = v
}

func GetProtocolExtension(name string) protocol.Protocol {
	return protocols[name]()
}
