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

package dubbo

import (
	"fmt"
	"math/rand"
	"net"
	"sync"
	"sync/atomic"
	"time"
)

import (
	"github.com/dubbogo/getty"
	perrors "github.com/pkg/errors"
)

import (
	"github.com/apache/dubbo-go/common/logger"
)

type gettyRPCClient struct {
	once     sync.Once
	protocol string
	addr     string
	active   int64 // zero, not create or be destroyed

	pool *gettyRPCClientPool

	lock        sync.RWMutex
	gettyClient getty.Client
	sessions    []*rpcSession
}

var (
	errClientPoolClosed = perrors.New("client pool closed")
)

func newGettyRPCClientConn(pool *gettyRPCClientPool, protocol, addr string) (*gettyRPCClient, error) {
	c := &gettyRPCClient{
		protocol: protocol,
		addr:     addr,
		pool:     pool,
		gettyClient: getty.NewTCPClient(
			getty.WithServerAddress(addr),
			getty.WithConnectionNumber((int)(pool.rpcClient.conf.ConnectionNum)),
			getty.WithReconnectInterval(pool.rpcClient.conf.ReconnectInterval),
		),
	}
	go c.gettyClient.RunEventLoop(c.newSession)
	idx := 1
	times := int(pool.rpcClient.opts.ConnectTimeout / 1e6)
	for {
		idx++
		if c.isAvailable() {
			break
		}

		if idx > times {
			c.gettyClient.Close()
			return nil, perrors.New(fmt.Sprintf("failed to create client connection to %s in %f seconds", addr, float32(times)/1000))
		}
		time.Sleep(1e6)
	}
	logger.Infof("client init ok")
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
	if c.sessions == nil {
		c.sessions = make([]*rpcSession, 0, 16)
	}
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
			logger.Debugf("delete session{%s}, its index{%d}", session.Stat(), i)
			break
		}
	}
	logger.Infof("after remove session{%s}, left session number:%d", session.Stat(), len(c.sessions))
	if len(c.sessions) == 0 {
		c.pool.safeRemove(c)
		c.close()
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
		c.gettyClient.Close()
		c.gettyClient = nil
		for _, s := range c.sessions {
			logger.Infof("close client session{%s, last active:%s, request number:%d}",
				s.session.Stat(), s.session.GetActive().String(), s.reqNum)
			s.session.Close()
		}
		c.sessions = c.sessions[:0]

		c.updateActive(0)
		closeErr = nil
	})
	return closeErr
}

func (c *gettyRPCClient) safeClose() error {
	c.lock.Lock()
	defer c.lock.Unlock()

	return c.close()
}

type gettyRPCClientPool struct {
	rpcClient *Client
	size      int   // size of []*gettyRPCClient
	ttl       int64 // ttl of every gettyRPCClient, it is checked when getConn

	sync.Mutex
	conns []*gettyRPCClient
}

func newGettyRPCClientConnPool(rpcClient *Client, size int, ttl time.Duration) *gettyRPCClientPool {
	return &gettyRPCClientPool{
		rpcClient: rpcClient,
		size:      size,
		ttl:       int64(ttl.Seconds()),
		conns:     make([]*gettyRPCClient, 0, 16),
	}
}

func (p *gettyRPCClientPool) close() {
	p.Lock()
	conns := p.conns
	p.conns = nil
	p.Unlock()
	for _, conn := range conns {
		conn.safeClose()
	}
}

func (p *gettyRPCClientPool) getGettyRpcClient(protocol, addr string) (*gettyRPCClient, error) {

	p.Lock()
	defer p.Unlock()
	if p.conns == nil {
		return nil, errClientPoolClosed
	}

	now := time.Now().Unix()

	for len(p.conns) > 0 {
		conn := p.conns[len(p.conns)-1]
		p.conns = p.conns[:len(p.conns)-1]

		if d := now - conn.getActive(); d > p.ttl {
			p.remove(conn)
			conn.safeClose()
			continue
		}
		conn.updateActive(now) //update active time

		return conn, nil
	}
	// create new conn
	return newGettyRPCClientConn(p, protocol, addr)
}

func (p *gettyRPCClientPool) release(conn *gettyRPCClient, err error) {
	if conn == nil || conn.getActive() == 0 {
		return
	}

	if err != nil {
		conn.safeClose()
		return
	}

	p.Lock()
	defer p.Unlock()

	if p.conns == nil {
		return
	}

	if len(p.conns) >= p.size {
		// delete @conn from client pool
		p.remove(conn)
		conn.safeClose()
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
