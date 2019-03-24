// Copyright (c) 2016 ~ 2019, Alex Stocks.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package hessian

import (
	"bufio"
	"time"
)

import (
	jerrors "github.com/juju/errors"
)

const (
	Error     PackgeType = 0x01
	Request              = 0x02
	Response             = 0x04
	Heartbeat            = 0x08
)

type PackgeType int

type DubboHeader struct {
	SerialID byte
	Type     PackgeType
	ID       int64
	BodyLen  int
}

type Service struct {
	Path      string
	Interface string
	Version   string
	Target    string // Service Name
	Method    string
	Timeout   time.Duration // request timeout
}

type HessianCodec struct {
	pkgType    PackgeType
	reader     *bufio.Reader
	rspBodyLen int
}

func NewHessianCodec(reader *bufio.Reader) *HessianCodec {
	return &HessianCodec{
		reader: reader,
	}
}

func (h *HessianCodec) Write(service Service, header DubboHeader, body interface{}) ([]byte, error) {
	switch header.Type {
	case Heartbeat, Request:
		return PackRequest(service, header, body)

	case Response:
		return nil, nil

	default:
		return nil, jerrors.Errorf("Unrecognised message type: %v", header.Type)
	}

	return nil, nil
}

func (h *HessianCodec) ReadHeader(header *DubboHeader, pkgType PackgeType) error {
	h.pkgType = pkgType

	switch pkgType {
	case Request:
		return nil

	case Heartbeat, Response:
		buf, err := h.reader.Peek(HEADER_LENGTH)
		if err != nil { // this is impossible
			return jerrors.Trace(err)
		}
		_, err = h.reader.Discard(HEADER_LENGTH)
		if err != nil { // this is impossible
			return jerrors.Trace(err)
		}

		err = UnpackResponseHeaer(buf[:], header)
		if err == ErrJavaException {
			bufSize := h.reader.Buffered()
			if bufSize > 2 {
				expBuf, expErr := h.reader.Peek(bufSize)
				if expErr == nil {
					err = jerrors.Errorf("java exception:%s", string(expBuf[2:bufSize-1]))
				}
			}
		}
		if err != nil {
			return jerrors.Trace(err)
		}
		h.rspBodyLen = header.BodyLen

		return nil

	default:
		return jerrors.Errorf("Unrecognised message type: %v", pkgType)
	}

	return nil
}

func (h *HessianCodec) ReadBody(body interface{}) error {
	switch h.pkgType {
	case Request:
		return nil

	case Heartbeat, Response:
		// remark on 20180611: the heartbeat return is nil
		//if ret == nil {
		//	return jerrors.Errorf("@ret is nil")
		//}

		buf, err := h.reader.Peek(h.rspBodyLen)
		if err == bufio.ErrBufferFull {
			return ErrBodyNotEnough
		}
		if err != nil {
			return jerrors.Trace(err)
		}
		_, err = h.reader.Discard(h.rspBodyLen)
		if err != nil { // this is impossible
			return jerrors.Trace(err)
		}

		if body != nil {
			if err = UnpackResponseBody(buf, body); err != nil {
				return jerrors.Trace(err)
			}
		}
	}

	return nil
}
