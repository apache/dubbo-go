package dubbo

import (
	"fmt"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"net"
	"os"
)

import (
	"github.com/AlexStocks/getty"
	log "github.com/AlexStocks/log4go"
	jerrors "github.com/juju/errors"
)

import (
	"github.com/dubbo/dubbo-go/config"
)

var srv = NewServer()

const CONF_SERVER_FILE_PATH = "CONF_SERVER_FILE_PATH"

type Server struct {
	conf          ServerConfig
	tcpServerList []getty.Server
}

func NewServer() *Server {

	s := &Server{}

	// load serverconfig from *.yml
	path := os.Getenv(CONF_SERVER_FILE_PATH)
	if path == "" {
		log.Info("CONF_SERVER_FILE_PATH is null")
		return s
	}

	file, err := ioutil.ReadFile(path)
	if err != nil {
		log.Error(jerrors.Trace(err))
		return s
	}

	conf := &ServerConfig{}
	err = yaml.Unmarshal(file, conf)
	if err != nil {
		log.Error(jerrors.Trace(err))
		return s
	}

	if err := conf.CheckValidity(); err != nil {
		log.Error("ServerConfig check failed: ", err)
		return s
	}

	s.conf = *conf

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
		tcpConn.SetKeepAlivePeriod(conf.GettySessionParam.keepAlivePeriod)
	}
	tcpConn.SetReadBuffer(conf.GettySessionParam.TcpRBufSize)
	tcpConn.SetWriteBuffer(conf.GettySessionParam.TcpWBufSize)

	session.SetName(conf.GettySessionParam.SessionName)
	session.SetMaxMsgLen(conf.GettySessionParam.MaxMsgLen)
	session.SetPkgHandler(NewRpcServerPackageHandler()) // TODO: now, a server will bind all service
	session.SetEventListener(NewRpcServerHandler(conf.SessionNumber, conf.sessionTimeout))
	session.SetRQLen(conf.GettySessionParam.PkgRQSize)
	session.SetWQLen(conf.GettySessionParam.PkgWQSize)
	session.SetReadTimeout(conf.GettySessionParam.tcpReadTimeout)
	session.SetWriteTimeout(conf.GettySessionParam.tcpWriteTimeout)
	session.SetCronPeriod((int)(conf.sessionTimeout.Nanoseconds() / 1e6))
	session.SetWaitTime(conf.GettySessionParam.waitTimeout)
	log.Debug("app accepts new session:%s\n", session.Stat())

	return nil
}

func (s *Server) Start(url config.URL) {
	var (
		addr      string
		tcpServer getty.Server
	)

	addr = url.Location
	tcpServer = getty.NewTCPServer(
		getty.WithLocalAddress(addr),
	)
	tcpServer.RunEventLoop(s.newSession)
	log.Debug("s bind addr{%s} ok!", addr)
	s.tcpServerList = append(s.tcpServerList, tcpServer)

}

func (s *Server) Stop() {
	list := s.tcpServerList
	s.tcpServerList = nil
	if list != nil {
		for _, tcpServer := range list {
			tcpServer.Close()
		}
	}
}
