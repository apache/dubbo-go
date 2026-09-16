/*
 * Licensed to the Apache Software Foundation (ASF) under one or more
 * contributor license agreements.  See the NOTICE file distributed with
 * this work for additional information regarding copyright ownership.
 * The ASF licenses this file to you under the Apache License, Version 2.0
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
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	annotations "google.golang.org/genproto/googleapis/api/annotations"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
	"google.golang.org/protobuf/types/descriptorpb"
	"google.golang.org/protobuf/types/dynamicpb"

	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/global"
	"dubbo.apache.org/dubbo-go/v3/internal/httpbinding"
	"dubbo.apache.org/dubbo-go/v3/protocol/base"
	"dubbo.apache.org/dubbo-go/v3/protocol/result"
	tri "dubbo.apache.org/dubbo-go/v3/protocol/triple/triple_protocol"
)

func TestHTTPTranscodingHandlerSupportsBodyPathQueryAndAdditionalBinding(t *testing.T) {
	_, requestDescriptor, responseDescriptor := registerHTTPTranscodingTestDescriptor(t)
	requestFields := requestDescriptor.Fields()
	responseFields := responseDescriptor.Fields()
	bookDescriptor := requestFields.ByName("book").Message()

	var lastRequest proto.Message
	invoker := &tripleServerTestInvoker{
		url: common.NewURLWithOptions(
			common.WithInterface("triple.http.test.Library"),
			common.WithParamsValue(constant.GroupKey, "g"),
			common.WithParamsValue(constant.VersionKey, "v1"),
		),
		invokeFn: func(_ context.Context, inv base.Invocation) result.Result {
			lastRequest = inv.Arguments()[0].(proto.Message)
			request := lastRequest.ProtoReflect()
			book := request.Get(requestFields.ByName("book")).Message()
			response := dynamicpb.NewMessage(responseDescriptor)
			response.Set(responseFields.ByName("message"), protoreflect.ValueOfString(
				request.Get(requestFields.ByName("name")).String()+":"+
					book.Get(bookDescriptor.Fields().ByName("title")).String()+":"+
					request.Get(requestFields.ByName("count")).String(),
			))
			responseBook := dynamicpb.NewMessage(bookDescriptor)
			responseBook.Set(bookDescriptor.Fields().ByName("title"), protoreflect.ValueOfString("response"))
			response.Set(responseFields.ByName("book"), protoreflect.ValueOfMessage(responseBook.ProtoReflect()))

			res := &result.RPCResult{}
			res.SetResult(response)
			res.SetAttachments(map[string]any{"X-Response-ID": []string{"resp-1"}})
			return res
		},
	}

	server := &Server{
		triServer: tri.NewServer("127.0.0.1:0", nil),
		cfg:       &global.TripleConfig{HTTPTranscoding: &global.HTTPTranscodingConfig{Enabled: true}},
	}
	routes, err := server.buildHTTPTranscodingRoutes("triple.http.test.Library", invoker, &common.ServiceInfo{
		InterfaceName: "triple.http.test.Library",
		Methods: []common.MethodInfo{{
			Name:        "GetBook",
			Type:        constant.CallUnary,
			ReqInitFunc: func() any { return dynamicpb.NewMessage(requestDescriptor) },
		}},
	})
	require.NoError(t, err)
	require.Len(t, routes, 2)

	post := routes[0]
	postRequest := httptest.NewRequest(http.MethodPost, "/v1/authors/alice?count=7", strings.NewReader(`{"title":"book"}`))
	postRequest.Header.Set("Content-Type", "application/json; charset=utf-8")
	postResponse := httptest.NewRecorder()
	post.Handler(postResponse, postRequest, map[string]string{"name": "alice"})
	assert.Equal(t, http.StatusOK, postResponse.Code)
	assert.Equal(t, `"alice:book:7"`, postResponse.Body.String())
	assert.Equal(t, []string{"resp-1"}, postResponse.Header().Values("X-Response-ID"))
	assert.Equal(t, "alice", lastRequest.ProtoReflect().Get(requestFields.ByName("name")).String())

	get := routes[1]
	getRequest := httptest.NewRequest(http.MethodGet, "/v1/books/alice?count=9", nil)
	getResponse := httptest.NewRecorder()
	get.Handler(getResponse, getRequest, map[string]string{"name": "alice"})
	assert.Equal(t, http.StatusOK, getResponse.Code)
	assert.Contains(t, getResponse.Body.String(), `"message":"alice::9"`)
}

func TestHTTPTranscodingHandlerRejectsUnsupportedBodyAndMapsRPCError(t *testing.T) {
	_, requestDescriptor, _ := registerHTTPTranscodingTestDescriptor(t)
	method := common.MethodInfo{
		Name:        "GetBook",
		Type:        constant.CallUnary,
		ReqInitFunc: func() any { return dynamicpb.NewMessage(requestDescriptor) },
	}
	binding := httpbinding.HTTPBinding{
		Method:       http.MethodPost,
		PathTemplate: "/v1/books/{name}",
		Body:         "book",
		PathFields:   []string{"name"},
		ResponseBody: "",
		RPC:          "triple.http.test.Library.GetBook",
	}
	invoker := &tripleServerTestInvoker{
		invokeFn: func(context.Context, base.Invocation) result.Result {
			res := &result.RPCResult{}
			res.SetError(tri.NewError(tri.CodeNotFound, assert.AnError))
			return res
		},
	}
	handler := newHTTPTranscodingHandler(binding, method, invoker)

	unsupported := httptest.NewRequest(http.MethodPost, "/v1/books/alice", strings.NewReader(`{"title":"book"}`))
	unsupported.Header.Set("Content-Type", "text/plain")
	unsupportedResponse := httptest.NewRecorder()
	handler(unsupportedResponse, unsupported, map[string]string{"name": "alice"})
	assert.Equal(t, http.StatusUnsupportedMediaType, unsupportedResponse.Code)

	failed := httptest.NewRequest(http.MethodPost, "/v1/books/alice", strings.NewReader(`{"title":"book"}`))
	failed.Header.Set("Content-Type", "application/json")
	failedResponse := httptest.NewRecorder()
	handler(failedResponse, failed, map[string]string{"name": "alice"})
	assert.Equal(t, http.StatusNotFound, failedResponse.Code)
	assert.Contains(t, failedResponse.Body.String(), `"code":5`)

	limited := newHTTPTranscodingHandler(binding, method, invoker, 4)
	tooLarge := httptest.NewRequest(http.MethodPost, "/v1/books/alice", strings.NewReader(`{"title":"book"}`))
	tooLarge.Header.Set("Content-Type", "application/json")
	tooLargeResponse := httptest.NewRecorder()
	limited(tooLargeResponse, tooLarge, map[string]string{"name": "alice"})
	assert.Equal(t, http.StatusRequestEntityTooLarge, tooLargeResponse.Code)
}

func registerHTTPTranscodingTestDescriptor(t *testing.T) (protoreflect.MethodDescriptor, protoreflect.MessageDescriptor, protoreflect.MessageDescriptor) {
	t.Helper()
	if descriptor, err := protoregistry.GlobalFiles.FindDescriptorByName("triple.http.test.Library.GetBook"); err == nil {
		method := descriptor.(protoreflect.MethodDescriptor)
		requestDescriptor, err := protoregistry.GlobalFiles.FindDescriptorByName("triple.http.test.Request")
		require.NoError(t, err)
		responseDescriptor, err := protoregistry.GlobalFiles.FindDescriptorByName("triple.http.test.Response")
		require.NoError(t, err)
		request := requestDescriptor.(protoreflect.MessageDescriptor)
		response := responseDescriptor.(protoreflect.MessageDescriptor)
		return method, request, response
	}
	options := &descriptorpb.MethodOptions{}
	proto.SetExtension(options, annotations.E_Http, &annotations.HttpRule{
		Pattern:      &annotations.HttpRule_Post{Post: "/v1/authors/{name}"},
		Body:         "book",
		ResponseBody: "message",
		AdditionalBindings: []*annotations.HttpRule{{
			Pattern: &annotations.HttpRule_Get{Get: "/v1/books/{name}"},
		}},
	})
	fileName := "triple/http_test/echo_" + strings.ReplaceAll(t.Name(), "/", "_") + ".proto"
	file, err := protodesc.NewFile(&descriptorpb.FileDescriptorProto{
		Name:    proto.String(fileName),
		Package: proto.String("triple.http.test"),
		Syntax:  proto.String("proto3"),
		Dependency: []string{
			"google/api/annotations.proto",
		},
		MessageType: []*descriptorpb.DescriptorProto{
			{
				Name:  proto.String("Book"),
				Field: []*descriptorpb.FieldDescriptorProto{httpTestField("title", 1, descriptorpb.FieldDescriptorProto_TYPE_STRING, "", false)},
			},
			{
				Name: proto.String("Request"),
				Field: []*descriptorpb.FieldDescriptorProto{
					httpTestField("name", 1, descriptorpb.FieldDescriptorProto_TYPE_STRING, "", false),
					httpTestField("book", 2, descriptorpb.FieldDescriptorProto_TYPE_MESSAGE, ".triple.http.test.Book", false),
					httpTestField("count", 3, descriptorpb.FieldDescriptorProto_TYPE_INT32, "", false),
				},
			},
			{
				Name: proto.String("Response"),
				Field: []*descriptorpb.FieldDescriptorProto{
					httpTestField("message", 1, descriptorpb.FieldDescriptorProto_TYPE_STRING, "", false),
					httpTestField("book", 2, descriptorpb.FieldDescriptorProto_TYPE_MESSAGE, ".triple.http.test.Book", false),
				},
			},
		},
		Service: []*descriptorpb.ServiceDescriptorProto{{
			Name: proto.String("Library"),
			Method: []*descriptorpb.MethodDescriptorProto{{
				Name:       proto.String("GetBook"),
				InputType:  proto.String(".triple.http.test.Request"),
				OutputType: proto.String(".triple.http.test.Response"),
				Options:    options,
			}},
		}},
	}, protoregistry.GlobalFiles)
	require.NoError(t, err)
	require.NoError(t, protoregistry.GlobalFiles.RegisterFile(file))
	service := file.Services().Get(0)
	return service.Methods().Get(0), file.Messages().ByName("Request"), file.Messages().ByName("Response")
}

func httpTestField(name string, number int32, kind descriptorpb.FieldDescriptorProto_Type, typeName string, repeated bool) *descriptorpb.FieldDescriptorProto {
	label := descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL
	if repeated {
		label = descriptorpb.FieldDescriptorProto_LABEL_REPEATED
	}
	field := &descriptorpb.FieldDescriptorProto{Name: proto.String(name), Number: proto.Int32(number), Label: label.Enum(), Type: kind.Enum()}
	if typeName != "" {
		field.TypeName = proto.String(typeName)
	}
	return field
}
