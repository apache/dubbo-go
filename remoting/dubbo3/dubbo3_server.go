package dubbo3

import (
	"github.com/apache/dubbo-go/common"
	"github.com/apache/dubbo-go/common/logger"
	"net"
	"sync"
)


type TripleServer struct {
	lst net.Listener
	addr string
	rpcService common.RPCService
	url *common.URL
}

func NewTripleServer(url *common.URL, service common.RPCService) *TripleServer {
	return &TripleServer{
		addr: url.Location,
		rpcService: service,
	}
}

// todo stop server
func (t*TripleServer) Stop(){

}


func (t*TripleServer) Start(){
	logger.Warn("In Start()")
	logger.Warn("tripleServer Start at ", t.addr)
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

func (t *TripleServer) handleRawConn(conn net.Conn) {
	h2Controller := NewH2Controller(conn, true, t.rpcService, t.url)
	h2Controller.H2ShakeHand()
	h2Controller.run()
}