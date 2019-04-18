package extension

import protocolmod "github.com/dubbo/dubbo-go/protocol"

var (
	protocols map[string]func() protocolmod.Protocol
)

func init() {
	protocols = make(map[string]func() protocolmod.Protocol)
}
