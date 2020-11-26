/*
 * Licensed to the Apache Software Foundation (ASF) under one or more
 * contributor license agreements.  See the NOTICE file distributed with
 * this work for additional information regarding copyright ownership.
 * The ASF licenses this file to You under the Apache License, Version 2.0
 * (the "License"); you may not use this file except in compliance with
 * the License.  You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package getty

import (
	"crypto/tls"
	"fmt"
	"math/rand"
	"net"
	"sync"
	"sync/atomic"
	"time"
)

import (
	"github.com/apache/dubbo-getty"
	perrors "github.com/pkg/errors"
)

import (
	"github.com/apache/dubbo-go/common/logger"
	"github.com/apache/dubbo-go/config"
)

type gettyRPCClient struct {
	once sync.Once
	//protocol string
	addr   string
	active int64 // zero, not create or be destroyed

	pool *gettyRPCClientPool

	lock        sync.RWMutex
	gettyClient getty.Client
	sessions    []*rpcSession
}

var (
	errClientPoolClosed = perrors.New("client pool closed")
)

func newGettyRPCClientConn(pool *gettyRPCClientPool, addr string) (*gettyRPCClient, error) {
	var (
		gettyClient getty.Client
		sslEnabled  bool
	)
	sslEnabled = pool.sslEnabled
	if sslEnabled {
		gettyClient = getty.NewTCPClient(
			getty.WithServerAddress(addr),
			getty.WithConnectionNumber((int)(pool.rpcClient.conf.ConnectionNum)),
			getty.WithReconnectInterval(pool.rpcClient.conf.ReconnectInterval),
			getty.WithClientSslEnabled(pool.sslEnabled),
			getty.WithClientTlsConfigBuilder(config.GetClientTlsConfigBuilder()),
		)
	} else {
		gettyClient = getty.NewTCPClient(
			getty.WithServerAddress(addr),
			getty.WithConnectionNumber((int)(pool.rpcClient.conf.ConnectionNum)),
			getty.WithReconnectInterval(pool.rpcClient.conf.ReconnectInterval),
		)
	}
	c := &gettyRPCClient{
		addr:        addr,
		pool:        pool,
		gettyClient: gettyClient,
	}
	go c.gettyClient.RunEventLoop(c.newSession)

	idx := 1
	start := time.Now()
	connectTimeout := pool.rpcClient.opts.ConnectTimeout
	for {
		idx++
		if c.isAvailable() {
			break
		}

		if time.Now().Sub(start) > connectTimeout {
			c.gettyClient.Close()
			return nil, perrors.New(fmt.Sprintf("failed to create client connection to %s in %s", addr, connectTimeout))
		}

		interval := time.Millisecond * time.Duration(idx)
		if interval > time.Duration(100e6) {
			interval = 100e6 // 100 ms
		}
		time.Sleep(interval)
	}
	logger.Debug("client init ok")
	c.updateActive(time.Now().Unix())

	return c, nil
}

func (c *gettyRPCClient) updateActive(active int64) {
	atomic.StoreInt64(&c.active, active)
}

func (c *gettyRPCClient) getActive() int64 {
	return atomic.LoadInt64(&c.active)
}

func (c *gettyRPCClient) newSession(session getty.Session) error {
	var (
		ok         bool
		tcpConn    *net.TCPConn
		conf       ClientConfig
		sslEnabled bool
	)
	conf = c.pool.rpcClient.conf
	sslEnabled = c.pool.sslEnabled
	if conf.GettySessionParam.CompressEncoding {
		session.SetCompressType(getty.CompressZip)
	}
	if sslEnabled {
		if _, ok = session.Conn().(*tls.Conn); !ok {
			panic(fmt.Sprintf("%s, session.conn{%#v} is not tls connection\n", session.Stat(), session.Conn()))
		}
		session.SetName(conf.GettySessionParam.SessionName)
		session.SetMaxMsgLen(conf.GettySessionParam.MaxMsgLen)
		session.SetPkgHandler(NewRpcClientPackageHandler(c.pool.rpcClient))
		session.SetEventListener(NewRpcClientHandler(c))
		session.SetWQLen(conf.GettySessionParam.PkgWQSize)
		session.SetReadTimeout(conf.GettySessionParam.tcpReadTimeout)
		session.SetWriteTimeout(conf.GettySessionParam.tcpWriteTimeout)
		session.SetCronPeriod((int)(conf.heartbeatPeriod.Nanoseconds() / 1e6))
		session.SetWaitTime(conf.GettySessionParam.waitTimeout)
		logger.Debugf("client new session:%s\n", session.Stat())
		session.SetTaskPool(clientGrpool)
		return nil
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
	session.SetWQLen(conf.GettySessionParam.PkgWQSize)
	session.SetReadTimeout(conf.GettySessionParam.tcpReadTimeout)
	session.SetWriteTimeout(conf.GettySessionParam.tcpWriteTimeout)
	session.SetCronPeriod((int)(conf.heartbeatPeriod.Nanoseconds() / 1e6))
	session.SetWaitTime(conf.GettySessionParam.waitTimeout)
	logger.Debugf("client new session:%s\n", session.Stat())

	session.SetTaskPool(clientGrpool)

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
	logger.Debugf("add session{%s}", session.Stat())
	if session == nil {
		return
	}

	c.lock.Lock()
	defer c.lock.Unlock()
	if c.sessions == nil {
		c.sessions = make([]*rpcSession, 0, 16)
	}
	c.sessions = append(c.sessions, &rpcSession{session: session})
}

func (c *gettyRPCClient) removeSession(session getty.Session) {
	if session == nil {
		return
	}

	var removeFlag bool
	func() {
		c.lock.Lock()
		defer c.lock.Unlock()
		if c.sessions == nil {
			return
		}

		for i, s := range c.sessions {
			if s.session == session {
				c.sessions = append(c.sessions[:i], c.sessions[i+1:]...)
				logger.Debugf("delete session{%s}, its index{%d}", session.Stat(), i)
				break
			}
		}
		logger.Infof("after remove session{%s}, left session number:%d", session.Stat(), len(c.sessions))
		if len(c.sessions) == 0 {
			removeFlag = true
		}
	}()
	if removeFlag {
		c.pool.safeRemove(c)
		c.close()
	}
}

func (c *gettyRPCClient) updateSession(session getty.Session) {
	if session == nil {
		return
	}

	var rs *rpcSession
	func() {
		c.lock.RLock()
		defer c.lock.RUnlock()
		if c.sessions == nil {
			return
		}

		for i, s := range c.sessions {
			if s.session == session {
				rs = c.sessions[i]
				break
			}
		}
	}()
	if rs != nil {
		rs.AddReqNum(1)
	}
}

func (c *gettyRPCClient) getClientRpcSession(session getty.Session) (rpcSession, error) {
	var (
		err        error
		rpcSession rpcSession
	)
	c.lock.RLock()
	defer c.lock.RUnlock()
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

	return rpcSession, perrors.WithStack(err)
}

func (c *gettyRPCClient) isAvailable() bool {
	if c.selectSession() == nil {
		return false
	}

	return true
}

func (c *gettyRPCClient) close() error {
	closeErr := perrors.Errorf("close gettyRPCClient{%#v} again", c)
	c.once.Do(func() {
		var (
			gettyClient getty.Client
			sessions    []*rpcSession
		)
		func() {
			c.lock.Lock()
			defer c.lock.Unlock()

			gettyClient = c.gettyClient
			c.gettyClient = nil

			sessions = make([]*rpcSession, 0, len(c.sessions))
			for _, s := range c.sessions {
				sessions = append(sessions, s)
			}
			c.sessions = c.sessions[:0]
		}()

		c.updateActive(0)

		go func() {
			if gettyClient != nil {
				gettyClient.Close()
			}
			for _, s := range sessions {
				logger.Infof("close client session{%s, last active:%s, request number:%d}",
					s.session.Stat(), s.session.GetActive().String(), s.GetReqNum())
				s.session.Close()
			}
		}()

		closeErr = nil
	})
	return closeErr
}

type gettyRPCClientPool struct {
	rpcClient  *Client
	size       int   // size of []*gettyRPCClient
	ttl        int64 // ttl of every gettyRPCClient, it is checked when getConn
	sslEnabled bool

	sync.Mutex
	conns []*gettyRPCClient
}

func newGettyRPCClientConnPool(rpcClient *Client, size int, ttl time.Duration) *gettyRPCClientPool {
	return &gettyRPCClientPool{
		rpcClient: rpcClient,
		size:      size,
		ttl:       int64(ttl.Seconds()),
		// init capacity : 2
		conns: make([]*gettyRPCClient, 0, 2),
	}
}

func (p *gettyRPCClientPool) close() {
	p.Lock()
	conns := p.conns
	p.conns = nil
	p.Unlock()
	for _, conn := range conns {
		conn.close()
	}
}

func (p *gettyRPCClientPool) getGettyRpcClient(addr string) (*gettyRPCClient, error) {
	conn, err := p.get()
	if err == nil && conn == nil {
		// create new conn
		rpcClientConn, err := newGettyRPCClientConn(p, addr)
		if err == nil {
			p.put(rpcClientConn)
		}
		return rpcClientConn, perrors.WithStack(err)
	}
	return conn, perrors.WithStack(err)
}

func (p *gettyRPCClientPool) get() (*gettyRPCClient, error) {
	now := time.Now().Unix()

	p.Lock()
	defer p.Unlock()
	if p.conns == nil {
		return nil, errClientPoolClosed
	}
	for num := len(p.conns); num > 0; {
		var conn *gettyRPCClient
		if num != 1 {
			conn = p.conns[rand.Int31n(int32(num))]
		} else {
			conn = p.conns[0]
		}
		// This will recreate gettyRpcClient for remove last position
		//p.conns = p.conns[:len(p.conns)-1]

		if d := now - conn.getActive(); d > p.ttl {
			p.remove(conn)
			go conn.close()
			num = len(p.conns)
			continue
		}
		conn.updateActive(now) //update active time
		return conn, nil
	}
	return nil, nil
}

func (p *gettyRPCClientPool) put(conn *gettyRPCClient) {
	if conn == nil || conn.getActive() == 0 {
		return
	}
	p.Lock()
	defer p.Unlock()
	if p.conns == nil {
		return
	}
	// check whether @conn has existed in p.conns or not.
	for i := range p.conns {
		if p.conns[i] == conn {
			return
		}
	}
	if len(p.conns) >= p.size {
		// delete @conn from client pool
		// p.remove(conn)
		conn.close()
		return
	}
	p.conns = append(p.conns, conn)
}

func (p *gettyRPCClientPool) remove(conn *gettyRPCClient) {
	if conn == nil || conn.getActive() == 0 {
		return
	}

	if p.conns == nil {
		return
	}

	if len(p.conns) > 0 {
		for idx, c := range p.conns {
			if conn == c {
				p.conns = append(p.conns[:idx], p.conns[idx+1:]...)
				break
			}
		}
	}
}

func (p *gettyRPCClientPool) safeRemove(conn *gettyRPCClient) {
	p.Lock()
	defer p.Unlock()

	p.remove(conn)
}
