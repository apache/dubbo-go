package remoting

import (
	"github.com/apache/dubbo-go/common"
)

type Server interface {
	//invoke once for connection
	Start()
	Stop()
}

type ExchangeServer struct {
	Server Server
}

func NewExchangeServer(url common.URL, server Server) *ExchangeServer {
	exchangServer := &ExchangeServer{
		Server: server,
	}
	return exchangServer
}

func (server *ExchangeServer) Start() {
	(server.Server).Start()
}

func (server *ExchangeServer) Stop() {
	(server.Server).Stop()
}
