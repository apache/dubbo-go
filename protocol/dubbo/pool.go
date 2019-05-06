package dubbo

import (
	"fmt"
	"math/rand"
	"net"
	"strings"
	"sync"
	"time"
)

import (
	"github.com/AlexStocks/getty"
	log "github.com/AlexStocks/log4go"
	jerrors "github.com/juju/errors"
)

type gettyRPCClient struct {
	once     sync.Once
	protocol string
	addr     string
	created  int64 // 为0，则说明没有被创建或者被销毁了

	pool *gettyRPCClientPool

	lock        sync.RWMutex
	gettyClient getty.Client
	sessions    []*rpcSession
}

var (
	errClientPoolClosed = jerrors.New("client pool closed")
)

func newGettyRPCClientConn(pool *gettyRPCClientPool, protocol, addr string) (*gettyRPCClient, error) {
	c := &gettyRPCClient{
		protocol: protocol,
		addr:     addr,
		pool:     pool,
		gettyClient: getty.NewTCPClient(
			getty.WithServerAddress(addr),
			getty.WithConnectionNumber((int)(pool.rpcClient.conf.ConnectionNum)),
		),
	}
	c.gettyClient.RunEventLoop(c.newSession)
	idx := 1
	for {
		idx++
		if c.isAvailable() {
			break
		}

		if idx > 5000 {
			return nil, jerrors.New(fmt.Sprintf("failed to create client connection to %s in 5 seconds", addr))
		}
		time.Sleep(1e6)
	}
	log.Info("client init ok")
	c.created = time.Now().Unix()

	return c, nil
}

func (c *gettyRPCClient) newSession(session getty.Session) error {
	var (
		ok      bool
		tcpConn *net.TCPConn
		conf    ClientConfig
	)

	conf = c.pool.rpcClient.conf
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
	session.SetPkgHandler(NewRpcClientPackageHandler(c.pool.rpcClient))
	session.SetEventListener(NewRpcClientHandler(c))
	session.SetRQLen(conf.GettySessionParam.PkgRQSize)
	session.SetWQLen(conf.GettySessionParam.PkgWQSize)
	session.SetReadTimeout(conf.GettySessionParam.tcpReadTimeout)
	session.SetWriteTimeout(conf.GettySessionParam.tcpWriteTimeout)
	session.SetCronPeriod((int)(conf.heartbeatPeriod.Nanoseconds() / 1e6))
	session.SetWaitTime(conf.GettySessionParam.waitTimeout)
	log.Debug("client new session:%s\n", session.Stat())

	return nil
}

func (c *gettyRPCClient) selectSession() getty.Session {
	c.lock.RLock()
	defer c.lock.RUnlock()

	if c.sessions == nil {
		return nil
	}

	count := len(c.sessions)
	if count == 0 {
		return nil
	}
	return c.sessions[rand.Int31n(int32(count))].session
}

func (c *gettyRPCClient) addSession(session getty.Session) {
	log.Debug("add session{%s}", session.Stat())
	if session == nil {
		return
	}

	c.lock.Lock()
	c.sessions = append(c.sessions, &rpcSession{session: session})
	c.lock.Unlock()
}

func (c *gettyRPCClient) removeSession(session getty.Session) {
	if session == nil {
		return
	}

	c.lock.Lock()
	defer c.lock.Unlock()
	if c.sessions == nil {
		return
	}

	for i, s := range c.sessions {
		if s.session == session {
			c.sessions = append(c.sessions[:i], c.sessions[i+1:]...)
			log.Debug("delete session{%s}, its index{%d}", session.Stat(), i)
			break
		}
	}
	log.Info("after remove session{%s}, left session number:%d", session.Stat(), len(c.sessions))
	if len(c.sessions) == 0 {
		c.close() // -> pool.remove(c)
	}
}

func (c *gettyRPCClient) updateSession(session getty.Session) {
	if session == nil {
		return
	}
	c.lock.Lock()
	defer c.lock.Unlock()
	if c.sessions == nil {
		return
	}

	for i, s := range c.sessions {
		if s.session == session {
			c.sessions[i].reqNum++
			break
		}
	}
}

func (c *gettyRPCClient) getClientRpcSession(session getty.Session) (rpcSession, error) {
	var (
		err        error
		rpcSession rpcSession
	)
	c.lock.Lock()
	defer c.lock.Unlock()
	if c.sessions == nil {
		return rpcSession, errClientClosed
	}

	err = errSessionNotExist
	for _, s := range c.sessions {
		if s.session == session {
			rpcSession = *s
			err = nil
			break
		}
	}

	return rpcSession, jerrors.Trace(err)
}

func (c *gettyRPCClient) isAvailable() bool {
	if c.selectSession() == nil {
		return false
	}

	return true
}

func (c *gettyRPCClient) close() error {
	err := jerrors.Errorf("close gettyRPCClient{%#v} again", c)
	c.once.Do(func() {
		// delete @c from client pool
		c.pool.remove(c)
		for _, s := range c.sessions {
			log.Info("close client session{%s, last active:%s, request number:%d}",
				s.session.Stat(), s.session.GetActive().String(), s.reqNum)
			s.session.Close()
		}
		c.gettyClient.Close()
		c.gettyClient = nil
		c.sessions = c.sessions[:0]

		c.created = 0
		err = nil
	})
	return err
}

type gettyRPCClientPool struct {
	rpcClient *Client
	size      int   // []*gettyRPCClient数组的size
	ttl       int64 // 每个gettyRPCClient的有效期时间. pool对象会在getConn时执行ttl检查

	sync.Mutex
	connMap map[string][]*gettyRPCClient // 从[]*gettyRPCClient 可见key是连接地址，而value是对应这个地址的连接数组
}

func newGettyRPCClientConnPool(rpcClient *Client, size int, ttl time.Duration) *gettyRPCClientPool {
	return &gettyRPCClientPool{
		rpcClient: rpcClient,
		size:      size,
		ttl:       int64(ttl.Seconds()),
		connMap:   make(map[string][]*gettyRPCClient),
	}
}

func (p *gettyRPCClientPool) close() {
	p.Lock()
	connMap := p.connMap
	p.connMap = nil
	p.Unlock()
	for _, connArray := range connMap {
		for _, conn := range connArray {
			conn.close()
		}
	}
}

func (p *gettyRPCClientPool) getGettyRpcClient(protocol, addr string) (*gettyRPCClient, error) {

	key := GenerateEndpointAddr(protocol, addr)

	p.Lock()
	defer p.Unlock()
	if p.connMap == nil {
		return nil, errClientPoolClosed
	}

	connArray := p.connMap[key]
	now := time.Now().Unix()

	for len(connArray) > 0 {
		conn := connArray[len(connArray)-1]
		connArray = connArray[:len(connArray)-1]
		p.connMap[key] = connArray

		if d := now - conn.created; d > p.ttl {
			conn.close() // -> pool.remove(c)
			continue
		}

		return conn, nil
	}

	// create new conn
	return newGettyRPCClientConn(p, protocol, addr)
}

func (p *gettyRPCClientPool) release(conn *gettyRPCClient, err error) {
	if conn == nil || conn.created == 0 {
		return
	}
	if err != nil {
		conn.close()
		return
	}

	key := GenerateEndpointAddr(conn.protocol, conn.addr)

	p.Lock()
	defer p.Unlock()
	if p.connMap == nil {
		return
	}

	connArray := p.connMap[key]
	if len(connArray) >= p.size {
		conn.close()
		return
	}
	p.connMap[key] = append(connArray, conn)
}

func (p *gettyRPCClientPool) remove(conn *gettyRPCClient) {
	if conn == nil || conn.created == 0 {
		return
	}

	key := GenerateEndpointAddr(conn.protocol, conn.addr)

	p.Lock()
	defer p.Unlock()
	if p.connMap == nil {
		return
	}

	connArray := p.connMap[key]
	if len(connArray) > 0 {
		for idx, c := range connArray {
			if conn == c {
				p.connMap[key] = append(connArray[:idx], connArray[idx+1:]...)
				break
			}
		}
	}
}

func GenerateEndpointAddr(protocol, addr string) string {
	var builder strings.Builder

	builder.WriteString(protocol)
	builder.WriteString("://")
	builder.WriteString(addr)

	return builder.String()
}
