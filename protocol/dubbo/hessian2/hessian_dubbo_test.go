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

package hessian2

import (
	"bufio"
	"bytes"
	"reflect"
	"testing"
	"time"
)

import (
	hessian "github.com/apache/dubbo-go-hessian2"

	"github.com/stretchr/testify/assert"
)

type Case struct {
	A string
	B int
}

type CaseA struct {
	A string
	B int
	C Case
}

type CaseB struct {
	A string
	B CaseA
}

func (c *CaseB) JavaClassName() string {
	return "com.test.caseb"
}

func (c CaseA) JavaClassName() string {
	return "com.test.casea"
}

// JavaClassName  java fully qualified path
func (c Case) JavaClassName() string {
	return "com.test.case"
}

func doTestHessianEncodeHeader(t *testing.T, packageType PackageType, responseStatus byte, body any) ([]byte, error) {
	hessian.RegisterPOJO(&Case{})
	codecW := NewHessianCodec(nil)
	resp, err := codecW.Write(Service{
		Path:      "test",
		Interface: "ITest",
		Version:   "v1.0",
		Method:    "test",
		Timeout:   time.Second * 10,
	}, DubboHeader{
		SerialID:       2,
		Type:           packageType,
		ID:             1,
		ResponseStatus: responseStatus,
	}, body)
	assert.Nil(t, err)
	return resp, err
}

func doTestResponse(t *testing.T, packageType PackageType, responseStatus byte, body any, decodedResponse *DubboResponse, assertFunc func()) {
	resp, err := doTestHessianEncodeHeader(t, packageType, responseStatus, body)
	assert.NoError(t, err)

	codecR := NewHessianCodec(bufio.NewReader(bytes.NewReader(resp)))

	h := &DubboHeader{}
	err = codecR.ReadHeader(h)
	assert.Nil(t, err)

	assert.Equal(t, byte(2), h.SerialID)
	assert.Equal(t, packageType, h.Type&(PackageRequest|PackageResponse|PackageHeartbeat))
	assert.Equal(t, int64(1), h.ID)
	assert.Equal(t, responseStatus, h.ResponseStatus)

	err = codecR.ReadBody(decodedResponse)
	assert.Nil(t, err)
	t.Log(decodedResponse)

	if assertFunc != nil {
		assertFunc()
		return
	}

	if h.ResponseStatus != Zero && h.ResponseStatus != Response_OK {
		assert.Equal(t, "java exception:"+body.(string), decodedResponse.Exception.Error())
		return
	}

	in, _ := hessian.EnsureInterface(hessian.UnpackPtrValue(hessian.EnsurePackValue(body)), nil)
	out, _ := hessian.EnsureInterface(hessian.UnpackPtrValue(hessian.EnsurePackValue(decodedResponse.RspObj)), nil)
	assert.Equal(t, in, out)
}

func TestResponse(t *testing.T) {
	caseObj := Case{A: "a", B: 1}
	decodedResponse := &DubboResponse{}
	hessian.RegisterPOJO(&caseObj)

	arr := []*Case{&caseObj}
	decodedResponse.RspObj = nil
	doTestResponse(t, PackageResponse, Response_OK, arr, decodedResponse, func() {
		arrRes, ok := decodedResponse.RspObj.([]*Case)
		if !ok {
			t.Errorf("expect []*Case, but get %s", reflect.TypeOf(decodedResponse.RspObj).String())
			return
		}
		assert.Equal(t, 1, len(arrRes))
		assert.Equal(t, &caseObj, arrRes[0])
	})

	doTestResponse(t, PackageResponse, Response_OK, &caseObj, decodedResponse, func() {
		assert.Equal(t, &caseObj, decodedResponse.RspObj)
	})

	s := "ok!!!!!"
	doTestResponse(t, PackageResponse, Response_OK, s, decodedResponse, func() {
		assert.Equal(t, s, decodedResponse.RspObj)
	})

	doTestResponse(t, PackageResponse, Response_OK, int64(3), decodedResponse, func() {
		assert.Equal(t, int64(3), decodedResponse.RspObj)
	})

	doTestResponse(t, PackageResponse, Response_OK, true, decodedResponse, func() {
		assert.Equal(t, true, decodedResponse.RspObj)
	})

	errorMsg := "error!!!!!"
	decodedResponse.RspObj = nil
	doTestResponse(t, PackageResponse, Response_SERVER_ERROR, errorMsg, decodedResponse, func() {
		assert.Equal(t, "java exception:error!!!!!", decodedResponse.Exception.Error())
	})

	decodedResponse.RspObj = nil
	decodedResponse.Exception = nil
	mapObj := map[string][]*Case{"key": {&caseObj}}
	doTestResponse(t, PackageResponse, Response_OK, mapObj, decodedResponse, func() {
		mapRes, ok := decodedResponse.RspObj.(map[any]any)
		if !ok {
			t.Errorf("expect map[string][]*Case, but get %s", reflect.TypeOf(decodedResponse.RspObj).String())
			return
		}
		c, ok := mapRes["key"]
		if !ok {
			assert.FailNow(t, "no key in decoded response map")
		}

		mapValueArr, ok := c.([]*Case)
		if !ok {
			assert.FailNow(t, "invalid decoded response map value", "expect []*Case, but get %v", reflect.TypeOf(c))
		}
		assert.Equal(t, 1, len(mapValueArr))
		assert.Equal(t, &caseObj, mapValueArr[0])
	})
}

func doTestRequest(t *testing.T, packageType PackageType, responseStatus byte, body any) {
	resp, err := doTestHessianEncodeHeader(t, packageType, responseStatus, body)
	assert.NoError(t, err)

	codecR := NewHessianCodec(bufio.NewReader(bytes.NewReader(resp)))

	h := &DubboHeader{}
	err = codecR.ReadHeader(h)
	assert.Nil(t, err)
	assert.Equal(t, byte(2), h.SerialID)
	assert.Equal(t, packageType, h.Type&(PackageRequest|PackageResponse|PackageHeartbeat))
	assert.Equal(t, int64(1), h.ID)
	assert.Equal(t, responseStatus, h.ResponseStatus)

	c := make([]any, 7)
	err = codecR.ReadBody(c)
	assert.Nil(t, err)
	t.Log(c)
	assert.True(t, len(body.([]any)) == len(c[5].([]any)))
}

func TestRequest(t *testing.T) {
	doTestRequest(t, PackageRequest, Zero, []any{"a"})
	doTestRequest(t, PackageRequest, Zero, []any{"a", 3})
	doTestRequest(t, PackageRequest, Zero, []any{"a", true})
	doTestRequest(t, PackageRequest, Zero, []any{"a", 3, true})
	doTestRequest(t, PackageRequest, Zero, []any{3.2, true})
	doTestRequest(t, PackageRequest, Zero, []any{"a", 3, true, &Case{A: "a", B: 3}})
	doTestRequest(t, PackageRequest, Zero, []any{"a", 3, true, []*Case{{A: "a", B: 3}}})
	doTestRequest(t, PackageRequest, Zero, []any{map[string][]*Case{"key": {{A: "a", B: 3}}}})
}

func TestHessianCodec_ReadAttachments(t *testing.T) {
	hessian.RegisterPOJO(&AttachTestObject{})
	body := &DubboResponse{
		RspObj:      &CaseB{A: "A", B: CaseA{A: "a", B: 1, C: Case{A: "c", B: 2}}},
		Exception:   nil,
		Attachments: map[string]any{DUBBO_VERSION_KEY: "2.6.4", "att": AttachTestObject{ID: 23, Name: "haha"}},
	}
	resp, err := doTestHessianEncodeHeader(t, PackageResponse, Response_OK, body)
	assert.NoError(t, err)
	hessian.UnRegisterPOJOs(&CaseB{}, &CaseA{})
	codecR1 := NewHessianCodec(bufio.NewReader(bytes.NewReader(resp)))
	codecR2 := NewHessianCodec(bufio.NewReader(bytes.NewReader(resp)))
	h := &DubboHeader{}
	assert.NoError(t, codecR1.ReadHeader(h))
	t.Log(h)
	assert.NoError(t, codecR2.ReadHeader(h))
	t.Log(h)

	err = codecR1.ReadBody(body)
	assert.NoError(t, err)
	attrs, err := codecR2.ReadAttachments()
	assert.NoError(t, err)
	assert.Equal(t, "2.6.4", attrs[DUBBO_VERSION_KEY])
	assert.Equal(t, AttachTestObject{ID: 23, Name: "haha"}, *(attrs["att"].(*AttachTestObject)))
	assert.NotEqual(t, AttachTestObject{ID: 24, Name: "nohaha"}, *(attrs["att"].(*AttachTestObject)))

	t.Log(attrs)
}

type AttachTestObject struct {
	ID   int32
	Name string `dubbo:"name"`
}

func (AttachTestObject) JavaClassName() string {
	return "com.test.Test"
}

type CaseStream struct {
	Payload string
}

func (CaseStream) JavaClassName() string {
	return "com.test.CaseStream"
}

func TestDecodeFromTcpStream(t *testing.T) {
	payload := make([]byte, 1024)
	alphabet := "abcdefghijklmnopqrstuvwxyz"
	for i, _ := range payload {
		payload[i] = alphabet[i%26]
	}
	cs := &CaseStream{
		Payload: string(payload),
	}

	hessian.RegisterPOJO(cs)
	codecW := NewHessianCodec(nil)
	service := Service{
		Path:      "test",
		Interface: "ITest",
		Version:   "v1.0",
		Method:    "test",
		Timeout:   time.Second * 10,
	}
	header := DubboHeader{
		SerialID:       2,
		Type:           PackageRequest,
		ID:             1,
		ResponseStatus: Zero,
	}
	resp, err := codecW.Write(service, header, []interface{}{cs})

	// set reader buffer = 1024 to split resp into two parts
	codec := NewStreamingHessianCodecCustom(0, bufio.NewReaderSize(bytes.NewReader(resp), 1024), 0)
	h := &DubboHeader{}
	assert.NoError(t, codec.ReadHeader(h))
	assert.Equal(t, h.SerialID, header.SerialID)
	assert.Equal(t, h.Type, header.Type)
	assert.Equal(t, h.ID, header.ID)
	assert.Equal(t, h.ResponseStatus, header.ResponseStatus)

	reqBody := make([]interface{}, 7)

	err = codec.ReadBody(reqBody)
	assert.NoError(t, err)
	assert.Equal(t, reqBody[1], service.Path)
	assert.Equal(t, reqBody[2], service.Version)
	assert.Equal(t, reqBody[3], service.Method)

	if list, ok := reqBody[5].([]interface{}); ok {
		assert.Len(t, list, 1)
		if infoPtr, ok2 := list[0].(*CaseStream); ok2 {
			assert.Equal(t, len(infoPtr.Payload), 1024)
		}
	}

	codec = NewHessianCodecCustom(0, bufio.NewReaderSize(bytes.NewReader(resp), 1024), 0)
	err = codec.ReadHeader(h)
	assert.ErrorIs(t, err, ErrBodyNotEnough)
}
