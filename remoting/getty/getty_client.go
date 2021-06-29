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
	gxsync "github.com/dubbogo/gost/sync"
	perrors "github.com/pkg/errors"
	uatomic "go.uber.org/atomic"
	"gopkg.in/yaml.v2"
)

import (
	"github.com/apache/dubbo-go/common"
	"github.com/apache/dubbo-go/common/constant"
	"github.com/apache/dubbo-go/common/logger"
	"github.com/apache/dubbo-go/config"
	"github.com/apache/dubbo-go/remoting"
)

var (
	errSessionNotExist   = perrors.New("session not exist")
	errClientClosed      = perrors.New("client closed")
	errClientReadTimeout = perrors.New("maybe the client read timeout or fail to decode tcp stream in Writer.Write")

	clientConf   *ClientConfig
	clientGrpool gxsync.GenericTaskPool
)

// it is init client for single protocol.
func initClient(protocol string) {
	if protocol == "" {
		return
	}

	// load clientconfig from consumer_config
	// default use dubbo
	consumerConfig := config.GetConsumerConfig()
	if consumerConfig.ApplicationConfig == nil {
		return
	}
	protocolConf := config.GetConsumerConfig().ProtocolConf
	defaultClientConfig := GetDefaultClientConfig()
	if protocolConf == nil {
		logger.Info("protocol_conf default use dubbo config")
	} else {
		dubboConf := protocolConf.(map[interface{}]interface{})[protocol]
		if dubboConf == nil {
			logger.Warnf("dubboConf is nil")
			return
		}
		dubboConfByte, err := yaml.Marshal(dubboConf)
		if err != nil {
			panic(err)
		}
		err = yaml.Unmarshal(dubboConfByte, &defaultClientConfig)
		if err != nil {
			panic(err)
		}
	}
	clientConf = &defaultClientConfig
	if err := clientConf.CheckValidity(); err != nil {
		logger.Warnf("[CheckValidity] error: %v", err)
		return
	}
	setClientGrpool()
	rand.Seed(time.Now().UnixNano())
}

// Config ClientConf
func SetClientConf(c ClientConfig) {
	clientConf = &c
	err := clientConf.CheckValidity()
	if err != nil {
		logger.Warnf("[ClientConfig CheckValidity] error: %v", err)
		return
	}
	setClientGrpool()
}

func setClientGrpool() {
	clientGrpool = gxsync.NewTaskPoolSimple(clientConf.GrPoolSize)
}

// Options : param config
type Options struct {
	// connect timeout
	// remove request timeout, it will be calculate for every request
	ConnectTimeout time.Duration
	// request timeout
	RequestTimeout time.Duration
}

// Client : some configuration for network communication.
type Client struct {
	addr               string
	opts               Options
	conf               ClientConfig
	mux                sync.RWMutex
	sslEnabled         bool
	clientClosed       bool
	gettyClient        *gettyRPCClient
	gettyClientMux     sync.RWMutex
	gettyClientCreated uatomic.Bool
	codec              remoting.Codec
}

// create client
func NewClient(opt Options) *Client {
	switch {
	case opt.ConnectTimeout == 0:
		opt.ConnectTimeout = 3 * time.Second
		fallthrough
	case opt.RequestTimeout == 0:
		opt.RequestTimeout = 3 * time.Second
	}

	c := &Client{
		opts:         opt,
		clientClosed: false,
	}
	c.gettyClientCreated.Store(false)
	return c
}

func (c *Client) SetExchangeClient(client *remoting.ExchangeClient) {
}

// init client and try to connection.
func (c *Client) Connect(url *common.URL) error {
	initClient(url.Protocol)
	c.conf = *clientConf
	c.sslEnabled = url.GetParamBool(constant.SSL_ENABLED_KEY, false)

	// codec
	c.codec = remoting.GetCodec(url.Protocol)
	c.addr = url.Location
	_, _, err := c.selectSession(c.addr)
	if err != nil {
		logger.Errorf("try to connect server %v failed for : %v", url.Location, err)
	}
	return err
}

// close network connection
func (c *Client) Close() {
	c.mux.Lock()
	client := c.gettyClient
	c.gettyClient = nil
	c.clientClosed = true
	c.mux.Unlock()
	if client != nil {
		client.close()
	}
}

// send request
func (c *Client) Request(request *remoting.Request, timeout time.Duration, response *remoting.PendingResponse) error {
	_, session, err := c.selectSession(c.addr)
	if err != nil {
		return perrors.WithStack(err)
	}
	if session == nil {
		return errSessionNotExist
	}
	var (
		totalLen int
		sendLen  int
	)
	if totalLen, sendLen, err = c.transfer(session, request, timeout); err != nil {
		if sendLen != 0 && totalLen != sendLen {
			logger.Warnf("start to close the session at request because %d of %d bytes data is sent success. err:%+v", sendLen, totalLen, err)
			go c.Close()
		}
		return perrors.WithStack(err)
	}

	if !request.TwoWay || response.Callback != nil {
		return nil
	}

	select {
	case <-getty.GetTimeWheel().After(timeout):
		return perrors.WithStack(errClientReadTimeout)
	case <-response.Done:
		err = response.Err
	}

	return perrors.WithStack(err)
}

// isAvailable returns true if the connection is available, or it can be re-established.
func (c *Client) IsAvailable() bool {
	client, _, err := c.selectSession(c.addr)
	return err == nil &&
		// defensive check
		client != nil
}

func (c *Client) selectSession(addr string) (*gettyRPCClient, getty.Session, error) {
	c.mux.RLock()
	defer c.mux.RUnlock()
	if c.clientClosed {
		return nil, nil, perrors.New("client have been closed")
	}
	if !c.gettyClientCreated.Load() {
		c.gettyClientMux.Lock()
		if c.gettyClient == nil {
			rpcClientConn, rpcErr := newGettyRPCClientConn(c, addr)
			if rpcErr != nil {
				c.gettyClientMux.Unlock()
				return nil, nil, perrors.WithStack(rpcErr)
			}
			c.gettyClientCreated.Store(true)
			c.gettyClient = rpcClientConn
		}
		client := c.gettyClient
		session := c.gettyClient.selectSession()
		c.gettyClientMux.Unlock()
		return client, session, nil
	}
	c.gettyClientMux.RLock()
	client := c.gettyClient
	session := c.gettyClient.selectSession()
	c.gettyClientMux.RUnlock()
	return client, session, nil
}

func (c *Client) transfer(session getty.Session, request *remoting.Request, timeout time.Duration) (int, int, error) {
	totalLen, sendLen, err := session.WritePkg(request, timeout)
	return totalLen, sendLen, perrors.WithStack(err)
}

func (c *Client) resetRpcConn() {
	c.gettyClientMux.Lock()
	c.gettyClient = nil
	c.gettyClientCreated.Store(false)
	c.gettyClientMux.Unlock()

}

type gettyRPCClient struct {
	once sync.Once
	//protocol string
	addr   string
	active int64 // zero, not create or be destroyed

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

	if clientGrpool != nil {
		clientOpts = append(clientOpts, getty.WithClientTaskPool(clientGrpool))
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
