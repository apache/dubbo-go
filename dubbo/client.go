package rpc

import (
	"math/rand"
	"sync"
	"time"
)

import (
	"github.com/AlexStocks/getty"
	"github.com/AlexStocks/goext/sync/atomic"
	jerrors "github.com/juju/errors"
)

var (
	errInvalidCodecType  = jerrors.New("illegal CodecType")
	errInvalidAddress    = jerrors.New("remote address invalid or empty")
	errSessionNotExist   = jerrors.New("session not exist")
	errClientClosed      = jerrors.New("client closed")
	errClientReadTimeout = jerrors.New("client read timeout")
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

type CallOptions struct {
	// request timeout
	RequestTimeout time.Duration
	// response timeout
	ResponseTimeout time.Duration
	Meta            map[interface{}]interface{}
}

type CallOption func(*CallOptions)

func WithCallRequestTimeout(d time.Duration) CallOption {
	return func(o *CallOptions) {
		o.RequestTimeout = d
	}
}

func WithCallResponseTimeout(d time.Duration) CallOption {
	return func(o *CallOptions) {
		o.ResponseTimeout = d
	}
}

func WithCallMeta(k, v interface{}) CallOption {
	return func(o *CallOptions) {
		if o.Meta == nil {
			o.Meta = make(map[interface{}]interface{})
		}
		o.Meta[k] = v
	}
}

type CallResponse struct {
	Opts      CallOptions
	Cause     error
	Start     time.Time // invoke(call) start time == write start time
	ReadStart time.Time // read start time, write duration = ReadStart - Start
	Reply     interface{}
}

type AsyncCallback func(response CallResponse)

type Client struct {
	conf     ClientConfig
	pool     *gettyRPCClientPool
	sequence gxatomic.Uint64

	pendingLock      sync.RWMutex
	pendingResponses map[SequenceType]*PendingResponse
}

func NewClient(conf *ClientConfig) (*Client, error) {
	if err := conf.CheckValidity(); err != nil {
		return nil, jerrors.Trace(err)
	}

	c := &Client{
		pendingResponses: make(map[SequenceType]*PendingResponse),
		conf:             *conf,
	}
	c.pool = newGettyRPCClientConnPool(c, conf.PoolSize, time.Duration(int(time.Second)*conf.PoolTTL))

	return c, nil
}

// call one way
func (c *Client) CallOneway(typ CodecType, addr, service, method string, args interface{}, opts ...CallOption) error {
	var copts CallOptions

	for _, o := range opts {
		o(&copts)
	}

	return jerrors.Trace(c.call(CT_OneWay, typ, addr, service, method, args, nil, nil, copts))
}

// if @reply is nil, the transport layer will get the response without notify the invoker.
func (c *Client) Call(typ CodecType, addr, service, method string, args, reply interface{}, opts ...CallOption) error {
	var copts CallOptions

	for _, o := range opts {
		o(&copts)
	}

	ct := CT_TwoWay
	if reply == nil {
		ct = CT_TwoWayNoReply
	}

	return jerrors.Trace(c.call(ct, typ, addr, service, method, args, reply, nil, copts))
}

func (c *Client) AsyncCall(typ CodecType, addr, service, method string, args interface{},
	callback AsyncCallback, reply interface{}, opts ...CallOption) error {

	var copts CallOptions
	for _, o := range opts {
		o(&copts)
	}

	return jerrors.Trace(c.call(CT_TwoWay, typ, addr, service, method, args, reply, callback, copts))
}

func (c *Client) call(ct CallType, typ CodecType, addr, service, method string,
	args, reply interface{}, callback AsyncCallback, opts CallOptions) error {

	if opts.RequestTimeout == 0 {
		opts.RequestTimeout = c.conf.GettySessionParam.tcpWriteTimeout
	}
	if opts.ResponseTimeout == 0 {
		opts.ResponseTimeout = c.conf.GettySessionParam.tcpReadTimeout
	}
	if !typ.CheckValidity() {
		return errInvalidCodecType
	}

	b := &GettyRPCRequest{}
	b.header.Service = service
	b.header.Method = method
	b.header.CallType = ct
	b.body = args

	var rsp *PendingResponse
	if ct != CT_OneWay {
		rsp = NewPendingResponse()
		rsp.reply = reply
		rsp.callback = callback
		rsp.opts = opts
	}

	var (
		err     error
		session getty.Session
		conn    *gettyRPCClient
	)
	conn, session, err = c.selectSession(typ, addr)
	if err != nil || session == nil {
		return errSessionNotExist
	}
	defer c.pool.release(conn, err)

	if err = c.transfer(session, typ, b, rsp, opts); err != nil {
		return jerrors.Trace(err)
	}

	if ct == CT_OneWay || callback != nil {
		return nil
	}

	select {
	case <-getty.GetTimeWheel().After(opts.ResponseTimeout):
		err = errClientReadTimeout
		c.removePendingResponse(SequenceType(rsp.seq))
	case <-rsp.done:
		err = rsp.err
	}

	return jerrors.Trace(err)
}

func (c *Client) Close() {
	if c.pool != nil {
		c.pool.close()
	}
	c.pool = nil
}

func (c *Client) selectSession(typ CodecType, addr string) (*gettyRPCClient, getty.Session, error) {
	rpcConn, err := c.pool.getConn(typ.String(), addr)
	if err != nil {
		return nil, nil, jerrors.Trace(err)
	}
	return rpcConn, rpcConn.selectSession(), nil
}

func (c *Client) heartbeat(session getty.Session, typ CodecType) error {
	return c.transfer(session, typ, nil, NewPendingResponse(), CallOptions{})
}

func (c *Client) transfer(session getty.Session, typ CodecType, req *GettyRPCRequest,
	rsp *PendingResponse, opts CallOptions) error {

	var (
		sequence uint64
		err      error
		pkg      GettyPackage
	)

	sequence = c.sequence.Add(1)
	pkg.H.Magic = MagicType(gettyPackageMagic)
	pkg.H.LogID = LogIDType(randomID())
	pkg.H.Sequence = SequenceType(sequence)
	pkg.H.Command = gettyCmdHbRequest
	pkg.H.CodecType = typ
	if req != nil {
		pkg.H.Command = gettyCmdRPCRequest
		pkg.B = req
	}

	// cond1
	if rsp != nil {
		rsp.seq = sequence
		c.addPendingResponse(rsp)
	}

	err = session.WritePkg(pkg, opts.RequestTimeout)
	if err != nil {
		c.removePendingResponse(SequenceType(rsp.seq))
	} else if rsp != nil { // cond2
		// cond2 should not merged with cond1. cause the response package may be returned very
		// soon and it will be handled by other goroutine.
		rsp.readStart = time.Now()
	}

	return jerrors.Trace(err)
}

// func (c *Client) PendingResponseCount() int {
// 	c.pendingLock.RLock()
// 	defer c.pendingLock.RUnlock()
// 	return len(c.pendingResponses)
// }

func (c *Client) addPendingResponse(pr *PendingResponse) {
	c.pendingLock.Lock()
	defer c.pendingLock.Unlock()
	c.pendingResponses[SequenceType(pr.seq)] = pr
}

func (c *Client) removePendingResponse(seq SequenceType) *PendingResponse {
	c.pendingLock.Lock()
	defer c.pendingLock.Unlock()
	if c.pendingResponses == nil {
		return nil
	}
	if presp, ok := c.pendingResponses[seq]; ok {
		delete(c.pendingResponses, seq)
		return presp
	}
	return nil
}

// func (c *Client) ClearPendingResponses() map[SequenceType]*PendingResponse {
// 	c.pendingLock.Lock()
// 	defer c.pendingLock.Unlock()
// 	presps := c.pendingResponses
// 	c.pendingResponses = nil
// 	return presps
// }
