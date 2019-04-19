package extension

import  "github.com/dubbo/dubbo-go/protocol"

var (
	protocols map[string]func() protocol.Protocol
)

func init() {
	protocols = make(map[string]func() protocol.Protocol)
}

func SetRefProtocol(fn func() protocol.Protocol){
	protocols["refProtocol"] = fn
}

func GetRefProtocol() protocol.Protocol{
	return protocols["refProtocol"]()
}
