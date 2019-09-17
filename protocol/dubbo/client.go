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
	"strings"
	"sync"
	"time"
)

import (
	"github.com/apache/dubbo-go-hessian2"
	"github.com/dubbogo/getty"
	"github.com/dubbogo/gost/sync"
	perrors "github.com/pkg/errors"
	"go.uber.org/atomic"
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
	protocolConf := config.GetConsumerConfig().ProtocolConf
	if protocolConf == nil {
		logger.Warnf("protocol_conf is nil")
		return
	}
	dubboConf := protocolConf.(map[interface{}]interface{})[DUBBO]
	if dubboConf == nil {
		logger.Warnf("dubboConf is nil")
		return
	}

	dubboConfByte, err := yaml.Marshal(dubboConf)
	if err != nil {
		panic(err)
	}
	conf := &ClientConfig{}
	err = yaml.Unmarshal(dubboConfByte, conf)
	if err != nil {
		panic(err)
	}

	if err := conf.CheckValidity(); err != nil {
		logger.Warnf("[CheckValidity] error: %v", err)
		return
	}

	clientConf = conf
	setClientGrpool()
}

func SetClientConf(c ClientConfig) {
	clientConf = &c
	err := clientConf.CheckValidity()
	if err != nil {
		logger.Warnf("[ClientConfig CheckValidity] error: %v", err)
		return
	}
	setClientGrpool()
}

func GetClientConf() ClientConfig {
	return *clientConf
}

func setClientGrpool() {
	if clientConf.GrPoolSize > 1 {
		clientGrpool = gxsync.NewTaskPool(gxsync.WithTaskPoolTaskPoolSize(clientConf.GrPoolSize), gxsync.WithTaskPoolTaskQueueLength(clientConf.QueueLen),
			gxsync.WithTaskPoolTaskQueueNumber(clientConf.QueueNumber))
	}
}

type Options struct {
	// connect timeout
	ConnectTimeout time.Duration
	// request timeout
	RequestTimeout time.Duration
}

type CallResponse struct {
	Opts      Options
	Cause     error
	Start     time.Time // invoke(call) start time == write start time
	ReadStart time.Time // read start time, write duration = ReadStart - Start
	Reply     interface{}
}

type AsyncCallback func(response CallResponse)

type Client struct {
	opts     Options
	conf     ClientConfig
	pool     *gettyRPCClientPool
	sequence atomic.Uint64

	pendingResponses *sync.Map
}

func NewClient(opt Options) *Client {

	switch {
	case opt.ConnectTimeout == 0:
		opt.ConnectTimeout = 3e9
		fallthrough
	case opt.RequestTimeout == 0:
		opt.RequestTimeout = 3e9
	}

	c := &Client{
		opts:             opt,
		pendingResponses: new(sync.Map),
		conf:             *clientConf,
	}
	c.pool = newGettyRPCClientConnPool(c, clientConf.PoolSize, time.Duration(int(time.Second)*clientConf.PoolTTL))

	return c
}

type Request struct {
	addr   string
	svcUrl common.URL
	method string
	args   interface{}
	atta   map[string]string
}

func NewRequest(addr string, svcUrl common.URL, method string, args interface{}, atta map[string]string) *Request {
	return &Request{
		addr:   addr,
		svcUrl: svcUrl,
		method: method,
		args:   args,
		atta:   atta,
	}
}

type Response struct {
	reply interface{}
	atta  map[string]string
}

func NewResponse(reply interface{}, atta map[string]string) *Response {
	return &Response{
		reply: reply,
		atta:  atta,
	}
}

// call one way
func (c *Client) CallOneway(request *Request) error {

	return perrors.WithStack(c.call(CT_OneWay, request, NewResponse(nil, nil), nil))
}

// if @response is nil, the transport layer will get the response without notify the invoker.
func (c *Client) Call(request *Request, response *Response) error {

	ct := CT_TwoWay
	if response.reply == nil {
		ct = CT_OneWay
	}

	return perrors.WithStack(c.call(ct, request, response, nil))
}

func (c *Client) AsyncCall(request *Request, callback AsyncCallback, response *Response) error {

	return perrors.WithStack(c.call(CT_TwoWay, request, response, callback))
}

func (c *Client) call(ct CallType, request *Request, response *Response, callback AsyncCallback) error {

	p := &DubboPackage{}
	p.Service.Path = strings.TrimPrefix(request.svcUrl.Path, "/")
	p.Service.Interface = request.svcUrl.GetParam(constant.INTERFACE_KEY, "")
	p.Service.Version = request.svcUrl.GetParam(constant.VERSION_KEY, "")
	p.Service.Method = request.method
	p.Service.Timeout = c.opts.RequestTimeout
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
	defer c.pool.release(conn, err)

	if err = c.transfer(session, p, rsp); err != nil {
		return perrors.WithStack(err)
	}

	if ct == CT_OneWay || callback != nil {
		return nil
	}

	select {
	case <-getty.GetTimeWheel().After(c.opts.RequestTimeout):
		err = errClientReadTimeout
		c.removePendingResponse(SequenceType(rsp.seq))
	case <-rsp.done:
		err = rsp.err
	}

	return perrors.WithStack(err)
}

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
