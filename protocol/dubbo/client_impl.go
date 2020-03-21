package dubbo

import (
	"bufio"
	"bytes"
	"github.com/apache/dubbo-go/protocol/dubbo/impl/remoting"

	"math/rand"
	"strings"
	"sync"
	"time"
)

import (
	hessian "github.com/apache/dubbo-go-hessian2"
	"github.com/dubbogo/getty"
	perrors "github.com/pkg/errors"
	"go.uber.org/atomic"
)

import (
	"github.com/apache/dubbo-go/common"
	"github.com/apache/dubbo-go/common/constant"
	"github.com/apache/dubbo-go/common/logger"
	"github.com/apache/dubbo-go/protocol/dubbo/impl"
)

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
	conf     remoting.ClientConfig
	pool     *gettyRPCClientPool
	sequence atomic.Uint64

	pendingResponses *sync.Map
	codec            impl.DubboCodec
}

// NewClient ...
func NewClient(clientConf *remoting.ClientConfig, opt Options) *Client {

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
		codec:            impl.DubboCodec{},
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
	// NOTE: compatible with old versions
	if svcUrl.GetParam(constant.SERIALIZATION_KEY, "") == "" {
		svcUrl.SetParam(constant.SERIALIZATION_KEY, constant.DEFAULT_SERIALIZATION)
	}
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

	return perrors.WithStack(c.call(impl.CT_OneWay, request, NewResponse(nil, nil), nil))
}

// Call if @response is nil, the transport layer will get the response without notify the invoker.
func (c *Client) Call(request *Request, response *Response) error {

	ct := impl.CT_TwoWay
	if response.reply == nil {
		ct = impl.CT_OneWay
	}

	return perrors.WithStack(c.call(ct, request, response, nil))
}

// AsyncCall ...
func (c *Client) AsyncCall(request *Request, callback common.AsyncCallback, response *Response) error {

	return perrors.WithStack(c.call(impl.CT_TwoWay, request, response, callback))
}

func (c *Client) call(ct impl.CallType, request *Request, response *Response, callback common.AsyncCallback) error {
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
		if err == nil {
			c.pool.put(conn)
			return
		}
		conn.close()
	}()

	var rsp *PendingResponse
	svc := impl.Service{}
	header := impl.DubboHeader{}
	svc.Path = strings.TrimPrefix(request.svcUrl.Path, "/")
	svc.Interface = request.svcUrl.GetParam(constant.INTERFACE_KEY, "")
	svc.Version = request.svcUrl.GetParam(constant.VERSION_KEY, "")
	svc.Group = request.svcUrl.GetParam(constant.GROUP_KEY, "")
	svc.Method = request.method
	svc.Timeout = c.opts.RequestTimeout
	var timeout = request.svcUrl.GetParam(strings.Join([]string{constant.METHOD_KEYS, request.method + constant.RETRIES_KEY}, "."), "")
	if len(timeout) != 0 {
		if t, err := time.ParseDuration(timeout); err == nil {
			svc.Timeout = t
		}
	}
	p := NewClientRequestPackage(header, svc)

	serialization := request.svcUrl.GetParam(constant.SERIALIZATION_KEY, c.conf.Serialization)
	if serialization == constant.HESSIAN2_SERIALIZATION {
		p.Header.SerialID = constant.S_Hessian2
	} else if serialization == constant.PROTOBUF_SERIALIZATION {
		p.Header.SerialID = constant.S_Proto
	}
	p.SetBody(impl.NewRequestPayload(request.args, request.atta))

	if err := loadSerializer(p); err != nil {
		return err
	}

	if ct != impl.CT_OneWay {
		p.Header.Type = impl.PackageRequest_TwoWay
		rsp = NewPendingResponse()
		rsp.response = response
		rsp.callback = callback
	} else {
		p.Header.Type = impl.PackageRequest
	}
	if err = c.transfer(session, p, rsp); err != nil {
		return perrors.WithStack(err)
	}

	if ct == impl.CT_OneWay || callback != nil {
		return nil
	}

	select {
	case <-getty.GetTimeWheel().After(c.opts.RequestTimeout):
		c.removePendingResponse(impl.SequenceType(rsp.seq))
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

func (c *Client) transfer(session getty.Session, pkg *impl.DubboPackage,
	rsp *PendingResponse) error {

	var (
		sequence uint64
		err      error
	)

	sequence = c.sequence.Add(1)

	if pkg == nil {
		// make heartbeat package
		header := impl.DubboHeader{
			Type:     impl.PackageHeartbeat,
			SerialID: constant.S_Hessian2,
		}
		pkg = NewClientRequestPackage(header, impl.Service{})
		// SetBody
		reqPayload := impl.NewRequestPayload([]interface{}{}, nil)
		pkg.SetBody(reqPayload)
		// set serializer
		if err := loadSerializer(pkg); err != nil {
			return err
		}
	}
	pkg.SetID(int64(sequence))

	// cond1
	if rsp != nil {
		rsp.seq = sequence
		c.addPendingResponse(rsp)
	}

	err = session.WritePkg(pkg, c.opts.RequestTimeout)
	if err != nil {
		c.removePendingResponse(impl.SequenceType(rsp.seq))
	} else if rsp != nil { // cond2
		// cond2 should not merged with cond1. cause the response package may be returned very
		// soon and it will be handled by other goroutine.
		rsp.readStart = time.Now()
	}

	return perrors.WithStack(err)
}

func (c *Client) addPendingResponse(pr *PendingResponse) {
	c.pendingResponses.Store(impl.SequenceType(pr.seq), pr)
}

func (c *Client) removePendingResponse(seq impl.SequenceType) *PendingResponse {
	if c.pendingResponses == nil {
		return nil
	}
	if presp, ok := c.pendingResponses.Load(seq); ok {
		c.pendingResponses.Delete(seq)
		return presp.(*PendingResponse)
	}
	return nil
}

// PendingResponse ...
type PendingResponse struct {
	seq       uint64
	err       error
	start     time.Time
	readStart time.Time
	callback  common.AsyncCallback
	response  *Response
	done      chan struct{}
}

// NewPendingResponse ...
func NewPendingResponse() *PendingResponse {
	return &PendingResponse{
		start:    time.Now(),
		response: &Response{},
		done:     make(chan struct{}),
	}
}

// GetCallResponse ...
func (r PendingResponse) GetCallResponse() common.CallbackResponse {
	return AsyncCallbackResponse{
		Cause:     r.err,
		Start:     r.start,
		ReadStart: r.readStart,
		Reply:     r.response,
	}
}

// client side request package, just for serialization
func NewClientRequestPackage(header impl.DubboHeader, svc impl.Service) *impl.DubboPackage {
	return &impl.DubboPackage{
		Header:  header,
		Service: svc,
		Body:    nil,
		Err:     nil,
		Codec:   impl.NewDubboCodec(nil),
	}
}

////////////////////////////////////////////
// RpcClientPackageHandler
////////////////////////////////////////////

// RpcClientPackageHandler ...
type RpcClientPackageHandler struct {
	client *Client
}

// NewRpcClientPackageHandler ...
func NewRpcClientPackageHandler(client *Client) *RpcClientPackageHandler {
	return &RpcClientPackageHandler{client: client}
}

func (p *RpcClientPackageHandler) Read(ss getty.Session, data []byte) (interface{}, int, error) {
	pkg := NewClientResponsePackage(data)
	if err := pkg.ReadHeader(); err != nil {
		originErr := perrors.Cause(err)
		if originErr == hessian.ErrHeaderNotEnough || originErr == hessian.ErrBodyNotEnough {
			return nil, 0, nil
		}
		logger.Errorf("[RpcClientPackageHandler.Read] ss:%+v, len(@data):%d) = error:%+v ", ss, len(data), err)
		return nil, 0, perrors.WithStack(err)
	}
	if pkg.IsHeartBeat() {
		// heartbeat package doesn't need deserialize
		return pkg, pkg.GetLen(), nil
	}

	if err := loadSerializer(pkg); err != nil {
		return nil, 0, err
	}

	// load response
	pendingRsp, ok := p.client.pendingResponses.Load(impl.SequenceType(pkg.GetHeader().ID))
	if !ok {
		return nil, 0, perrors.Errorf("client.GetPendingResopnse(%v) = nil", pkg.GetHeader().ID)
	}
	// set package body
	body := impl.NewResponsePayload(pendingRsp.(*PendingResponse).response.reply, nil, nil)
	pkg.SetBody(body)
	err := pkg.Unmarshal()
	if err != nil {
		return nil, 0, perrors.WithStack(err)
	}
	resp := pkg.Body.(*impl.ResponsePayload)
	pkg.Err = resp.Exception
	pkg.Body = NewResponse(resp.RspObj, resp.Attachments)
	return pkg, pkg.GetLen(), nil
}

func (p *RpcClientPackageHandler) Write(ss getty.Session, pkg interface{}) ([]byte, error) {
	req, ok := pkg.(*impl.DubboPackage)
	if !ok {
		return nil, perrors.New("invalid rpc request")
	}
	buf, err := req.Marshal()
	if err != nil {
		logger.Warnf("binary.Write(req{%#v}) = err{%#v}", req, perrors.WithStack(err))
		return nil, perrors.WithStack(err)
	}
	return buf.Bytes(), nil
}

func NewClientResponsePackage(data []byte) *impl.DubboPackage {
	return &impl.DubboPackage{
		Header:  impl.DubboHeader{},
		Service: impl.Service{},
		Body:    &impl.ResponsePayload{},
		Err:     nil,
		Codec:   impl.NewDubboCodec(bufio.NewReaderSize(bytes.NewBuffer(data), len(data))),
	}
}
