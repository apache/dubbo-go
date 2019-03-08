package jsonrpc

import (
	"bufio"
	"bytes"
	"context"
	"io/ioutil"
	"net"
	"net/http"
	"runtime"
	"runtime/debug"
	"sync"
	"time"
)

import (
	"github.com/AlexStocks/goext/net"
	log "github.com/AlexStocks/log4go"
	"github.com/dubbo/dubbo-go/registry"
	jerrors "github.com/juju/errors"
)

const (
	DefaultMaxSleepTime      = 1 * time.Second // accept中间最大sleep interval
	DefaultHTTPRspBufferSize = 1024
	PathPrefix               = byte('/')
)

// Handler interface represents a Service request handler. It's generated
// by passing any type of public concrete object with methods into server.NewHandler.
// Most will pass in a struct.
//
// Example:
//
//	type Hello struct {}
//
//	func (s *Hello) Method(context, request, response) error {
//		return nil
//	}
//
//  func (s *Hello) Service() string {
//      return "com.youni.service"
//  }
//
//  func (s *Hello) Version() string {
//      return "1.0.0"
//  }

type Handler interface {
	Service() string // Service Interface
	Version() string
}

type Option func(*Options)

type Options struct {
	Registry        registry.Registry
	ConfList        []registry.ServerConfig
	ServiceConfList []registry.ServiceConfig
	Timeout         time.Duration
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

func ConfList(confList []registry.ServerConfig) Option {
	return func(o *Options) {
		o.ConfList = confList
		for i := 0; i < len(o.ConfList); i++ {
			if o.ConfList[i].IP == "" {
				o.ConfList[i].IP, _ = gxnet.GetLocalIP()
			}
		}
	}
}

func ServiceConfList(confList []registry.ServiceConfig) Option {
	return func(o *Options) {
		o.ServiceConfList = confList
	}
}

type Server struct {
	rpc  []*serviceMap
	done chan struct{}
	once sync.Once

	sync.RWMutex
	opts     Options
	handlers map[string]Handler
	wg       sync.WaitGroup
}

func NewServer(opts ...Option) *Server {
	var (
		num int
	)
	options := newOptions(opts...)
	Servers := make([]*serviceMap, len(options.ConfList))
	num = len(options.ConfList)
	for i := 0; i < num; i++ {
		Servers[i] = initServer()
	}
	return &Server{
		opts:     options,
		rpc:      Servers,
		handlers: make(map[string]Handler),
		done:     make(chan struct{}),
	}
}

func (s *Server) handlePkg(rpc *serviceMap, conn net.Conn) {
	defer func() {
		if r := recover(); r != nil {
			log.Warn("connection{local:%v, remote:%v} panic error:%#v, debug stack:%s",
				conn.LocalAddr(), conn.RemoteAddr(), r, string(debug.Stack()))
		}

		conn.Close()
	}()

	setReadTimeout := func(conn net.Conn, timeout time.Duration) {
		t := time.Time{}
		if timeout > time.Duration(0) {
			t = time.Now().Add(timeout)
		}

		conn.SetDeadline(t)
	}

	sendErrorResp := func(header http.Header, body []byte) error {
		rsp := &http.Response{
			Header:        header,
			StatusCode:    500,
			ContentLength: int64(len(body)),
			Body:          ioutil.NopCloser(bytes.NewReader(body)),
		}
		rsp.Header.Del("Content-Type")
		rsp.Header.Del("Content-Length")
		rsp.Header.Del("Timeout")

		rspBuf := bytes.NewBuffer(make([]byte, DefaultHTTPRspBufferSize))
		rspBuf.Reset()
		err := rsp.Write(rspBuf)
		if err != nil {
			return jerrors.Trace(err)
		}
		_, err = rspBuf.WriteTo(conn)
		return jerrors.Trace(err)
	}

	for {
		bufReader := bufio.NewReader(conn)
		r, err := http.ReadRequest(bufReader)
		if err != nil {
			return
		}

		reqBody, err := ioutil.ReadAll(r.Body)
		if err != nil {
			return
		}
		r.Body.Close()

		reqHeader := make(map[string]string)
		for k := range r.Header {
			reqHeader[k] = r.Header.Get(k)
		}
		reqHeader["Path"] = r.URL.Path[1:] // to get service name
		if r.URL.Path[0] != PathPrefix {
			reqHeader["Path"] = r.URL.Path
		}
		reqHeader["HttpMethod"] = r.Method

		httpTimeout := s.Options().Timeout
		contentType := reqHeader["Content-Type"]
		if contentType != "application/json" && contentType != "application/json-rpc" {
			setReadTimeout(conn, httpTimeout)
			r.Header.Set("Content-Type", "text/plain")
			if errRsp := sendErrorResp(r.Header, []byte(jerrors.ErrorStack(err))); errRsp != nil {
				log.Warn("sendErrorResp(header:%#v, error:%s) = error:%s",
					r.Header, jerrors.ErrorStack(err), errRsp)
			}
			return
		}

		ctx := context.Background()
		if len(reqHeader["Timeout"]) > 0 {
			timeout, err := time.ParseDuration(reqHeader["Timeout"])
			if err == nil {
				httpTimeout = timeout
				ctx, _ = context.WithTimeout(ctx, httpTimeout)
			}
			delete(reqHeader, "Timeout")
		}
		setReadTimeout(conn, httpTimeout)

		if err := rpc.serveRequest(ctx, reqHeader, reqBody, conn); err != nil {
			if errRsp := sendErrorResp(r.Header, []byte(jerrors.ErrorStack(err))); errRsp != nil {
				log.Warn("sendErrorResp(header:%#v, error:%s) = error:%s",
					r.Header, jerrors.ErrorStack(err), errRsp)
			}

			log.Info("Unexpected error serving request, closing socket: %v", err)
			return
		}
	}
}

func (s *Server) Options() Options {
	s.RLock()
	defer s.RUnlock()
	return s.opts
}

func (s *Server) Handle(h Handler) error {
	var (
		err         error
		serviceConf registry.ProviderServiceConfig
	)

	opts := s.Options()

	serviceConf.Service = h.Service()
	serviceConf.Version = h.Version()

	flag := 0
	serviceNum := len(opts.ServiceConfList)
	ServerNum := len(opts.ConfList)
	for i := 0; i < serviceNum; i++ {
		if opts.ServiceConfList[i].Service == serviceConf.Service &&
			opts.ServiceConfList[i].Version == serviceConf.Version {

			serviceConf.Protocol = opts.ServiceConfList[i].Protocol
			serviceConf.Group = opts.ServiceConfList[i].Group
			// serviceConf.Version = opts.ServiceConfList[i].Version
			for j := 0; j < ServerNum; j++ {
				if opts.ConfList[j].Protocol == serviceConf.Protocol {
					s.Lock()
					serviceConf.Methods, err = s.rpc[j].register(h)
					s.Unlock()
					if err != nil {
						return err
					}

					serviceConf.Path = opts.ConfList[j].Address()
					err = opts.Registry.Register(serviceConf)
					if err != nil {
						return err
					}
					flag = 1
				}
			}
		}
	}

	if flag == 0 {
		return jerrors.Errorf("fail to register Handler{service:%s, version:%s}",
			serviceConf.Service, serviceConf.Version)
	}

	s.Lock()
	s.handlers[h.Service()] = h
	s.Unlock()

	return nil
}

func accept(listener net.Listener, fn func(net.Conn)) error {
	var (
		err      error
		c        net.Conn
		ok       bool
		ne       net.Error
		tmpDelay time.Duration
	)

	for {
		c, err = listener.Accept()
		if err != nil {
			if ne, ok = err.(net.Error); ok && ne.Temporary() {
				if tmpDelay != 0 {
					tmpDelay <<= 1
				} else {
					tmpDelay = 5 * time.Millisecond
				}
				if tmpDelay > DefaultMaxSleepTime {
					tmpDelay = DefaultMaxSleepTime
				}
				log.Info("http: Accept error: %v; retrying in %v\n", err, tmpDelay)
				time.Sleep(tmpDelay)
				continue
			}
			return jerrors.Trace(err)
		}

		go func() {
			defer func() {
				if r := recover(); r != nil {
					const size = 64 << 10
					buf := make([]byte, size)
					buf = buf[:runtime.Stack(buf, false)]
					log.Error("http: panic serving %v: %v\n%s", c.RemoteAddr(), r, buf)
					c.Close()
				}
			}()

			fn(c)
		}()
	}
}

func (s *Server) Start() error {
	config := s.Options()

	ServerNum := len(config.ConfList)
	for i := 0; i < ServerNum; i++ {
		listener, err := net.Listen("tcp", config.ConfList[i].Address())
		if err != nil {
			return err
		}
		log.Info("rpc server start to listen on %s", listener.Addr())

		s.Lock()
		rpc := s.rpc[i]
		s.Unlock()

		s.wg.Add(1)
		go func(servo *serviceMap) {
			accept(listener, func(conn net.Conn) { s.handlePkg(rpc, conn) })
			s.wg.Done()
		}(rpc)

		s.wg.Add(1)
		go func(servo *serviceMap) { // Server done goroutine
			var err error
			<-s.done               // step1: block to wait for done channel(wait Server.Stop step2)
			err = listener.Close() // step2: and then close listener
			if err != nil {
				log.Warn("listener{addr:%s}.Close() = error{%#v}", listener.Addr(), err)
			}
			s.wg.Done()
		}(rpc)
	}

	return nil
}

func (s *Server) Stop() {
	s.once.Do(func() {
		close(s.done)
		s.wg.Wait()
		if s.opts.Registry != nil {
			s.opts.Registry.Close()
			s.opts.Registry = nil
		}
	})
}
