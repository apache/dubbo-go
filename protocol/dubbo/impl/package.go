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
	"bytes"
	"fmt"
	"time"
)

import (
	"github.com/pkg/errors"
)

type PackageType int

// enum part
const (
	PackageError              = PackageType(0x01)
	PackageRequest            = PackageType(0x02)
	PackageResponse           = PackageType(0x04)
	PackageHeartbeat          = PackageType(0x08)
	PackageRequest_TwoWay     = PackageType(0x10)
	PackageResponse_Exception = PackageType(0x20)
	PackageType_BitSize       = 0x2f
)

type DubboHeader struct {
	SerialID       byte
	Type           PackageType
	ID             int64
	BodyLen        int
	ResponseStatus byte
}

// Service defines service instance
type Service struct {
	Path      string
	Interface string
	Group     string
	Version   string
	Method    string
	Timeout   time.Duration // request timeout
}

type DubboPackage struct {
	Header  DubboHeader
	Service Service
	Body    interface{}
	Err     error
	Codec   *ProtocolCodec
}

func (p DubboPackage) String() string {
	return fmt.Sprintf("HessianPackage: Header-%v, Path-%v, Body-%v", p.Header, p.Service, p.Body)
}

func (p *DubboPackage) ReadHeader() error {
	return p.Codec.ReadHeader(&p.Header)
}

func (p *DubboPackage) Marshal() (*bytes.Buffer, error) {
	if p.Codec == nil {
		return nil, errors.New("Codec is nil")
	}
	pkg, err := p.Codec.Encode(*p)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	return bytes.NewBuffer(pkg), nil
}

func (p *DubboPackage) Unmarshal() error {
	if p.Codec == nil {
		return errors.New("Codec is nil")
	}
	return p.Codec.Decode(p)
}

func (p DubboPackage) IsHeartBeat() bool {
	return p.Header.Type&PackageHeartbeat != 0
}

func (p DubboPackage) IsRequest() bool {
	return p.Header.Type&(PackageRequest_TwoWay|PackageRequest) != 0
}

func (p DubboPackage) IsResponse() bool {
	return p.Header.Type == PackageResponse
}

func (p DubboPackage) IsResponseWithException() bool {
	flag := PackageResponse | PackageResponse_Exception
	return p.Header.Type&flag == flag
}

func (p DubboPackage) GetBodyLen() int {
	return p.Header.BodyLen
}

func (p DubboPackage) GetLen() int {
	return HEADER_LENGTH + p.Header.BodyLen
}

func (p DubboPackage) GetBody() interface{} {
	return p.Body
}

func (p *DubboPackage) SetBody(body interface{}) {
	p.Body = body
}

func (p *DubboPackage) SetHeader(header DubboHeader) {
	p.Header = header
}

func (p *DubboPackage) SetService(svc Service) {
	p.Service = svc
}

func (p *DubboPackage) SetID(id int64) {
	p.Header.ID = id
}

func (p DubboPackage) GetHeader() DubboHeader {
	return p.Header
}

func (p DubboPackage) GetService() Service {
	return p.Service
}

func (p *DubboPackage) SetResponseStatus(status byte) {
	p.Header.ResponseStatus = status
}

func (p *DubboPackage) SetSerializer(serializer Serializer) {
	p.Codec.SetSerializer(serializer)
}

func NewDubboPackage(data *bytes.Buffer) *DubboPackage {
	var codec *ProtocolCodec
	if data == nil {
		codec = NewDubboCodec(nil)
	} else {
		codec = NewDubboCodec(bufio.NewReaderSize(data, len(data.Bytes())))
	}
	return &DubboPackage{
		Header:  DubboHeader{},
		Service: Service{},
		Body:    nil,
		Err:     nil,
		Codec:   codec,
	}
}
