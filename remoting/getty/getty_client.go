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
	"context"
	"errors"
	"math/rand"
	"sync"
	"sync/atomic"
	"time"
)

import (
	getty "github.com/apache/dubbo-getty"

	"github.com/dubbogo/gost/log/logger"
	gxsync "github.com/dubbogo/gost/sync"

	perrors "github.com/pkg/errors"

	"go.yaml.in/yaml/v4"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	commonCfg "dubbo.apache.org/dubbo-go/v3/common/config"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/global"
	"dubbo.apache.org/dubbo-go/v3/remoting"
	dubbotls "dubbo.apache.org/dubbo-go/v3/tls"
)

var (
	errSessionNotExist   = errors.New("session not exist")
	errClientClosed      = errors.New("client closed")
	errClientReadTimeout = errors.New("maybe the client read timeout or fail to decode tcp stream in Writer.Write")

	clientConf = GetDefaultClientConfig()

	clientGrPool gxsync.GenericTaskPool
)

// it is init client for single protocol.
func initClient(url *common.URL) {
	if url.Protocol == "" {
		return
	}

	// Ensure application config is available via URL attribute
	commonCfg.EnsureApplicationAttribute(url)

	protocolConfRaw, ok := url.GetAttribute(constant.ProtocolConfigKey)
	if !ok || protocolConfRaw == nil {
		logger.Warn("[Remoting][Getty] protocolConfig not found in URL attributes")
		return
	}
	protocolConfMap, ok := protocolConfRaw.(map[string]*global.ProtocolConfig)
	if !ok || protocolConfMap == nil {
		logger.Warn("[Remoting][Getty] protocolConfig assert failed or is nil")
		return
	}

	protocolConf := protocolConfMap[url.Protocol]
	if protocolConf == nil {
		logger.Info("[Remoting][Getty] use default getty client config")
		return
	} else {
		// client tls config
		if tlsConfRaw, ok := url.GetAttribute(constant.TLSConfigKey); ok {
			tlsConf, ok := tlsConfRaw.(*global.TLSConfig)
			if !ok {
				logger.Error("[Remoting][Getty] getty client initialized the TLSConfig configuration failed")
				return
			}
			if dubbotls.IsClientTLSValid(tlsConf) {
				clientConf.SSLEnabled = true
				clientConf.TLSBuilder = &getty.ClientTlsConfigBuilder{
					ClientKeyCertChainPath:        tlsConf.TLSCertFile,
					ClientPrivateKeyPath:          tlsConf.TLSKeyFile,
					ClientTrustCertCollectionPath: tlsConf.CACertFile,
				}
				logger.Info("[Remoting][Getty] getty client initialized the TLSConfig configuration")
			}
		}
		//getty params
		gettyClientConfig := protocolConf.Params
		if gettyClientConfig == nil {
			logger.Debug("[Remoting][Getty] gettyClientConfig is nil")
			return
		}
		gettyClientConfigBytes, err := yaml.Marshal(gettyClientConfig)
		if err != nil {
			panic(err)
		}
		err = yaml.Unmarshal(gettyClientConfigBytes, clientConf)
		if err != nil {
			panic(err)
		}
	}
	if err := clientConf.CheckValidity(); err != nil {
		logger.Warnf("[Remoting][Getty] checkValidity error, err=%v", err)
		return
	}
	setClientGrPool()

	rand.Seed(time.Now().UnixNano())
}

// SetClientConf ClientConf
func SetClientConf(c ClientConfig) {
	clientConf = &c
	err := clientConf.CheckValidity()
	if err != nil {
		logger.Warnf("[Remoting][Getty] clientConfig checkValidity error, err=%v", err)
		return
	}
	setClientGrPool()
}

func setClientGrPool() {
	clientGrPool = gxsync.NewTaskPoolSimple(clientConf.GrPoolSize)
}

// Options : param config
type Options struct {
	ConnectTimeout time.Duration
	RequestTimeout time.Duration
}

// Client : some configuration for network communication.
type Client struct {
	addr           string
	opts           Options
	conf           ClientConfig
	connectMu      sync.Mutex
	closeOnce      sync.Once
	done           chan struct{}
	sslEnabled     bool
	closed         atomic.Bool
	gettyClient    *gettyRPCClient
	gettyClientMux sync.RWMutex
	codec          remoting.Codec
}

// NewClient create client
func NewClient(opt Options) *Client {
	switch {
	case opt.ConnectTimeout == 0:
		opt.ConnectTimeout = 3 * time.Second
		fallthrough
	case opt.RequestTimeout == 0:
		opt.RequestTimeout = 3 * time.Second
	}

	c := &Client{
		opts: opt,
		done: make(chan struct{}),
	}
	return c
}

func (c *Client) SetExchangeClient(client *remoting.ExchangeClient) {
}

// Connect init client and try to connection.
func (c *Client) Connect(url *common.URL) error {
	if c.closed.Load() {
		return errClientClosed
	}
	initClient(url)
	c.conf = *clientConf
	c.sslEnabled = c.conf.SSLEnabled
	// codec
	c.codec = remoting.GetCodec(url.Protocol)
	c.addr = url.Location
	_, _, err := c.selectSession(c.addr)
	if err != nil {
		logger.Errorf("[Remoting][Getty] try to connect server %v failed, err=%v", url.Location, err)
	}
	return err
}

// Close close network connection
func (c *Client) Close() {
	c.closeOnce.Do(func() {
		c.closed.Store(true)
		if c.done != nil {
			close(c.done)
		}
		c.gettyClientMux.Lock()
		client := c.gettyClient
		c.gettyClient = nil
		c.gettyClientMux.Unlock()
		if client != nil {
			client.close()
		}
	})
}

// Request send request
func (c *Client) Request(request *remoting.Request, timeout time.Duration, response *remoting.PendingResponse) error {
	return c.RequestContext(context.Background(), request, timeout, response)
}

// RequestContext sends a request and stops waiting when ctx is canceled.
func (c *Client) RequestContext(ctx context.Context, request *remoting.Request, timeout time.Duration, response *remoting.PendingResponse) error {
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if timeout <= 0 {
		timeout = c.opts.RequestTimeout
	}
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
			logger.Warnf("[Remoting][Getty] start to close the session at request because %d of %d bytes data is sent success. err=%+v", sendLen, totalLen, err)
			go c.Close()
		}
		return perrors.WithStack(err)
	}

	if !request.TwoWay || response.Callback != nil {
		return nil
	}

	timer := time.NewTimer(timeout)
	defer timer.Stop()
	select {
	case <-timer.C:
		return perrors.WithStack(errClientReadTimeout)
	case <-response.Done:
		err = response.Err
	case <-ctx.Done():
		return perrors.WithStack(ctx.Err())
	}

	return perrors.WithStack(err)
}

// IsAvailable returns true if the connection is available, or it can be re-established.
func (c *Client) IsAvailable() bool {
	client, _, err := c.selectSession(c.addr)
	return err == nil &&
		// defensive check
		client != nil
}

func (c *Client) selectSession(addr string) (*gettyRPCClient, getty.Session, error) {
	if c.closed.Load() {
		return nil, nil, errClientClosed
	}

	c.gettyClientMux.RLock()
	client := c.gettyClient
	c.gettyClientMux.RUnlock()
	if client == nil {
		var err error
		client, err = c.getOrCreateGettyClient(addr, newGettyRPCClientConn)
		if err != nil {
			return nil, nil, perrors.WithStack(err)
		}
	}

	if c.closed.Load() {
		return nil, nil, errClientClosed
	}
	return client, client.selectSession(), nil
}

func (c *Client) getOrCreateGettyClient(addr string, newClientConn func(*Client, string) (*gettyRPCClient, error)) (*gettyRPCClient, error) {
	c.connectMu.Lock()
	defer c.connectMu.Unlock()

	if c.closed.Load() {
		return nil, errClientClosed
	}
	c.gettyClientMux.RLock()
	client := c.gettyClient
	c.gettyClientMux.RUnlock()
	if client != nil {
		return client, nil
	}

	client, err := newClientConn(c, addr)
	if err != nil {
		return nil, err
	}

	c.gettyClientMux.Lock()
	if c.closed.Load() {
		c.gettyClientMux.Unlock()
		_ = client.close()
		return nil, errClientClosed
	}
	c.gettyClient = client
	c.gettyClientMux.Unlock()
	return client, nil
}

func (c *Client) transfer(session getty.Session, request *remoting.Request, timeout time.Duration) (int, int, error) {
	totalLen, sendLen, err := session.WritePkg(request, timeout)
	return totalLen, sendLen, perrors.WithStack(err)
}

func (c *Client) resetRpcConn(expected *gettyRPCClient) {
	c.connectMu.Lock()
	defer c.connectMu.Unlock()

	c.gettyClientMux.Lock()
	defer c.gettyClientMux.Unlock()
	if c.gettyClient != expected {
		return
	}
	c.gettyClient = nil
}
