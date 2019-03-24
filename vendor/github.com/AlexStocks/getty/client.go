/******************************************************
# DESC       : getty client
# MAINTAINER : Alex Stocks
# LICENCE    : Apache License 2.0
# EMAIL      : alexstocks@foxmail.com
# MOD        : 2016-09-01 21:32
# FILE       : client.go
******************************************************/

package getty

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"io/ioutil"
	"net"
	"strings"
	"sync"
	"time"
)

import (
	"github.com/AlexStocks/goext/net"
	"github.com/AlexStocks/goext/sync"
	log "github.com/AlexStocks/log4go"
	"github.com/gorilla/websocket"
	jerrors "github.com/juju/errors"
)

const (
	connInterval   = 3e9 // 3s
	connectTimeout = 5e9
	maxTimes       = 10
)

var (
	connectPingPackage = []byte("connect-ping")
)

/////////////////////////////////////////
// getty tcp client
/////////////////////////////////////////

type client struct {
	ClientOptions

	// net
	sync.Mutex
	endPointType EndPointType

	newSession NewSessionCallback
	ssMap      map[Session]gxsync.Empty

	sync.Once
	done chan gxsync.Empty
	wg   sync.WaitGroup
}

func (c *client) init(opts ...ClientOption) {
	for _, opt := range opts {
		opt(&(c.ClientOptions))
	}
}

func newClient(t EndPointType, opts ...ClientOption) *client {
	c := &client{
		endPointType: t,
		done:         make(chan gxsync.Empty),
	}

	c.init(opts...)

	if c.number <= 0 || c.addr == "" {
		panic(fmt.Sprintf("client type:%s, @connNum:%d, @serverAddr:%s", t, c.number, c.addr))
	}

	c.ssMap = make(map[Session]gxsync.Empty, c.number)

	return c
}

// NewTcpClient function builds a tcp client.
func NewTCPClient(opts ...ClientOption) Client {
	return newClient(TCP_CLIENT, opts...)
}

// NewUdpClient function builds a connected udp client
func NewUDPClient(opts ...ClientOption) Client {
	return newClient(UDP_CLIENT, opts...)
}

// NewWsClient function builds a ws client.
func NewWSClient(opts ...ClientOption) Client {
	c := newClient(WS_CLIENT, opts...)

	if !strings.HasPrefix(c.addr, "ws://") {
		panic(fmt.Sprintf("the prefix @serverAddr:%s is not ws://", c.addr))
	}

	return c
}

// NewWSSClient function builds a wss client.
func NewWSSClient(opts ...ClientOption) Client {
	c := newClient(WSS_CLIENT, opts...)

	if c.cert == "" {
		panic(fmt.Sprintf("@cert:%s", c.cert))
	}
	if !strings.HasPrefix(c.addr, "wss://") {
		panic(fmt.Sprintf("the prefix @serverAddr:%s is not wss://", c.addr))
	}

	return c
}

func (c client) EndPointType() EndPointType {
	return c.endPointType
}

func (c *client) dialTCP() Session {
	var (
		err  error
		conn net.Conn
	)

	for {
		if c.IsClosed() {
			return nil
		}
		conn, err = net.DialTimeout("tcp", c.addr, connectTimeout)
		if err == nil && gxnet.IsSameAddr(conn.RemoteAddr(), conn.LocalAddr()) {
			conn.Close()
			err = errSelfConnect
		}
		if err == nil {
			return newTCPSession(conn, c)
		}

		log.Info("net.DialTimeout(addr:%s, timeout:%v) = error{%s}", c.addr, jerrors.ErrorStack(err))
		// time.Sleep(connInterval)
		<-wheel.After(connInterval)
	}
}

func (c *client) dialUDP() Session {
	var (
		err       error
		conn      *net.UDPConn
		localAddr *net.UDPAddr
		peerAddr  *net.UDPAddr
		length    int
		buf       []byte
	)

	buf = make([]byte, 128)
	localAddr = &net.UDPAddr{IP: net.IPv4zero, Port: 0}
	peerAddr, _ = net.ResolveUDPAddr("udp", c.addr)
	for {
		if c.IsClosed() {
			return nil
		}
		conn, err = net.DialUDP("udp", localAddr, peerAddr)
		if err == nil && gxnet.IsSameAddr(conn.RemoteAddr(), conn.LocalAddr()) {
			conn.Close()
			err = errSelfConnect
		}
		if err != nil {
			log.Warn("net.DialTimeout(addr:%s, timeout:%v) = error{%s}", c.addr, jerrors.ErrorStack(err))
			// time.Sleep(connInterval)
			<-wheel.After(connInterval)
			continue
		}

		// check connection alive by write/read action
		conn.SetWriteDeadline(wheel.Now().Add(1e9))
		if length, err = conn.Write(connectPingPackage[:]); err != nil {
			conn.Close()
			log.Warn("conn.Write(%s) = {length:%d, err:%s}", string(connectPingPackage), length, jerrors.ErrorStack(err))
			// time.Sleep(connInterval)
			<-wheel.After(connInterval)
			continue
		}
		conn.SetReadDeadline(wheel.Now().Add(1e9))
		length, err = conn.Read(buf)
		if netErr, ok := jerrors.Cause(err).(net.Error); ok && netErr.Timeout() {
			err = nil
		}
		if err != nil {
			log.Info("conn{%#v}.Read() = {length:%d, err:%s}", conn, length, jerrors.ErrorStack(err))
			conn.Close()
			// time.Sleep(connInterval)
			<-wheel.After(connInterval)
			continue
		}
		//if err == nil {
		return newUDPSession(conn, c)
		//}
	}
}

func (c *client) dialWS() Session {
	var (
		err    error
		dialer websocket.Dialer
		conn   *websocket.Conn
		ss     Session
	)

	dialer.EnableCompression = true
	for {
		if c.IsClosed() {
			return nil
		}
		conn, _, err = dialer.Dial(c.addr, nil)
		log.Info("websocket.dialer.Dial(addr:%s) = error:%s", c.addr, jerrors.ErrorStack(err))
		if err == nil && gxnet.IsSameAddr(conn.RemoteAddr(), conn.LocalAddr()) {
			conn.Close()
			err = errSelfConnect
		}
		if err == nil {
			ss = newWSSession(conn, c)
			if ss.(*session).maxMsgLen > 0 {
				conn.SetReadLimit(int64(ss.(*session).maxMsgLen))
			}

			return ss
		}

		log.Info("websocket.dialer.Dial(addr:%s) = error:%s", c.addr, jerrors.ErrorStack(err))
		// time.Sleep(connInterval)
		<-wheel.After(connInterval)
	}
}

func (c *client) dialWSS() Session {
	var (
		err      error
		root     *x509.Certificate
		roots    []*x509.Certificate
		certPool *x509.CertPool
		config   *tls.Config
		dialer   websocket.Dialer
		conn     *websocket.Conn
		ss       Session
	)

	dialer.EnableCompression = true

	config = &tls.Config{
		InsecureSkipVerify: true,
	}

	if c.cert != "" {
		certPEMBlock, err := ioutil.ReadFile(c.cert)
		if err != nil {
			panic(fmt.Sprintf("ioutil.ReadFile(cert:%s) = error{%s}", c.cert, jerrors.ErrorStack(err)))
		}

		var cert tls.Certificate
		for {
			var certDERBlock *pem.Block
			certDERBlock, certPEMBlock = pem.Decode(certPEMBlock)
			if certDERBlock == nil {
				break
			}
			if certDERBlock.Type == "CERTIFICATE" {
				cert.Certificate = append(cert.Certificate, certDERBlock.Bytes)
			}
		}
		config.Certificates = make([]tls.Certificate, 1)
		config.Certificates[0] = cert
	}

	certPool = x509.NewCertPool()
	for _, c := range config.Certificates {
		roots, err = x509.ParseCertificates(c.Certificate[len(c.Certificate)-1])
		if err != nil {
			panic(fmt.Sprintf("error parsing server's root cert: %s\n", jerrors.ErrorStack(err)))
		}
		for _, root = range roots {
			certPool.AddCert(root)
		}
	}
	config.InsecureSkipVerify = true
	config.RootCAs = certPool

	// dialer.EnableCompression = true
	dialer.TLSClientConfig = config
	for {
		if c.IsClosed() {
			return nil
		}
		conn, _, err = dialer.Dial(c.addr, nil)
		if err == nil && gxnet.IsSameAddr(conn.RemoteAddr(), conn.LocalAddr()) {
			conn.Close()
			err = errSelfConnect
		}
		if err == nil {
			ss = newWSSession(conn, c)
			if ss.(*session).maxMsgLen > 0 {
				conn.SetReadLimit(int64(ss.(*session).maxMsgLen))
			}
			ss.SetName(defaultWSSSessionName)

			return ss
		}

		log.Info("websocket.dialer.Dial(addr:%s) = error{%s}", c.addr, jerrors.ErrorStack(err))
		// time.Sleep(connInterval)
		<-wheel.After(connInterval)
	}
}

func (c *client) dial() Session {
	switch c.endPointType {
	case TCP_CLIENT:
		return c.dialTCP()
	case UDP_CLIENT:
		return c.dialUDP()
	case WS_CLIENT:
		return c.dialWS()
	case WSS_CLIENT:
		return c.dialWSS()
	}

	return nil
}

func (c *client) sessionNum() int {
	var num int

	c.Lock()
	for s := range c.ssMap {
		if s.IsClosed() {
			delete(c.ssMap, s)
		}
	}
	num = len(c.ssMap)
	c.Unlock()

	return num
}

func (c *client) connect() {
	var (
		err error
		ss  Session
	)

	for {
		ss = c.dial()
		if ss == nil {
			// client has been closed
			break
		}
		err = c.newSession(ss)
		if err == nil {
			// ss.RunEventLoop()
			ss.(*session).run()
			c.Lock()
			c.ssMap[ss] = gxsync.Empty{}
			c.Unlock()
			break
		}
		// don't distinguish between tcp connection and websocket connection. Because
		// gorilla/websocket/conn.go:(Conn)Close also invoke net.Conn.Close()
		ss.Conn().Close()
	}
}

// there are two methods to keep connection pool. the first approch is like
// redigo's lazy connection pool(https://github.com/gomodule/redigo/blob/master/redis/pool.go:),
// in which you should apply testOnBorrow to check alive of the connection.
// the second way is as follows. @RunEventLoop detects the aliveness of the connection
// in regular time interval.
// the active method maybe overburden the cpu slightly.
// however, you can get a active tcp connection very quickly.
func (c *client) RunEventLoop(newSession NewSessionCallback) {
	c.Lock()
	c.newSession = newSession
	c.Unlock()

	c.wg.Add(1)
	// a for-loop goroutine to make sure the connection is valid
	go func() {
		var num, max, times int
		defer c.wg.Done()

		c.Lock()
		max = c.number
		c.Unlock()
		// log.Info("maximum client connection number:%d", max)
		for {
			if c.IsClosed() {
				log.Warn("client{peer:%s} goroutine exit now.", c.addr)
				break
			}

			num = c.sessionNum()
			// log.Info("current client connction number:%d", num)
			if max <= num {
				times++
				if maxTimes < times {
					times = maxTimes
				}
				// time.Sleep(time.Duration(int64(times) * connInterval))
				<-wheel.After(time.Duration(int64(times) * connInterval))
				continue
			}
			times = 0
			c.connect()
			//if c.endPointType == UDP_CLIENT {
			//	break
			//}
			// time.Sleep(c.interval) // build c.number connections asap
		}
	}()
}

func (c *client) stop() {
	select {
	case <-c.done:
		return
	default:
		c.Once.Do(func() {
			close(c.done)
			c.Lock()
			for s := range c.ssMap {
				s.Close()
			}
			c.ssMap = nil
			c.Unlock()
		})
	}
}

func (c *client) IsClosed() bool {
	select {
	case <-c.done:
		return true
	default:
		return false
	}
}

func (c *client) Close() {
	c.stop()
	c.wg.Wait()
}
