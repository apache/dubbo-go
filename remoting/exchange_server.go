package remoting

import (
	"github.com/apache/dubbo-go/common"
	"github.com/apache/dubbo-go/protocol"
	"github.com/apache/dubbo-go/protocol/invocation"
	"github.com/apache/dubbo-go/remoting/getty"
)


type ExchangeServer struct {
	Server *getty.Server
}

func NewExchangeServer(url common.URL, handler func(*invocation.RPCInvocation) protocol.RPCResult) *ExchangeServer {
	server := getty.NewServer(url, handler)
	exchangServer := &ExchangeServer{
		Server: server,
	}
	return exchangServer
}

func (server *ExchangeServer) Start() {
	server.Server.Start()
}

func (server *ExchangeServer) Stop() {
	server.Server.Stop()
}
