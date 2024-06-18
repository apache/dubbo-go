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

package triple

import (
	"context"
	"net/http"
	"testing"
)

import (
	"github.com/stretchr/testify/assert"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/protocol"
	"dubbo.apache.org/dubbo-go/v3/protocol/invocation"
	tri "dubbo.apache.org/dubbo-go/v3/protocol/triple/triple_protocol"
)

func Test_parseInvocation(t *testing.T) {
	tests := []struct {
		desc   string
		ctx    func() context.Context
		url    *common.URL
		invo   func() protocol.Invocation
		expect func(t *testing.T, callType string, inRaw []interface{}, methodName string, err error)
	}{
		{
			desc: "miss callType",
			ctx: func() context.Context {
				return context.Background()
			},
			url: common.NewURLWithOptions(),
			invo: func() protocol.Invocation {
				return invocation.NewRPCInvocationWithOptions()
			},
			expect: func(t *testing.T, callType string, inRaw []interface{}, methodName string, err error) {
				assert.NotNil(t, err)
			},
		},
		{
			desc: "wrong callType",
			ctx: func() context.Context {
				return context.Background()
			},
			url: common.NewURLWithOptions(),
			invo: func() protocol.Invocation {
				iv := invocation.NewRPCInvocationWithOptions()
				iv.SetAttribute(constant.CallTypeKey, 1)
				return iv
			},
			expect: func(t *testing.T, callType string, inRaw []interface{}, methodName string, err error) {
				assert.NotNil(t, err)
			},
		},
		{
			desc: "empty methodName",
			ctx: func() context.Context {
				return context.Background()
			},
			url: common.NewURLWithOptions(),
			invo: func() protocol.Invocation {
				iv := invocation.NewRPCInvocationWithOptions()
				iv.SetAttribute(constant.CallTypeKey, constant.CallUnary)
				return iv
			},
			expect: func(t *testing.T, callType string, inRaw []interface{}, methodName string, err error) {
				assert.NotNil(t, err)
			},
		},
	}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			callType, inRaw, methodName, err := parseInvocation(test.ctx(), test.url, test.invo())
			test.expect(t, callType, inRaw, methodName, err)
		})
	}
}

func Test_parseAttachments(t *testing.T) {
	tests := []struct {
		desc   string
		ctx    func() context.Context
		url    *common.URL
		invo   func() protocol.Invocation
		expect func(t *testing.T, ctx context.Context, err error)
	}{
		{
			desc: "url has pre-defined keys in triAttachmentKeys",
			ctx: func() context.Context {
				return context.Background()
			},
			url: common.NewURLWithOptions(
				common.WithInterface("interface"),
				common.WithToken("token"),
			),
			invo: func() protocol.Invocation {
				return invocation.NewRPCInvocationWithOptions()
			},
			expect: func(t *testing.T, ctx context.Context, err error) {
				assert.Nil(t, err)
				header := http.Header(tri.ExtractFromOutgoingContext(ctx))
				assert.NotNil(t, header)
				assert.Equal(t, "interface", header.Get(constant.InterfaceKey))
				assert.Equal(t, "token", header.Get(constant.TokenKey))
			},
		},
		{
			desc: "user passed-in legal attachments",
			ctx: func() context.Context {
				userDefined := make(map[string]interface{})
				userDefined["key1"] = "val1"
				userDefined["key2"] = []string{"key2_1", "key2_2"}
				return context.WithValue(context.Background(), constant.AttachmentKey, userDefined)
			},
			url: common.NewURLWithOptions(),
			invo: func() protocol.Invocation {
				return invocation.NewRPCInvocationWithOptions()
			},
			expect: func(t *testing.T, ctx context.Context, err error) {
				assert.Nil(t, err)
				header := http.Header(tri.ExtractFromOutgoingContext(ctx))
				assert.NotNil(t, header)
				assert.Equal(t, "val1", header.Get("key1"))
				assert.Equal(t, []string{"key2_1", "key2_2"}, header.Values("key2"))
			},
		},
		{
			desc: "user passed-in illegal attachments",
			ctx: func() context.Context {
				userDefined := make(map[string]interface{})
				userDefined["key1"] = 1
				return context.WithValue(context.Background(), constant.AttachmentKey, userDefined)
			},
			url: common.NewURLWithOptions(),
			invo: func() protocol.Invocation {
				return invocation.NewRPCInvocationWithOptions()
			},
			expect: func(t *testing.T, ctx context.Context, err error) {
				assert.NotNil(t, err)
			},
		},
	}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			ctx := test.ctx()
			inv := test.invo()
			parseAttachments(ctx, test.url, inv)
			ctx, err := mergeAttachmentToOutgoing(ctx, inv)
			test.expect(t, ctx, err)
		})
	}
}
