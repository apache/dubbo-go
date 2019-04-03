package jsonrpc

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync/atomic"
	"time"
)

import (
	jerrors "github.com/juju/errors"
)

import (
	"github.com/dubbo/dubbo-go/public"
	"github.com/dubbo/dubbo-go/registry"
)

//////////////////////////////////////////////
// Request
//////////////////////////////////////////////

type Request struct {
	ID          int64
	group       string
	protocol    string
	version     string
	service     string
	method      string
	args        interface{}
	contentType string
}

func (r *Request) ServiceConfig() registry.ServiceConfigIf {
	return &registry.ServiceConfig{
		Protocol: r.protocol,
		Service:  r.service,
		Group:    r.group,
		Version:  r.version,
	}
}

//////////////////////////////////////////////
// HTTP Client
//////////////////////////////////////////////

type HTTPOptions struct {
	HandshakeTimeout time.Duration
	HTTPTimeout      time.Duration
}

var defaultHTTPOptions = HTTPOptions{
	HandshakeTimeout: 3e9,
	HTTPTimeout:      3e9,
}

type HTTPClient struct {
	ID      int64
	options HTTPOptions
}

func NewHTTPClient(opt *HTTPOptions) *HTTPClient {
	if opt == nil {
		opt = &defaultHTTPOptions
	}

	switch {
	case opt.HandshakeTimeout == 0:
		opt.HandshakeTimeout = defaultHTTPOptions.HandshakeTimeout
	case opt.HTTPTimeout == 0:
		opt.HTTPTimeout = defaultHTTPOptions.HTTPTimeout
	}

	t := time.Now()
	return &HTTPClient{
		ID:      int64(uint32(os.Getpid() * t.Second() * t.Nanosecond())),
		options: *opt,
	}
}

func (c *HTTPClient) NewRequest(conf registry.ServiceConfig, method string, args interface{}) Request {
	return Request{
		ID:       atomic.AddInt64(&c.ID, 1),
		group:    conf.Group,
		protocol: conf.Protocol,
		version:  conf.Version,
		service:  conf.Service,
		method:   method,
		args:     args,
	}
}

func (c *HTTPClient) Call(ctx context.Context, service registry.ServiceURL, req Request, rsp interface{}) error {
	// header
	httpHeader := http.Header{}
	httpHeader.Set("Content-Type", "application/json")
	httpHeader.Set("Accept", "application/json")

	reqTimeout := c.options.HTTPTimeout
	if service.Timeout != 0 && service.Timeout < reqTimeout {
		reqTimeout = time.Duration(service.Timeout)
	}
	if reqTimeout <= 0 {
		reqTimeout = 1e8
	}
	httpHeader.Set("Timeout", reqTimeout.String())
	if md, ok := ctx.Value(public.DUBBOGO_CTX_KEY).(map[string]string); ok {
		for k := range md {
			httpHeader.Set(k, md[k])
		}
	}

	// body
	codec := newJsonClientCodec()
	codecData := CodecData{
		ID:     req.ID,
		Method: req.method,
		Args:   req.args,
	}
	reqBody, err := codec.Write(&codecData)
	if err != nil {
		return jerrors.Trace(err)
	}

	rspBody, err := c.Do(service.Location, service.Query.Get("interface"), httpHeader, reqBody)
	if err != nil {
		return jerrors.Trace(err)
	}

	return jerrors.Trace(codec.Read(rspBody, rsp))
}

// !!The high level of complexity and the likelihood that the fasthttp client has not been extensively used
// in production means that you would need to expect a very large benefit to justify the adoption of fasthttp today.
// from: http://big-elephants.com/2016-12/fasthttp-client/
func (c *HTTPClient) Do(addr, path string, httpHeader http.Header, body []byte) ([]byte, error) {
	u := url.URL{Host: strings.TrimSuffix(addr, ":"), Path: path}
	httpReq, err := http.NewRequest("POST", u.String(), bytes.NewBuffer(body))
	if err != nil {
		return nil, jerrors.Trace(err)
	}
	httpReq.Header = httpHeader
	httpReq.Close = true

	reqBuf := bytes.NewBuffer(make([]byte, 0))
	if err := httpReq.Write(reqBuf); err != nil {
		return nil, jerrors.Trace(err)
	}

	tcpConn, err := net.DialTimeout("tcp", addr, c.options.HandshakeTimeout)
	if err != nil {
		return nil, jerrors.Trace(err)
	}
	defer tcpConn.Close()
	setNetConnTimeout := func(conn net.Conn, timeout time.Duration) {
		t := time.Time{}
		if timeout > time.Duration(0) {
			t = time.Now().Add(timeout)
		}

		conn.SetReadDeadline(t)
	}
	setNetConnTimeout(tcpConn, c.options.HTTPTimeout)

	if _, err := reqBuf.WriteTo(tcpConn); err != nil {
		return nil, jerrors.Trace(err)
	}

	httpRsp, err := http.ReadResponse(bufio.NewReader(tcpConn), httpReq)
	if err != nil {
		return nil, jerrors.Trace(err)
	}
	defer httpRsp.Body.Close()

	b, err := ioutil.ReadAll(httpRsp.Body)
	if err != nil {
		return nil, jerrors.Trace(err)
	}

	if httpRsp.StatusCode != http.StatusOK {
		return nil, jerrors.New(fmt.Sprintf("http status:%q, error string:%q", httpRsp.Status, string(b)))
	}

	return b, nil
}
