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
	"github.com/apache/dubbo-getty"
	gxsync "github.com/dubbogo/gost/sync"
	perrors "github.com/pkg/errors"
	"go.uber.org/atomic"
	"gopkg.in/yaml.v2"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/common/logger"
	"dubbo.apache.org/dubbo-go/v3/config"
	"dubbo.apache.org/dubbo-go/v3/remoting"
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
	gettyClientCreated atomic.Bool
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
