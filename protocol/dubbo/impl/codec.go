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

package impl

import (
	"bufio"
	"encoding/binary"
)

import (
	hessian "github.com/apache/dubbo-go-hessian2"

	"github.com/dubbogo/gost/log/logger"

	perrors "github.com/pkg/errors"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/remoting"
)

type ProtocolCodec struct {
	reader     *bufio.Reader
	pkgType    PackageType
	bodyLen    int
	serializer Serializer
	headerRead bool
}

func (c *ProtocolCodec) ReadHeader(header *DubboHeader) error {
	var err error
	if c.reader.Size() < HEADER_LENGTH {
		return hessian.ErrHeaderNotEnough
	}
	buf, err := c.reader.Peek(HEADER_LENGTH)
	if err != nil { // this is impossible
		return perrors.WithStack(err)
	}
	_, err = c.reader.Discard(HEADER_LENGTH)
	if err != nil { // this is impossible
		return perrors.WithStack(err)
	}

	//// read header
	if buf[0] != MAGIC_HIGH && buf[1] != MAGIC_LOW {
		return hessian.ErrIllegalPackage
	}

	// Header{serialization id(5 bit), event, two way, req/response}
	if header.SerialID = buf[2] & SERIAL_MASK; header.SerialID == Zero {
		return perrors.Errorf("serialization ID:%v", header.SerialID)
	}

	flag := buf[2] & FLAG_EVENT
	if flag != Zero {
		header.Type |= PackageHeartbeat
	}
	flag = buf[2] & FLAG_REQUEST
	if flag != Zero {
		header.Type |= PackageRequest
		flag = buf[2] & FLAG_TWOWAY
		if flag != Zero {
			header.Type |= PackageRequest_TwoWay
		}
	} else {
		header.Type |= PackageResponse
		header.ResponseStatus = buf[3]
		if header.ResponseStatus != Response_OK {
			header.Type |= PackageResponse_Exception
		}
	}

	// Header{req id}
	header.ID = int64(binary.BigEndian.Uint64(buf[4:]))

	// Header{body len}
	header.BodyLen = int(binary.BigEndian.Uint32(buf[12:]))
	if header.BodyLen < 0 {
		return hessian.ErrIllegalPackage
	}

	c.pkgType = header.Type
	c.bodyLen = header.BodyLen

	if c.reader.Buffered() < c.bodyLen {
		return hessian.ErrBodyNotEnough
	}
	c.headerRead = true
	return perrors.WithStack(err)
}

func (c *ProtocolCodec) EncodeHeader(p DubboPackage) []byte {
	header := p.Header
	bs := make([]byte, 0)
	switch header.Type {
	case PackageHeartbeat:
		if header.ResponseStatus == Zero {
			bs = append(bs, hessian.DubboRequestHeartbeatHeader[:]...)
		} else {
			bs = append(bs, hessian.DubboResponseHeartbeatHeader[:]...)
		}
	case PackageResponse:
		bs = append(bs, hessian.DubboResponseHeaderBytes[:]...)
		if header.ResponseStatus != 0 {
			bs[3] = header.ResponseStatus
		}
	case PackageRequest_TwoWay:
		bs = append(bs, hessian.DubboRequestHeaderBytesTwoWay[:]...)
	}
	bs[2] |= header.SerialID & hessian.SERIAL_MASK
	binary.BigEndian.PutUint64(bs[4:], uint64(header.ID))
	return bs
}

func (c *ProtocolCodec) Encode(p DubboPackage) ([]byte, error) {
	// header
	if c.serializer == nil {
		return nil, perrors.New("serializer should not be nil")
	}
	header := p.Header
	switch header.Type {
	case PackageHeartbeat:
		if header.ResponseStatus == Zero {
			return packRequest(p, c.serializer)
		}
		return packResponse(p, c.serializer)

	case PackageRequest, PackageRequest_TwoWay:
		return packRequest(p, c.serializer)

	case PackageResponse:
		return packResponse(p, c.serializer)

	default:
		return nil, perrors.Errorf("Unrecognized message type: %v", header.Type)
	}
}

func (c *ProtocolCodec) Decode(p *DubboPackage) error {
	if !c.headerRead {
		if err := c.ReadHeader(&p.Header); err != nil {
			return err
		}
	}
	if c.reader.Size() < p.GetBodyLen() {
		return hessian.ErrBodyNotEnough
	}
	body, err := c.reader.Peek(p.GetBodyLen())
	if err != nil {
		return err
	}
	if p.IsResponseWithException() {
		logger.Infof("response with exception: %+v", p.Header)
		decoder := hessian.NewDecoder(body)
		p.Body = &ResponsePayload{}
		exception, err := decoder.Decode()
		if err != nil {
			return perrors.WithStack(err)
		}
		p.Body.(*ResponsePayload).Exception = perrors.Errorf("java exception:%s", exception.(string))
		return nil
	} else if p.IsHeartBeat() {
		// heartbeat no need to unmarshal contents
		return nil
	}
	if c.serializer == nil {
		return perrors.New("Codec serializer is nil")
	}
	if p.IsResponse() {
		p.Body = &ResponsePayload{
			RspObj: remoting.GetPendingResponse(remoting.SequenceType(p.Header.ID)).Reply,
		}
	}
	return c.serializer.Unmarshal(body, p)
}

func (c *ProtocolCodec) SetSerializer(serializer Serializer) {
	c.serializer = serializer
}

func packRequest(p DubboPackage, serializer Serializer) ([]byte, error) {
	var (
		byteArray []byte
		pkgLen    int
	)

	header := p.Header

	//////////////////////////////////////////
	// byteArray
	//////////////////////////////////////////
	// magic
	switch header.Type {
	case PackageHeartbeat:
		byteArray = append(byteArray, DubboRequestHeartbeatHeader[:]...)
	case PackageRequest_TwoWay:
		byteArray = append(byteArray, DubboRequestHeaderBytesTwoWay[:]...)
	default:
		byteArray = append(byteArray, DubboRequestHeaderBytes[:]...)
	}

	// serialization id, two way flag, event, request/response flag
	// SerialID is id of serialization approach in java dubbo
	byteArray[2] |= header.SerialID & SERIAL_MASK
	// request id
	binary.BigEndian.PutUint64(byteArray[4:], uint64(header.ID))

	//////////////////////////////////////////
	// body
	//////////////////////////////////////////
	if p.IsHeartBeat() {
		byteArray = append(byteArray, byte('N'))
		pkgLen = 1
	} else {
		body, err := serializer.Marshal(p)
		if err != nil {
			return nil, err
		}
		pkgLen = len(body)
		if pkgLen > int(DEFAULT_LEN) { // recommand 8M
			logger.Warnf("Data length %d too large, recommand max payload %d. "+
				"Dubbo java can't handle the package whose size is greater than %d!!!", pkgLen, DEFAULT_LEN, DEFAULT_LEN)
		}
		byteArray = append(byteArray, body...)
	}
	binary.BigEndian.PutUint32(byteArray[12:], uint32(pkgLen))
	return byteArray, nil
}

func packResponse(p DubboPackage, serializer Serializer) ([]byte, error) {
	var byteArray []byte
	header := p.Header
	hb := p.IsHeartBeat()

	// magic
	if hb {
		byteArray = append(byteArray, DubboResponseHeartbeatHeader[:]...)
	} else {
		byteArray = append(byteArray, DubboResponseHeaderBytes[:]...)
	}
	// set serialID, identify serialization types, eg: fastjson->6, hessian2->2
	byteArray[2] |= header.SerialID & SERIAL_MASK
	// response status
	if header.ResponseStatus != 0 {
		byteArray[3] = header.ResponseStatus
	}

	// request id
	binary.BigEndian.PutUint64(byteArray[4:], uint64(header.ID))

	// body
	body, err := serializer.Marshal(p)
	if err != nil {
		return nil, err
	}

	pkgLen := len(body)
	if pkgLen > int(DEFAULT_LEN) { // recommand 8M
		logger.Warnf("Data length %d too large, recommand max payload %d. "+
			"Dubbo java can't handle the package whose size is greater than %d!!!", pkgLen, DEFAULT_LEN, DEFAULT_LEN)
	}
	// byteArray{body length}
	binary.BigEndian.PutUint32(byteArray[12:], uint32(pkgLen))
	byteArray = append(byteArray, body...)
	return byteArray, nil
}

func NewDubboCodec(reader *bufio.Reader) *ProtocolCodec {
	s, _ := GetSerializerById(constant.SHessian2)
	return &ProtocolCodec{
		reader:     reader,
		pkgType:    0,
		bodyLen:    0,
		headerRead: false,
		serializer: s,
	}
}
