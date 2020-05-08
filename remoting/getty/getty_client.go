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
	"time"

	"github.com/apache/dubbo-go/remoting"
	"github.com/dubbogo/getty"
	"gopkg.in/yaml.v2"

	gxsync "github.com/dubbogo/gost/sync"

	"github.com/apache/dubbo-go/common"
	"github.com/apache/dubbo-go/common/logger"
	"github.com/apache/dubbo-go/config"
	perrors "github.com/pkg/errors"
)

var (
	errInvalidCodecType  = perrors.New("illegal CodecType")
	errInvalidAddress    = perrors.New("remote address invalid or empty")
	errSessionNotExist   = perrors.New("session not exist")
	errClientClosed      = perrors.New("client closed")
	errClientReadTimeout = perrors.New("client read timeout")

	clientConf   *ClientConfig
	clientGrpool *gxsync.TaskPool
)

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

// SetClientConf ...
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
	if clientConf.GrPoolSize > 1 {
		clientGrpool = gxsync.NewTaskPool(gxsync.WithTaskPoolTaskPoolSize(clientConf.GrPoolSize), gxsync.WithTaskPoolTaskQueueLength(clientConf.QueueLen),
			gxsync.WithTaskPoolTaskQueueNumber(clientConf.QueueNumber))
	}
}

// Options ...
type Options struct {
	// connect timeout
	ConnectTimeout time.Duration
}

////AsyncCallbackResponse async response for dubbo
//type AsyncCallbackResponse struct {
//	common.CallbackResponse
//	Opts      Options
//	Cause     error
//	Start     time.Time // invoke(call) start time == write start time
//	ReadStart time.Time // read start time, write duration = ReadStart - Start
//	Reply     interface{}
//}

// Client ...
type Client struct {
	addr            string
	opts            Options
	conf            ClientConfig
	pool            *gettyRPCClientPool
	codec           remoting.Codec
	responseHandler remoting.ResponseHandler
	ExchangeClient  *remoting.ExchangeClient
}

// NewClient ...
func NewClient(opt Options) *Client {
	switch {
	case opt.ConnectTimeout == 0:
		opt.ConnectTimeout = 3 * time.Second
	}

	c := &Client{
		opts: opt,
	}
	return c
}

func (c *Client) SetExchangeClient(client *remoting.ExchangeClient) {
	c.ExchangeClient = client
}
func (c *Client) SetResponseHandler(responseHandler remoting.ResponseHandler) {
	c.responseHandler = responseHandler
}

func (c *Client) Connect(url common.URL) {
	initClient(url.Protocol)
	c.conf = *clientConf
	// new client
	c.pool = newGettyRPCClientConnPool(c, clientConf.PoolSize, time.Duration(int(time.Second)*clientConf.PoolTTL))
	// codec
	c.codec = remoting.GetCodec(url.Protocol)
	c.addr = url.Location
}

func (c *Client) Close() {
	if c.pool != nil {
		c.pool.close()
	}
	c.pool = nil
}

func (c *Client) Request(request *remoting.Request, timeout time.Duration, response *remoting.PendingResponse) error {

	var (
		err     error
		session getty.Session
		conn    *gettyRPCClient
	)
	conn, session, err = c.selectSession(c.addr)
	if err != nil {
		return perrors.WithStack(err)
	}
	if session == nil {
		return errSessionNotExist
	}
	defer func() {
		if err == nil {
			c.pool.put(conn)
			return
		}
		conn.close()
	}()

	if err = c.transfer(session, request, timeout); err != nil {
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

func (c *Client) selectSession(addr string) (*gettyRPCClient, getty.Session, error) {
	rpcClient, err := c.pool.getGettyRpcClient(addr)
	if err != nil {
		return nil, nil, perrors.WithStack(err)
	}
	return rpcClient, rpcClient.selectSession(), nil
}

func (c *Client) heartbeat(session getty.Session) error {
	req := remoting.NewRequest("2.0.2")
	req.TwoWay = true
	req.Event = true
	resp := remoting.NewPendingResponse(req.Id)
	remoting.AddPendingResponse(resp)
	return c.transfer(session, req, 3*time.Second)
}

func (c *Client) transfer(session getty.Session, request *remoting.Request, timeout time.Duration) error {
	err := session.WritePkg(request, timeout)
	//if rsp != nil { // cond2
	//	// cond2 should not merged with cond1. cause the response package may be returned very
	//	// soon and it will be handled by other goroutine.
	//	rsp.ReadStart = time.Now()
	//}
	return perrors.WithStack(err)
}
