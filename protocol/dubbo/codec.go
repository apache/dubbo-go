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
	return fmt.Sprintf("DubboPackage: Header-%v, Path-%v, Body-%v", p.Header, p.Service, p.Body)
}

func (p *DubboPackage) Marshal() (*bytes.Buffer, error) {
	codec := hessian.NewHessianCodec(nil)

	pkg, err := codec.Write(p.Service, p.Header, p.Body)
	if err != nil {
		return nil, jerrors.Trace(err)
	}

	return bytes.NewBuffer(pkg), nil
}

func (p *DubboPackage) Unmarshal(buf *bytes.Buffer, opts ...interface{}) error {
	codec := hessian.NewHessianCodec(bufio.NewReader(buf))

	// read header
	err := codec.ReadHeader(&p.Header)
	if err != nil {
		return jerrors.Trace(err)
	}

	if len(opts) != 0 { // for client
		if client, ok := opts[0].(*Client); ok {

			r := client.pendingResponses[SequenceType(p.Header.ID)]
			if r == nil {
				return fmt.Errorf("pendingResponses[%v] = nil", p.Header.ID)
			}
			p.Body = client.pendingResponses[SequenceType(p.Header.ID)].reply
		} else {
			return fmt.Errorf("opts[0] is not *Client")
		}
	}

	if p.Header.Type&hessian.PackageHeartbeat != 0x00 {
		return nil
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
