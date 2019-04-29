package dubbo

import (
	"fmt"
	"github.com/dubbo/dubbo-go/protocol"
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

var srvConf *ServerConfig

const CONF_DUBBO_SERVER_FILE_PATH = "CONF_SERVER_FILE_PATH"

func init() {
	// load serverconfig from *.yml
	path := os.Getenv(CONF_DUBBO_SERVER_FILE_PATH)
	if path == "" {
		log.Warn("CONF_SERVER_FILE_PATH is null")
		return
	}

	file, err := ioutil.ReadFile(path)
	if err != nil {
		log.Warn(jerrors.Trace(err))
		return
	}

	conf := &ServerConfig{}
	err = yaml.Unmarshal(file, conf)
	if err != nil {
		log.Warn(jerrors.Trace(err))
		return
	}

	if err := conf.CheckValidity(); err != nil {
		log.Warn("ServerConfig check failed: ", err)
		return
	}

	srvConf = conf
}

func SetServerConfig(s ServerConfig) {
	srvConf = &s
}

func GetServerConfig() ServerConfig {
	return *srvConf
}

type Server struct {
	conf      ServerConfig
	tcpServer getty.Server
	exporter  protocol.Exporter
}

func NewServer(exporter protocol.Exporter) *Server {

	s := &Server{
		exporter: exporter,
		conf:     *srvConf,
	}

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
	session.SetPkgHandler(NewRpcServerPackageHandler())
	session.SetEventListener(NewRpcServerHandler(s.exporter, conf.SessionNumber, conf.sessionTimeout))
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
	s.tcpServer = tcpServer

}

func (s *Server) Stop() {
	s.tcpServer.Close()
}
