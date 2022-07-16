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
	getty "github.com/apache/dubbo-getty"

	"github.com/dubbogo/gost/log/logger"

	perrors "github.com/pkg/errors"
)

import (
	"dubbo.apache.org/dubbo-go/v3/config"
)

type gettyRPCClient struct {
	once   sync.Once
	addr   string // protocol string
	active int64  // zero, not create or be destroyed

	rpcClient *Client

	lock        sync.RWMutex
	gettyClient getty.Client
	sessions    []*rpcSession
}

func newGettyRPCClientConn(rpcClient *Client, addr string) (*gettyRPCClient, error) {
	var (
		gettyClient getty.Client
		sslEnabled  bool
	)
	sslEnabled = rpcClient.sslEnabled
	clientOpts := []getty.ClientOption{
		getty.WithServerAddress(addr),
		getty.WithConnectionNumber((int)(rpcClient.conf.ConnectionNum)),
		getty.WithReconnectInterval(rpcClient.conf.ReconnectInterval),
	}
	if sslEnabled {
		clientOpts = append(clientOpts, getty.WithClientSslEnabled(sslEnabled), getty.WithClientTlsConfigBuilder(config.GetClientTlsConfigBuilder()))
	}

	if clientGrPool != nil {
		clientOpts = append(clientOpts, getty.WithClientTaskPool(clientGrPool))
	}

	gettyClient = getty.NewTCPClient(clientOpts...)
	c := &gettyRPCClient{
		addr:        addr,
		rpcClient:   rpcClient,
		gettyClient: gettyClient,
	}
	go c.gettyClient.RunEventLoop(c.newSession)

	idx := 1
	start := time.Now()
	connectTimeout := rpcClient.opts.ConnectTimeout
	for {
		idx++
		if c.isAvailable() {
			break
		}

		if time.Since(start) > connectTimeout {
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
	conf = c.rpcClient.conf
	sslEnabled = c.rpcClient.sslEnabled
	if conf.GettySessionParam.CompressEncoding {
		session.SetCompressType(getty.CompressZip)
	}
	if sslEnabled {
		if _, ok = session.Conn().(*tls.Conn); !ok {
			panic(fmt.Sprintf("%s, session.conn{%#v} is not tls connection\n", session.Stat(), session.Conn()))
		}
		session.SetName(conf.GettySessionParam.SessionName)
		session.SetMaxMsgLen(conf.GettySessionParam.MaxMsgLen)
		session.SetPkgHandler(NewRpcClientPackageHandler(c.rpcClient))
		session.SetEventListener(NewRpcClientHandler(c))
		session.SetReadTimeout(conf.GettySessionParam.tcpReadTimeout)
		session.SetWriteTimeout(conf.GettySessionParam.tcpWriteTimeout)
		session.SetCronPeriod((int)(conf.heartbeatPeriod.Nanoseconds() / 1e6))
		session.SetWaitTime(conf.GettySessionParam.waitTimeout)
		logger.Debugf("client new session:%s\n", session.Stat())
		return nil
	}
	if tcpConn, ok = session.Conn().(*net.TCPConn); !ok {
		panic(fmt.Sprintf("%s, session.conn{%#v} is not tcp connection\n", session.Stat(), session.Conn()))
	}

	if err := tcpConn.SetNoDelay(conf.GettySessionParam.TcpNoDelay); err != nil {
		logger.Error("tcpConn.SetNoDelay() = error:%v", err)
	}
	if err := tcpConn.SetKeepAlive(conf.GettySessionParam.TcpKeepAlive); err != nil {
		logger.Error("tcpConn.SetKeepAlive() = error:%v", err)
	}
	if conf.GettySessionParam.TcpKeepAlive {
		if err := tcpConn.SetKeepAlivePeriod(conf.GettySessionParam.keepAlivePeriod); err != nil {
			logger.Error("tcpConn.SetKeepAlivePeriod() = error:%v", err)
		}
	}
	if err := tcpConn.SetReadBuffer(conf.GettySessionParam.TcpRBufSize); err != nil {
		logger.Error("tcpConn.SetReadBuffer() = error:%v", err)
	}
	if err := tcpConn.SetWriteBuffer(conf.GettySessionParam.TcpWBufSize); err != nil {
		logger.Error("tcpConn.SetWriteBuffer() = error:%v", err)
	}

	session.SetName(conf.GettySessionParam.SessionName)
	session.SetMaxMsgLen(conf.GettySessionParam.MaxMsgLen)
	session.SetPkgHandler(NewRpcClientPackageHandler(c.rpcClient))
	session.SetEventListener(NewRpcClientHandler(c))
	session.SetReadTimeout(conf.GettySessionParam.tcpReadTimeout)
	session.SetWriteTimeout(conf.GettySessionParam.tcpWriteTimeout)
	session.SetCronPeriod((int)(conf.heartbeatPeriod.Nanoseconds() / 1e6))
	session.SetWaitTime(conf.GettySessionParam.waitTimeout)
	logger.Debugf("client new session:%s\n", session.Stat())
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
		c.rpcClient.resetRpcConn()
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
		err error
		rs  rpcSession
	)
	c.lock.RLock()
	defer c.lock.RUnlock()
	if c.sessions == nil {
		return rs, errClientClosed
	}

	err = errSessionNotExist
	for _, s := range c.sessions {
		if s.session == session {
			rs = *s
			err = nil
			break
		}
	}

	return rs, perrors.WithStack(err)
}

func (c *gettyRPCClient) isAvailable() bool {
	return c.selectSession() != nil
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
			sessions = append(sessions, c.sessions...)
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
