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
	"math/rand"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

import (
	hessian "github.com/apache/dubbo-go-hessian2"
	"github.com/dubbogo/getty"
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

func init() {

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
		dubboConf := protocolConf.(map[interface{}]interface{})[DUBBO]
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

// GetClientConf ...
func GetClientConf() ClientConfig {
	return *clientConf
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
	// request timeout
	RequestTimeout time.Duration
}

//AsyncCallbackResponse async response for dubbo
type AsyncCallbackResponse struct {
	common.CallbackResponse
	Opts      Options
	Cause     error
	Start     time.Time // invoke(call) start time == write start time
	ReadStart time.Time // read start time, write duration = ReadStart - Start
	Reply     interface{}
}

// Client ...
type Client struct {
	opts     Options
	conf     ClientConfig
	pool     *gettyRPCClientPool
	sequence uatomic.Uint64

	pendingResponses *sync.Map
}

// NewClient ...
func NewClient(opt Options) *Client {

	switch {
	case opt.ConnectTimeout == 0:
		opt.ConnectTimeout = 3 * time.Second
		fallthrough
	case opt.RequestTimeout == 0:
		opt.RequestTimeout = 3 * time.Second
	}

	// make sure that client request sequence is an odd number
	initSequence := uint64(rand.Int63n(time.Now().UnixNano()))
	if initSequence%2 == 0 {
		initSequence++
	}

	c := &Client{
		opts:             opt,
		pendingResponses: new(sync.Map),
		conf:             *clientConf,
	}
	c.sequence.Store(initSequence)
	c.pool = newGettyRPCClientConnPool(c, clientConf.PoolSize, time.Duration(int(time.Second)*clientConf.PoolTTL))

	return c
}

// Request ...
type Request struct {
	addr   string
	svcUrl common.URL
	method string
	args   interface{}
	atta   map[string]string
}

// NewRequest ...
func NewRequest(addr string, svcUrl common.URL, method string, args interface{}, atta map[string]string) *Request {
	return &Request{
		addr:   addr,
		svcUrl: svcUrl,
		method: method,
		args:   args,
		atta:   atta,
	}
}

// Response ...
type Response struct {
	reply interface{}
	atta  map[string]string
}

// NewResponse ...
func NewResponse(reply interface{}, atta map[string]string) *Response {
	return &Response{
		reply: reply,
		atta:  atta,
	}
}

// CallOneway call one way
func (c *Client) CallOneway(request *Request) error {

	return perrors.WithStack(c.call(CT_OneWay, request, NewResponse(nil, nil), nil))
}

// Call if @response is nil, the transport layer will get the response without notify the invoker.
func (c *Client) Call(request *Request, response *Response) error {

	ct := CT_TwoWay
	if response.reply == nil {
		ct = CT_OneWay
	}

	return perrors.WithStack(c.call(ct, request, response, nil))
}

// AsyncCall ...
func (c *Client) AsyncCall(request *Request, callback common.AsyncCallback, response *Response) error {

	return perrors.WithStack(c.call(CT_TwoWay, request, response, callback))
}

func (c *Client) call(ct CallType, request *Request, response *Response, callback common.AsyncCallback) error {

	p := &DubboPackage{}
	p.Service.Path = strings.TrimPrefix(request.svcUrl.Path, "/")
	p.Service.Interface = request.svcUrl.GetParam(constant.INTERFACE_KEY, "")
	p.Service.Version = request.svcUrl.GetParam(constant.VERSION_KEY, "")
	p.Service.Group = request.svcUrl.GetParam(constant.GROUP_KEY, "")
	p.Service.Method = request.method

	p.Service.Timeout = c.opts.RequestTimeout
	var timeout = request.svcUrl.GetParam(strings.Join([]string{constant.METHOD_KEYS, request.method + constant.RETRIES_KEY}, "."), "")
	if len(timeout) != 0 {
		if t, err := time.ParseDuration(timeout); err == nil {
			p.Service.Timeout = t
		}
	}

	p.Header.SerialID = byte(S_Dubbo)
	p.Body = hessian.NewRequest(request.args, request.atta)

	var rsp *PendingResponse
	if ct != CT_OneWay {
		p.Header.Type = hessian.PackageRequest_TwoWay
		rsp = NewPendingResponse()
		rsp.response = response
		rsp.callback = callback
	} else {
		p.Header.Type = hessian.PackageRequest
	}

	var (
		err     error
		session getty.Session
		conn    *gettyRPCClient
	)
	conn, session, err = c.selectSession(request.addr)
	if err != nil {
		return perrors.WithStack(err)
	}
	if session == nil {
		return errSessionNotExist
	}
	defer func() {
		failNumber := 0
		if err == nil {
			for {
				ok := atomic.CompareAndSwapUint32(&c.pool.pushing, 0, 1)
				if ok {
					c.pool.poolQueue.pushHead(conn)
					c.pool.pushing = 0
					c.pool.ch <- struct{}{}
					return
				}
				failNumber++
				if failNumber%10 == 0 {
					time.Sleep(1e6)
				}
			}
		} else {
			c.pool.ch <- struct{}{}
			conn.close()
		}
	}()

	if err = c.transfer(session, p, rsp); err != nil {
		return perrors.WithStack(err)
	}

	if ct == CT_OneWay || callback != nil {
		return nil
	}

	select {
	case <-getty.GetTimeWheel().After(c.opts.RequestTimeout):
		c.removePendingResponse(SequenceType(rsp.seq))
		return perrors.WithStack(errClientReadTimeout)
	case <-rsp.done:
		err = rsp.err
	}

	return perrors.WithStack(err)
}

// Close ...
func (c *Client) Close() {
	if c.pool != nil {
		c.pool.close()
	}
	c.pool = nil
}

func (c *Client) selectSession(addr string) (*gettyRPCClient, getty.Session, error) {
	rpcClient, err := c.pool.getGettyRpcClient(DUBBO, addr)
	if err != nil {
		return nil, nil, perrors.WithStack(err)
	}
	return rpcClient, rpcClient.selectSession(), nil
}

func (c *Client) heartbeat(session getty.Session) error {
	return c.transfer(session, nil, NewPendingResponse())
}

func (c *Client) transfer(session getty.Session, pkg *DubboPackage,
	rsp *PendingResponse) error {

	var (
		sequence uint64
		err      error
	)

	sequence = c.sequence.Add(1)

	if pkg == nil {
		pkg = &DubboPackage{}
		pkg.Body = hessian.NewRequest([]interface{}{}, nil)
		pkg.Body = []interface{}{}
		pkg.Header.Type = hessian.PackageHeartbeat
		pkg.Header.SerialID = byte(S_Dubbo)
	}
	pkg.Header.ID = int64(sequence)

	// cond1
	if rsp != nil {
		rsp.seq = sequence
		c.addPendingResponse(rsp)
	}

	err = session.WritePkg(pkg, c.opts.RequestTimeout)
	if err != nil {
		c.removePendingResponse(SequenceType(rsp.seq))
	} else if rsp != nil { // cond2
		// cond2 should not merged with cond1. cause the response package may be returned very
		// soon and it will be handled by other goroutine.
		rsp.readStart = time.Now()
	}

	return perrors.WithStack(err)
}

func (c *Client) addPendingResponse(pr *PendingResponse) {
	c.pendingResponses.Store(SequenceType(pr.seq), pr)
}

func (c *Client) removePendingResponse(seq SequenceType) *PendingResponse {
	if c.pendingResponses == nil {
		return nil
	}
	if presp, ok := c.pendingResponses.Load(seq); ok {
		c.pendingResponses.Delete(seq)
		return presp.(*PendingResponse)
	}
	return nil
}
