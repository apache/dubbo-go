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

// serial ID
type SerialID byte

const (
	S_Dubbo SerialID = 2
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
	Err     error
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

	// read header
	err := codec.ReadHeader(&p.Header, pkgType)
	if err != nil {
		return jerrors.Trace(err)
	}

	// read body
	err = codec.ReadBody(p.Body)
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
