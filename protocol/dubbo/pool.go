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

	"github.com/dubbogo/getty"

	"github.com/apache/dubbo-go/common/logger"
	perrors "github.com/pkg/errors"
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
	rpcClient     *Client
	maxSize       int   // maxSize of poolQueue
	ttl           int64 // ttl of every gettyRPCClient, it is checked when getConn
	activeNumber  uint32
	chInitialized uint32 // set to 1 when field ch is initialized
	ch            chan struct{}
	closeCh       chan struct{}
	poolQueue     *poolDequeue // store *gettyRPCClient
	pushing       chan struct{}
	sync.RWMutex
}

func newGettyRPCClientConnPool(rpcClient *Client, size int, ttl time.Duration) *gettyRPCClientPool {
	pq := &poolDequeue{
		vals: make([]eface, size),
	}
	return &gettyRPCClientPool{
		rpcClient: rpcClient,
		maxSize:   size,
		ttl:       int64(ttl.Seconds()),
		closeCh:   make(chan struct{}, 0),
		pushing:   make(chan struct{}, 1),
		poolQueue: pq,
	}
}

func (p *gettyRPCClientPool) close() {
	p.Lock()
	connPool := p.poolQueue
	p.poolQueue = nil
	p.Unlock()
	for {
		conn, ok := connPool.popTail()
		if ok {
			c := conn.(*gettyRPCClient)
			c.close()
		} else {
			break
		}
	}
}

func (p *gettyRPCClientPool) lazyInit() {
	// Fast path.
	if atomic.LoadUint32(&p.chInitialized) == 1 {
		return
	}
	// Slow path.
	p.Lock()
	if p.chInitialized == 0 {
		p.ch = make(chan struct{}, p.maxSize)
		for i := 0; i < p.maxSize; i++ {
			p.ch <- struct{}{}
		}
		p.pushing <- struct{}{}
		atomic.StoreUint32(&p.chInitialized, 1)
	}
	p.Unlock()
}

func (p *gettyRPCClientPool) waitVacantConn() error {
	p.lazyInit()
	select {
	case <-p.ch:
		// Additionally check that close chan hasn't expired while we were waiting,
		// because `select` picks a random `case` if several of them are "ready".
		select {
		case <-p.closeCh:
			return errClientPoolClosed
		default:
		}
	case <-p.closeCh:
		return errClientPoolClosed
	}
	return nil
}

func (p *gettyRPCClientPool) getGettyRpcClient(protocol, addr string) (*gettyRPCClient, error) {
	err := p.waitVacantConn()
	if err != nil {
		return nil, err
	}
	conn, err := p.getConnFromPoll()
	if err == nil && conn == nil {
		rpcClientConn, err := newGettyRPCClientConn(p, protocol, addr)
		return rpcClientConn, perrors.WithStack(err)

	}
	return conn, perrors.WithStack(err)
}

func (p *gettyRPCClientPool) getConnFromPoll() (*gettyRPCClient, error) {
	now := time.Now().Unix()
	if p.poolQueue == nil {
		return nil, errClientPoolClosed
	}
	for {
		value, ok := p.poolQueue.popTail()
		if ok {
			conn := value.(*gettyRPCClient)
			if d := now - conn.getActive(); d > p.ttl {
				go conn.close()
				continue
			}
			conn.updateActive(now)
			return conn, nil
		} else {
			break
		}
	}
	return nil, nil
}
