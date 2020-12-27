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
	"context"
)

import (
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/hpack"
)

import (
	"github.com/apache/dubbo-go/common"
	"github.com/apache/dubbo-go/remoting"
)

func init() {
	remoting.SetProtocolHeaderHandler(DUBBO3, NewTripleHeaderHandler)
}

// TripleHeader define the h2 header of triple impl
type TripleHeader struct {
	Method         string
	StreamID       uint32
	ContentType    string
	ServiceVersion string
	ServiceGroup   string
	RPCID          string
	TracingID      string
	TracingRPCID   string
	TracingContext string
	ClusterInfo    string
}

func (t *TripleHeader) GetMethod() string {
	return t.Method
}
func (t *TripleHeader) GetStreamID() uint32 {
	return t.StreamID
}

// FieldToCtx parse triple Header that user defined, to ctx of server end
func (t *TripleHeader) FieldToCtx() context.Context {
	ctx := context.WithValue(context.Background(), "tri-service-version", t.ServiceVersion)
	ctx = context.WithValue(ctx, "tri-service-group", t.ServiceGroup)
	ctx = context.WithValue(ctx, "tri-req-id", t.RPCID)
	ctx = context.WithValue(ctx, "tri-trace-traceid", t.TracingID)
	ctx = context.WithValue(ctx, "tri-trace-rpcid", t.TracingRPCID)
	ctx = context.WithValue(ctx, "tri-trace-proto-bin", t.TracingContext)
	ctx = context.WithValue(ctx, "tri-unit-info", t.ClusterInfo)
	return ctx
}

func NewTripleHeaderHandler() remoting.ProtocolHeaderHandler {
	return &TripleHeaderHandler{}
}

// TripleHeaderHandler Handler the change of triple header field and h2 field
type TripleHeaderHandler struct {
}

// WriteHeaderField called when comsumer call server invoke,
// it parse field of url and ctx to HTTP2 Header field, developer must assure "tri-" prefix field be string
// if not, it will cause panic!
func (t TripleHeaderHandler) WriteHeaderField(url *common.URL, ctx context.Context) []hpack.HeaderField {
	headerFields := make([]hpack.HeaderField, 0, 2) // at least :status, content-type will be there if none else.
	headerFields = append(headerFields, hpack.HeaderField{Name: ":method", Value: "POST"})
	headerFields = append(headerFields, hpack.HeaderField{Name: ":scheme", Value: "http"})
	headerFields = append(headerFields, hpack.HeaderField{Name: ":path", Value: url.GetParam(":path", "")}) // added when invoke, parse grpc 'method' to :path
	headerFields = append(headerFields, hpack.HeaderField{Name: ":authority", Value: url.Location})
	headerFields = append(headerFields, hpack.HeaderField{Name: "content-type", Value: url.GetParam("content-type", "application/grpc")})
	headerFields = append(headerFields, hpack.HeaderField{Name: "user-agent", Value: "grpc-go/1.35.0-dev"})
	headerFields = append(headerFields, hpack.HeaderField{Name: "tri-service-version", Value: getCtxVaSave(ctx, "tri-service-version")})
	headerFields = append(headerFields, hpack.HeaderField{Name: "tri-service-group", Value: getCtxVaSave(ctx, "tri-service-group")})
	headerFields = append(headerFields, hpack.HeaderField{Name: "tri-req-id", Value: getCtxVaSave(ctx, "tri-req-id")})
	headerFields = append(headerFields, hpack.HeaderField{Name: "tri-trace-traceid", Value: getCtxVaSave(ctx, "tri-trace-traceid")})
	headerFields = append(headerFields, hpack.HeaderField{Name: "tri-trace-rpcid", Value: getCtxVaSave(ctx, "tri-trace-rpcid")})
	headerFields = append(headerFields, hpack.HeaderField{Name: "tri-trace-proto-bin", Value: getCtxVaSave(ctx, "tri-trace-proto-bin")})
	headerFields = append(headerFields, hpack.HeaderField{Name: "tri-unit-info", Value: getCtxVaSave(ctx, "tri-unit-info")})
	return headerFields
}

func getCtxVaSave(ctx context.Context, field string) string {
	val, ok := ctx.Value(field).(string)
	if ok {
		return val
	}
	return ""
}

// ReadFromH2MetaHeader read meta header field from h2 header, and parse it to ProtocolHeader as user defined
func (t TripleHeaderHandler) ReadFromH2MetaHeader(frame *http2.MetaHeadersFrame) remoting.ProtocolHeader {
	tripleHeader := &TripleHeader{
		StreamID: frame.StreamID,
	}
	for _, f := range frame.Fields {
		switch f.Name {
		case "tri-service-version":
			tripleHeader.ServiceVersion = f.Value
		case "tri-service-group":
			tripleHeader.ServiceGroup = f.Value
		case "tri-req-id":
			tripleHeader.RPCID = f.Value
		case "tri-trace-traceid":
			tripleHeader.TracingID = f.Value
		case "tri-trace-rpcid":
			tripleHeader.TracingRPCID = f.Value
		case "tri-trace-proto-bin":
			tripleHeader.TracingContext = f.Value
		case "tri-unit-info":
			tripleHeader.ClusterInfo = f.Value
		case "content-type":
			tripleHeader.ContentType = f.Value
		case ":path":
			tripleHeader.Method = f.Value
		// todo: usage of these part of fields needs to be discussed later
		//case "grpc-encoding":
		//case "grpc-status":
		//case "grpc-message":
		//case "grpc-status-details-bin":
		//case "grpc-timeout":
		//case ":status":
		//case "grpc-tags-bin":
		//case "grpc-trace-bin":
		default:
		}
	}
	return tripleHeader
}
