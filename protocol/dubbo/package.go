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
	"bufio"
	"bytes"
	"fmt"
	"time"
)

import (
	"github.com/pkg/errors"
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
	codec   *DubboCodec
}

func (p DubboPackage) String() string {
	return fmt.Sprintf("HessianPackage: Header-%v, Path-%v, Body-%v", p.Header, p.Service, p.Body)
}

func (p *DubboPackage) ReadHeader() error {
	return p.codec.ReadHeader(&p.Header)
}

func (p *DubboPackage) Marshal() (*bytes.Buffer, error) {
	if p.codec == nil {
		return nil, errors.New("codec is nil")
	}
	pkg, err := p.codec.Write(*p)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	return bytes.NewBuffer(pkg), nil
}

func (p *DubboPackage) Unmarshal() error {
	return p.codec.Read(p)
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
	p.codec.SetSerializer(serializer)
}

func NewClientResponsePackage(data []byte) *DubboPackage {
	return &DubboPackage{
		Header:  DubboHeader{},
		Service: Service{},
		Body:    &ResponsePayload{},
		Err:     nil,
		codec:   NewDubboCodec(bufio.NewReaderSize(bytes.NewBuffer(data), len(data))),
	}
}

// server side receive request package, just for deserialization
func NewServerRequestPackage(data []byte) *DubboPackage {
	return &DubboPackage{
		Header:  DubboHeader{},
		Service: Service{},
		Body:    make([]interface{}, 7),
		Err:     nil,
		codec:   NewDubboCodec(bufio.NewReaderSize(bytes.NewBuffer(data), len(data))),
	}

}

// client side request package, just for serialization
func NewClientRequestPackage(header DubboHeader, svc Service) *DubboPackage {
	return &DubboPackage{
		Header:  header,
		Service: svc,
		Body:    nil,
		Err:     nil,
		codec:   NewDubboCodec(nil),
	}
}

// server side response package, just for serialization
func NewServerResponsePackage(header DubboHeader) *DubboPackage {
	return &DubboPackage{
		Header: header,
		Body:   nil,
		Err:    nil,
		codec:  NewDubboCodec(nil),
	}
}

func NewDubboPackage(data *bytes.Buffer) *DubboPackage {
	return &DubboPackage{
		Header:  DubboHeader{},
		Service: Service{},
		Body:    nil,
		Err:     nil,
		codec:   NewDubboCodec(bufio.NewReaderSize(data, len(data.Bytes()))),
	}
}