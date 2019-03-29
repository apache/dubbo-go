package dubbo

import (
	"bufio"
	"bytes"
	"fmt"
	"time"
)

import (
	"github.com/dubbogo/hessian2"
	jerrors "github.com/juju/errors"
)

// call type
type CallType int32

const (
	CT_UNKOWN CallType = 0
	CT_OneWay CallType = 1
	CT_TwoWay CallType = 2
)

////////////////////////////////////////////
// dubbo package
////////////////////////////////////////////

type SequenceType int64

type DubboPackage struct {
	Header  hessian.DubboHeader
	Service hessian.Service
	Body    interface{}
	Codec   *hessian.HessianCodec
	Buf     *bytes.Buffer
}

func (p DubboPackage) String() string {
	return fmt.Sprintf("DubboPackage: Header-%v, Service-%v, Body-%v", p.Header, p.Service, p.Body)
}

func (p *DubboPackage) Marshal() (*bytes.Buffer, error) {
	codec := hessian.NewHessianCodec(nil)

	pkg, err := codec.Write(p.Service, p.Header, p.Body)
	if err != nil {
		return nil, jerrors.Trace(err)
	}

	return bytes.NewBuffer(pkg), nil
}

func (p *DubboPackage) Unmarshal(buf *bytes.Buffer, pkgType hessian.PackgeType) error {
	codec := hessian.NewHessianCodec(bufio.NewReader(buf))
	p.Codec = codec
	p.Buf = buf

	// read header
	err := codec.ReadHeader(&p.Header, pkgType)
	return jerrors.Trace(err)
}

func (p *DubboPackage) ReadBody(body interface{}) error {
	err := p.Codec.ReadBody(&body)
	return jerrors.Trace(err)
}

////////////////////////////////////////////
// PendingResponse
////////////////////////////////////////////

type PendingResponse struct {
	seq       uint64
	err       error
	start     time.Time
	readStart time.Time
	callback  AsyncCallback
	reply     interface{}
	opts      CallOptions
	done      chan struct{}
}

func NewPendingResponse() *PendingResponse {
	return &PendingResponse{
		start: time.Now(),
		done:  make(chan struct{}),
	}
}

func (r PendingResponse) GetCallResponse() CallResponse {
	return CallResponse{
		Opts:      r.opts,
		Cause:     r.err,
		Start:     r.start,
		ReadStart: r.readStart,
		Reply:     r.reply,
	}
}
