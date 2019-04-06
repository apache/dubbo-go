package dubbo

import (
	"fmt"
	"github.com/dubbo/dubbo-go/plugins"
	"net"
	"reflect"
	"strconv"
)

import (
	"github.com/AlexStocks/getty"
	"github.com/AlexStocks/goext/net"
	log "github.com/AlexStocks/log4go"
	jerrors "github.com/juju/errors"
)

import (
	"github.com/dubbo/dubbo-go/registry"
)

type Option func(*Options)

type Options struct {
	Registry        registry.Registry
	ConfList        []ServerConfig
	ServiceConfList []registry.ServiceConfig
}

func newOptions(opt ...Option) Options {
	opts := Options{}
	for _, o := range opt {
		o(&opts)
	}

	if opts.Registry == nil {
		panic("server.Options.Registry is nil")
	}

	return opts
}

// Registry used for discovery
func Registry(r registry.Registry) Option {
	return func(o *Options) {
		o.Registry = r
	}
}

func ConfList(confList []ServerConfig) Option {
	return func(o *Options) {
		o.ConfList = confList
		for i := 0; i < len(o.ConfList); i++ {
			if err := o.ConfList[i].CheckValidity(); err != nil {
				log.Error("ServerConfig check failed: ", err)
				o.ConfList = []ServerConfig{}
				return
			}
			if o.ConfList[i].IP == "" {
				o.ConfList[i].IP, _ = gxnet.GetLocalIP()
			}
		}
	}
}

func ServiceConfList(confList []registry.ServiceConfig) Option {
	return func(o *Options) {
		o.ServiceConfList = confList
		if o.ServiceConfList == nil {
			o.ServiceConfList = []registry.ServiceConfig{}
		}
	}
}

type serviceMap map[string]*service

type Server struct {
	opts            Options
	indexOfConfList int
	srvs            []serviceMap
	tcpServerList   []getty.Server
}

func NewServer(opts ...Option) *Server {
	options := newOptions(opts...)
	num := len(options.ConfList)
	servers := make([]serviceMap, len(options.ConfList))

	for i := 0; i < num; i++ {
		servers[i] = map[string]*service{}
	}

	s := &Server{
		opts: options,
		srvs: servers,
	}

	return s
}

// Register export services and register with the registry
func (s *Server) Register(rcvr GettyRPCService) error {

	serviceConf := plugins.DefaultProviderServiceConfig()()

	opts := s.opts

	serviceConf.SetService(rcvr.Service())
	serviceConf.SetVersion(rcvr.Version())

	flag := false
	serviceNum := len(opts.ServiceConfList)
	serverNum := len(opts.ConfList)
	for i := 0; i < serviceNum; i++ {
		if opts.ServiceConfList[i].Service() == serviceConf.Service() &&
			opts.ServiceConfList[i].Version() == serviceConf.Version() {

			serviceConf.SetProtocol(opts.ServiceConfList[i].Protocol())
			serviceConf.SetGroup(opts.ServiceConfList[i].Group())

			for j := 0; j < serverNum; j++ {
				if opts.ConfList[j].Protocol == serviceConf.Protocol() {
					rcvrName := reflect.Indirect(reflect.ValueOf(rcvr)).Type().Name()
					svc := &service{
						rcvrType: reflect.TypeOf(rcvr),
						rcvr:     reflect.ValueOf(rcvr),
					}
					if rcvrName == "" {
						s := "rpc.Register: no service name for type " + svc.rcvrType.String()
						log.Error(s)
						return jerrors.New(s)
					}
					if !isExported(rcvrName) {
						s := "rpc.Register: type " + rcvrName + " is not exported"
						log.Error(s)
						return jerrors.New(s)
					}

					svc.name = rcvr.Service() // service name is from 'Service()'
					if _, present := s.srvs[j][svc.name]; present {
						return jerrors.New("rpc: service already defined: " + svc.name)
					}

					// Install the methods
					mts, methods := suitableMethods(svc.rcvrType)
					svc.method = methods

					if len(svc.method) == 0 {
						// To help the user, see if a pointer receiver would work.
						mts, methods = suitableMethods(reflect.PtrTo(svc.rcvrType))
						str := "rpc.Register: type " + rcvrName + " has no exported methods of suitable type"
						if len(methods) != 0 {
							str = "rpc.Register: type " + rcvrName + " has no exported methods of suitable type (" +
								"hint: pass a pointer to value of that type)"
						}
						log.Error(str)

						return jerrors.New(str)
					}

					s.srvs[j][svc.name] = svc

					serviceConf.SetMethods(mts)
					serviceConf.SetPath(opts.ConfList[j].Address())

					err := opts.Registry.Register(serviceConf)
					if err != nil {
						return err
					}
					flag = true
				}
			}
		}
	}

	if !flag {
		return jerrors.Errorf("fail to register Handler{service:%s, version:%s}",
			serviceConf.Service, serviceConf.Version)
	}
	return nil
}

func (s *Server) newSession(session getty.Session) error {
	var (
		ok      bool
		tcpConn *net.TCPConn
	)
	conf := s.opts.ConfList[s.indexOfConfList]

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
	session.SetPkgHandler(NewRpcServerPackageHandler(s, s.srvs[s.indexOfConfList]))
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

func (s *Server) Start() {
	var (
		addr      string
		tcpServer getty.Server
	)

	if len(s.opts.ConfList) == 0 {
		panic("ConfList is nil")
	}

	for i := 0; i < len(s.opts.ConfList); i++ {
		addr = gxnet.HostAddress2(s.opts.ConfList[i].IP, strconv.Itoa(s.opts.ConfList[i].Port))
		tcpServer = getty.NewTCPServer(
			getty.WithLocalAddress(addr),
		)
		s.indexOfConfList = i
		tcpServer.RunEventLoop(s.newSession)
		log.Debug("s bind addr{%s} ok!", addr)
		s.tcpServerList = append(s.tcpServerList, tcpServer)
	}

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
