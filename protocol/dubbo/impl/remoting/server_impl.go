package remoting

import (
	"fmt"
	"net"
)

import (
	"github.com/apache/dubbo-go/common"
	"github.com/apache/dubbo-go/common/logger"
)

import (
	"github.com/dubbogo/getty"
	gxsync "github.com/dubbogo/gost/sync"
)

// TODO: 需要移动到 业务的实现
var (
	srvConf   *ServerConfig
	srvGrpool *gxsync.TaskPool
)

// SetServerConfig ...
func SetServerConfig(s ServerConfig) {
	srvConf = &s
	err := srvConf.CheckValidity()
	if err != nil {
		panic(err)
		return
	}
	SetServerGrpool()
}

// GetServerConfig ...
func GetServerConfig() *ServerConfig {
	return srvConf
}

// SetServerGrpool ...
func SetServerGrpool() {
	if srvConf.GrPoolSize > 1 {
		srvGrpool = gxsync.NewTaskPool(gxsync.WithTaskPoolTaskPoolSize(srvConf.GrPoolSize), gxsync.WithTaskPoolTaskQueueLength(srvConf.QueueLen),
			gxsync.WithTaskPoolTaskQueueNumber(srvConf.QueueNumber))
	}
}

// Server ...
type Server struct {
	conf       ServerConfig
	tcpServer  getty.Server
	rpcHandler *RpcServerHandler
}

// NewServer ...
func NewServer(handler StubHandler) *Server {

	s := &Server{
		conf: *srvConf,
	}

	s.rpcHandler = NewRpcServerHandler(handler, s.conf.SessionNumber, s.conf.SessionTimeoutD)

	return s
}

func (s *Server) newSession(session getty.Session) error {
	var (
		ok      bool
		tcpConn *net.TCPConn
	)
	conf := s.conf

	if conf.GettySessionParam.CompressEncoding {
		session.SetCompressType(getty.CompressZip)
	}

	if tcpConn, ok = session.Conn().(*net.TCPConn); !ok {
		panic(fmt.Sprintf("%s, session.conn{%#v} is not tcp connection\n", session.Stat(), session.Conn()))
	}

	tcpConn.SetNoDelay(conf.GettySessionParam.TcpNoDelay)
	tcpConn.SetKeepAlive(conf.GettySessionParam.TcpKeepAlive)
	if conf.GettySessionParam.TcpKeepAlive {
		tcpConn.SetKeepAlivePeriod(conf.GettySessionParam.KeepAlivePeriodD)
	}
	tcpConn.SetReadBuffer(conf.GettySessionParam.TcpRBufSize)
	tcpConn.SetWriteBuffer(conf.GettySessionParam.TcpWBufSize)

	session.SetName(conf.GettySessionParam.SessionName)
	session.SetMaxMsgLen(conf.GettySessionParam.MaxMsgLen)
	session.SetPkgHandler(rpcServerPkgHandler)
	session.SetEventListener(s.rpcHandler)
	session.SetWQLen(conf.GettySessionParam.PkgWQSize)
	session.SetReadTimeout(conf.GettySessionParam.TcpReadTimeoutD)
	session.SetWriteTimeout(conf.GettySessionParam.TcpWriteTimeoutD)
	session.SetCronPeriod((int)(conf.SessionTimeoutD.Nanoseconds() / 1e6))
	session.SetWaitTime(conf.GettySessionParam.WaitTimeoutD)
	logger.Debugf("app accepts new session:%s\n", session.Stat())

	session.SetTaskPool(srvGrpool)

	return nil
}

// Start ...
func (s *Server) Start(url common.URL) {
	var (
		addr      string
		tcpServer getty.Server
	)

	addr = url.Location
	tcpServer = getty.NewTCPServer(
		getty.WithLocalAddress(addr),
	)
	tcpServer.RunEventLoop(s.newSession)
	logger.Debugf("s bind addr{%s} ok!", addr)
	s.tcpServer = tcpServer

}

// Stop ...
func (s *Server) Stop() {
	s.tcpServer.Close()
}
