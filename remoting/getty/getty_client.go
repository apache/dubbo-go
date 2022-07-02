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
	"math/rand"
	"sync"
	"time"
)

import (
	getty "github.com/apache/dubbo-getty"

	"github.com/dubbogo/gost/log/logger"
	gxsync "github.com/dubbogo/gost/sync"
	gxtime "github.com/dubbogo/gost/time"

	perrors "github.com/pkg/errors"

	"go.uber.org/atomic"

	"gopkg.in/yaml.v2"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/config"
	"dubbo.apache.org/dubbo-go/v3/remoting"
)

var (
	errSessionNotExist   = perrors.New("session not exist")
	errClientClosed      = perrors.New("client closed")
	errClientReadTimeout = perrors.New("maybe the client read timeout or fail to decode tcp stream in Writer.Write")

	clientConf *ClientConfig

	clientGrPool gxsync.GenericTaskPool
)

// it is init client for single protocol.
func initClient(protocol string) {
	clientConf = GetDefaultClientConfig()
	if protocol == "" {
		return
	}

	// load client config from rootConfig.Protocols
	// default use dubbo
	if config.GetApplicationConfig() == nil {
		return
	}
	if config.GetRootConfig().Protocols == nil {
		return
	}

	protocolConf := config.GetRootConfig().Protocols[protocol]
	if protocolConf == nil {
		logger.Info("use default getty client config")
		return
	} else {
		gettyClientConfig := protocolConf.Params
		if gettyClientConfig == nil {
			logger.Debugf("gettyClientConfig is nil")
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
		logger.Warnf("[CheckValidity] error: %v", err)
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
		logger.Warnf("[ClientConfig CheckValidity] error: %v", err)
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
	addr               string
	opts               Options
	conf               ClientConfig
	mux                sync.RWMutex
	sslEnabled         bool
	clientClosed       bool
	gettyClient        *gettyRPCClient
	gettyClientMux     sync.RWMutex
	gettyClientCreated atomic.Bool
	codec              remoting.Codec
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
		opts:         opt,
		clientClosed: false,
	}
	c.gettyClientCreated.Store(false)
	return c
}

func (c *Client) SetExchangeClient(client *remoting.ExchangeClient) {
}

// Connect init client and try to connection.
func (c *Client) Connect(url *common.URL) error {
	initClient(url.Protocol)
	c.conf = *clientConf
	c.sslEnabled = url.GetParamBool(constant.SslEnabledKey, false)
	// codec
	c.codec = remoting.GetCodec(url.Protocol)
	c.addr = url.Location
	_, _, err := c.selectSession(c.addr)
	if err != nil {
		logger.Errorf("try to connect server %v failed for : %v", url.Location, err)
	}
	return err
}

// Close close network connection
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

// Request send request
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
	case <-gxtime.After(timeout):
		return perrors.WithStack(errClientReadTimeout)
	case <-response.Done:
		err = response.Err
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
