package dubbo3

import (
	"net"
	"sync"
)

import (
	"github.com/apache/dubbo-go/common"
	"github.com/apache/dubbo-go/common/logger"
)

type TripleServer struct {
	lst        net.Listener
	addr       string
	rpcService common.RPCService
	url        *common.URL
}

func NewTripleServer(url *common.URL, service common.RPCService) *TripleServer {
	return &TripleServer{
		addr:       url.Location,
		rpcService: service,
		url:        url,
	}
}

// todo stop server
func (t *TripleServer) Stop() {

}

func (t *TripleServer) Start() {
	logger.Info("tripleServer Start at ", t.addr)
	lst, err := net.Listen("tcp", t.addr)
	if err != nil {
		panic(err)
	}
	t.lst = lst
	t.Run()
}

func (t *TripleServer) Run() {
	wg := sync.WaitGroup{}
	for {
		conn, err := t.lst.Accept()
		if err != nil {
			panic(err)
		}
		wg.Add(1)
		go func() {
			t.handleRawConn(conn)
			wg.Done()
		}()
	}
}

func (t *TripleServer) handleRawConn(conn net.Conn) error {
	h2Controller, err := NewH2Controller(conn, true, t.rpcService, t.url)
	if err != nil {
		return err
	}
	h2Controller.H2ShakeHand()
	return nil
}
