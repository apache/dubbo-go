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
	"testing"
	"time"
)

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common/constant"
)

func TestDubboPackage_MarshalAndUnmarshal(t *testing.T) {
	pkg := NewDubboPackage(nil)
	pkg.Body = []any{"a"}
	pkg.Header.Type = PackageHeartbeat
	pkg.Header.SerialID = constant.SHessian2
	pkg.Header.ID = 10086
	pkg.SetSerializer(HessianSerializer{})

	// heartbeat
	data, err := pkg.Marshal()
	require.NoError(t, err)

	pkgres := NewDubboPackage(data)
	pkgres.SetSerializer(HessianSerializer{})

	pkgres.Body = []any{}
	err = pkgres.Unmarshal()
	require.NoError(t, err)
	assert.Equal(t, PackageHeartbeat|PackageRequest|PackageRequest_TwoWay, pkgres.Header.Type)
	assert.Equal(t, constant.SHessian2, pkgres.Header.SerialID)
	assert.Equal(t, int64(10086), pkgres.Header.ID)
	assert.Empty(t, pkgres.Body.([]any))

	// request
	pkg.Header.Type = PackageRequest
	pkg.Service.Interface = "Service"
	pkg.Service.Path = "path"
	pkg.Service.Version = "2.6"
	pkg.Service.Method = "Method"
	pkg.Service.Timeout = time.Second
	data, err = pkg.Marshal()
	require.NoError(t, err)

	pkgres = NewDubboPackage(data)
	pkgres.SetSerializer(HessianSerializer{})
	pkgres.Body = make([]any, 7)
	err = pkgres.Unmarshal()
	reassembleBody := pkgres.GetBody().(map[string]any)
	require.NoError(t, err)
	assert.Equal(t, PackageRequest, pkgres.Header.Type)
	assert.Equal(t, constant.SHessian2, pkgres.Header.SerialID)
	assert.Equal(t, int64(10086), pkgres.Header.ID)
	assert.Equal(t, "2.0.2", reassembleBody["dubboVersion"].(string))
	assert.Equal(t, "path", pkgres.Service.Path)
	assert.Equal(t, "2.6", pkgres.Service.Version)
	assert.Equal(t, "Method", pkgres.Service.Method)
	assert.Equal(t, "Ljava/lang/String;", reassembleBody["argsTypes"].(string))
	assert.Equal(t, []any{"a"}, reassembleBody["args"])
	tmpData := map[string]any{
		"dubbo":     "2.0.2",
		"interface": "Service",
		"path":      "path",
		"timeout":   "1000",
		"version":   "2.6",
	}
	assert.Equal(t, tmpData, reassembleBody["attachments"])
}

// TestEncodeHeaderHeartbeat tests EncodeHeader for heartbeat packages
func TestEncodeHeaderHeartbeat(t *testing.T) {
	codec := NewDubboCodec(nil)
	pkg := NewDubboPackage(nil)
	pkg.Header.Type = PackageHeartbeat
	pkg.Header.SerialID = constant.SHessian2
	pkg.Header.ID = 12345

	header := codec.EncodeHeader(*pkg)
	assert.NotNil(t, header)
	assert.NotEmpty(t, header)
	assert.Equal(t, MAGIC_HIGH, header[0])
	assert.Equal(t, MAGIC_LOW, header[1])
}

// TestEncodeHeaderResponse tests EncodeHeader for response packages
func TestEncodeHeaderResponse(t *testing.T) {
	codec := NewDubboCodec(nil)
	pkg := NewDubboPackage(nil)
	pkg.Header.Type = PackageResponse
	pkg.Header.ResponseStatus = Response_OK
	pkg.Header.SerialID = constant.SHessian2
	pkg.Header.ID = 99999

	header := codec.EncodeHeader(*pkg)
	assert.NotNil(t, header)
	assert.Equal(t, MAGIC_HIGH, header[0])
	assert.Equal(t, MAGIC_LOW, header[1])
	assert.Equal(t, Response_OK, header[3])
}

// TestEncodeHeaderResponseWithError tests EncodeHeader for error response packages
func TestEncodeHeaderResponseWithError(t *testing.T) {
	codec := NewDubboCodec(nil)
	pkg := NewDubboPackage(nil)
	pkg.Header.Type = PackageResponse
	pkg.Header.ResponseStatus = Response_SERVICE_ERROR
	pkg.Header.SerialID = constant.SHessian2
	pkg.Header.ID = 11111

	header := codec.EncodeHeader(*pkg)
	assert.NotNil(t, header)
	assert.Equal(t, Response_SERVICE_ERROR, header[3])
}

// TestEncodeHeaderRequestTwoWay tests EncodeHeader for two-way request packages
func TestEncodeHeaderRequestTwoWay(t *testing.T) {
	codec := NewDubboCodec(nil)
	pkg := NewDubboPackage(nil)
	pkg.Header.Type = PackageRequest_TwoWay
	pkg.Header.SerialID = constant.SHessian2
	pkg.Header.ID = 22222

	header := codec.EncodeHeader(*pkg)
	assert.NotNil(t, header)
	assert.NotEmpty(t, header)
}

// TestProtocolCodecSetSerializer tests SetSerializer method of ProtocolCodec
func TestProtocolCodecSetSerializer(t *testing.T) {
	codec := NewDubboCodec(nil)
	serializer := &HessianSerializer{}

	codec.SetSerializer(serializer)
	assert.NotNil(t, codec.serializer)
}

// TestPackageResponseWorkflow tests response package workflow
func TestPackageResponseWorkflow(t *testing.T) {
	mockSerializer := &HessianSerializer{}
	SetSerializer(constant.Hessian2Serialization, mockSerializer)

	pkg := NewDubboPackage(nil)
	pkg.Header.Type = PackageResponse
	pkg.Header.SerialID = constant.SHessian2
	pkg.Header.ID = 99999
	pkg.Header.ResponseStatus = Response_OK
	pkg.Body = "response"
	pkg.SetSerializer(mockSerializer)

	assert.False(t, pkg.IsHeartBeat())
	assert.False(t, pkg.IsRequest())
	assert.True(t, pkg.IsResponse())
	assert.False(t, pkg.IsResponseWithException())

	data, err := pkg.Marshal()
	require.NoError(t, err)
	assert.NotNil(t, data)
}

// TestPackageRequestWorkflow tests request package workflow
func TestPackageRequestWorkflow(t *testing.T) {
	mockSerializer := &HessianSerializer{}
	SetSerializer(constant.Hessian2Serialization, mockSerializer)

	pkg := NewDubboPackage(nil)
	pkg.Header.Type = PackageRequest
	pkg.Header.SerialID = constant.SHessian2
	pkg.Header.ID = 66666
	pkg.Service.Interface = "TestService"
	pkg.Service.Path = "/dubbo"
	pkg.Service.Method = "test"
	pkg.Body = []any{"arg1"}
	pkg.SetSerializer(mockSerializer)

	assert.False(t, pkg.IsHeartBeat())
	assert.True(t, pkg.IsRequest())
	assert.False(t, pkg.IsResponse())

	data, err := pkg.Marshal()
	require.NoError(t, err)
	assert.NotNil(t, data)
}

// TestPackageBodyManipulation tests body manipulation methods
func TestPackageBodyManipulation(t *testing.T) {
	pkg := NewDubboPackage(nil)

	testBodies := []any{
		"string",
		123,
		[]string{"a", "b"},
		map[string]any{"key": "value"},
	}

	for _, body := range testBodies {
		pkg.SetBody(body)
		assert.Equal(t, body, pkg.GetBody())
	}
}

// TestPackageIDManipulation tests ID manipulation
func TestPackageIDManipulation(t *testing.T) {
	pkg := NewDubboPackage(nil)

	ids := []int64{1, 100, 1000, 999999, 9223372036854775807}

	for _, id := range ids {
		pkg.SetID(id)
		assert.Equal(t, id, pkg.Header.ID)
	}
}

// TestPackageServiceManipulation tests service manipulation
func TestPackageServiceManipulation(t *testing.T) {
	pkg := NewDubboPackage(nil)

	services := []Service{
		{Path: "/a", Interface: "InterfaceA"},
		{Path: "/b", Interface: "InterfaceB", Version: "1.0"},
		{Path: "/c", Interface: "InterfaceC", Group: "g1", Version: "2.0"},
	}

	for _, svc := range services {
		pkg.SetService(svc)
		retrieved := pkg.GetService()
		assert.Equal(t, svc.Path, retrieved.Path)
		assert.Equal(t, svc.Interface, retrieved.Interface)
		assert.Equal(t, svc.Version, retrieved.Version)
		assert.Equal(t, svc.Group, retrieved.Group)
	}
}

// TestPackageHeaderManipulation tests header manipulation
func TestPackageHeaderManipulation(t *testing.T) {
	pkg := NewDubboPackage(nil)

	headers := []DubboHeader{
		{ID: 1, BodyLen: 100, Type: PackageRequest},
		{ID: 2, BodyLen: 200, Type: PackageResponse, ResponseStatus: Response_OK},
		{ID: 3, BodyLen: 300, Type: PackageHeartbeat, SerialID: constant.SHessian2},
	}

	for _, header := range headers {
		pkg.SetHeader(header)
		retrieved := pkg.GetHeader()
		assert.Equal(t, header.ID, retrieved.ID)
		assert.Equal(t, header.BodyLen, retrieved.BodyLen)
		assert.Equal(t, header.Type, retrieved.Type)
	}
}

// TestPackageResponseStatusWorkflow tests response status workflow
func TestPackageResponseStatusWorkflow(t *testing.T) {
	pkg := NewDubboPackage(nil)
	pkg.Header.Type = PackageResponse

	statuses := []struct {
		status byte
		name   string
	}{
		{Response_OK, "OK"},
		{Response_CLIENT_TIMEOUT, "CLIENT_TIMEOUT"},
		{Response_SERVER_TIMEOUT, "SERVER_TIMEOUT"},
		{Response_BAD_REQUEST, "BAD_REQUEST"},
	}

	for _, s := range statuses {
		pkg.SetResponseStatus(s.status)
		assert.Equal(t, s.status, pkg.Header.ResponseStatus)
	}
}

// TestDubboPackageIntegrationWithLoadSerializer tests integration with LoadSerializer
func TestDubboPackageIntegrationWithLoadSerializer(t *testing.T) {
	mockSerializer := &HessianSerializer{}
	SetSerializer(constant.Hessian2Serialization, mockSerializer)

	pkg := NewDubboPackage(nil)
	pkg.Header.SerialID = constant.SHessian2
	pkg.Header.Type = PackageRequest
	pkg.Header.ID = 55555
	pkg.Service.Interface = "TestService"
	pkg.Body = []any{"test"}

	err := LoadSerializer(pkg)
	require.NoError(t, err)

	data, err := pkg.Marshal()
	require.NoError(t, err)
	assert.NotNil(t, data)
	assert.Positive(t, data.Len())
}
