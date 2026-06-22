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
	"sync"
	"testing"
)

import (
	hessian "github.com/apache/dubbo-go-hessian2"
	"github.com/apache/dubbo-go-hessian2/java_exception"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func doTestReflectResponse(t *testing.T, in any, out any) {
	err := ReflectResponse(in, out)
	require.NoError(t, err)

	result := hessian.UnpackPtrValue(reflect.ValueOf(out)).Interface()

	equal := reflect.DeepEqual(in, result)
	if !equal {
		t.Errorf("expect [%v]: %v, but got [%v]: %v", reflect.TypeOf(in), in, reflect.TypeOf(result), result)
	}
}

func TestReflectResponse(t *testing.T) {
	var b bool
	doTestReflectResponse(t, true, &b)
	doTestReflectResponse(t, false, &b)

	var i int
	doTestReflectResponse(t, 123, &i)
	doTestReflectResponse(t, 234, &i)

	var i16 int16
	doTestReflectResponse(t, int16(456), &i16)

	var i64 int64
	doTestReflectResponse(t, int64(789), &i64)

	var s string
	doTestReflectResponse(t, "hello world", &s)

	type rr struct {
		Name string
		Num  int
	}

	var r1 rr
	doTestReflectResponse(t, rr{"dubbogo", 32}, &r1)

	// ------ map test -------
	m1 := make(map[any]any)
	var m1r map[any]any
	m1["hello"] = "world"
	m1[1] = "go"
	m1["dubbo"] = 666
	doTestReflectResponse(t, m1, &m1r)

	m2 := make(map[string]string)
	var m2r map[string]string
	m2["hello"] = "world"
	m2["dubbo"] = "666"
	doTestReflectResponse(t, m2, &m2r)

	m3 := make(map[string]rr)
	var m3r map[string]rr
	m3["dubbo"] = rr{"hello", 123}
	m3["go"] = rr{"world", 456}
	doTestReflectResponse(t, m3, &m3r)

	// ------ slice test -------
	s1 := []string{"abc", "def", "hello", "world"}
	var s1r []string
	doTestReflectResponse(t, s1, &s1r)

	s2 := []rr{{"dubbo", 666}, {"go", 999}}
	var s2r []rr
	doTestReflectResponse(t, s2, &s2r)

	s3 := []any{rr{"dubbo", 666}, 123, "hello"}
	var s3r []any
	doTestReflectResponse(t, s3, &s3r)

	// ------ interface test -------
	in1 := []any{rr{"dubbo", 666}, 123, "hello"}
	var inr1 *any
	doTestReflectResponse(t, in1, reflect.New(reflect.TypeOf(inr1).Elem()).Interface())

	in2 := make(map[string]rr)
	var inr2 map[string]rr
	m3["dubbo"] = rr{"hello", 123}
	m3["go"] = rr{"world", 456}
	doTestReflectResponse(t, in2, &inr2)
}

// separately test copy normal map to map[any]any
func TestCopyMap(t *testing.T) {
	type rr struct {
		Name string
		Num  int
	}

	m3 := make(map[string]rr)
	var m3r map[any]any
	r1 := rr{"hello", 123}
	r2 := rr{"world", 456}
	m3["dubbo"] = r1
	m3["go"] = r2

	err := ReflectResponse(m3, &m3r)
	require.NoError(t, err)

	assert.Len(t, m3r, 2)

	rr1, ok := m3r["dubbo"]
	assert.True(t, ok)
	assert.True(t, reflect.DeepEqual(r1, rr1))

	rr2, ok := m3r["go"]
	assert.True(t, ok)
	assert.True(t, reflect.DeepEqual(r2, rr2))
}

// separately test copy normal slice to []any
func TestCopySlice(t *testing.T) {
	type rr struct {
		Name string
		Num  int
	}

	r1 := rr{"hello", 123}
	r2 := rr{"world", 456}

	s1 := []rr{r1, r2}
	var s1r []any

	err := ReflectResponse(s1, &s1r)
	require.NoError(t, err)

	assert.Len(t, s1r, 2)
	assert.True(t, reflect.DeepEqual(r1, s1r[0]))
	assert.True(t, reflect.DeepEqual(r2, s1r[1]))
}

func TestIsSupportResponseAttachment(t *testing.T) {
	is := isSupportResponseAttachment("2.X")
	assert.False(t, is)

	is = isSupportResponseAttachment("2.0.10")
	assert.False(t, is)

	is = isSupportResponseAttachment("2.5.3")
	assert.False(t, is)

	is = isSupportResponseAttachment("2.6.2")
	assert.False(t, is)

	is = isSupportResponseAttachment("1.5.5")
	assert.False(t, is)

	is = isSupportResponseAttachment("0.0.0")
	assert.False(t, is)

	is = isSupportResponseAttachment("2.0.2")
	assert.True(t, is)

	is = isSupportResponseAttachment("2.7.2")
	assert.True(t, is)
}

func TestIsSupportResponseAttachmentConcurrent(t *testing.T) {
	versions := []any{"2.X", "2.0.10", "2.5.3", "2.6.2", "2.0.2", "2.7.2", ""}
	var wg sync.WaitGroup

	for i := 0; i < 200; i++ {
		for _, version := range versions {
			wg.Add(1)
			go func(v any) {
				defer wg.Done()
				_ = isSupportResponseAttachment(v)
			}(version)
		}
	}

	wg.Wait()
}

func TestToGenericExceptionUsesHessianExceptionType(t *testing.T) {
	exception, ok := ToGenericException(java_exception.DubboGenericException{
		ExceptionClass:   "com.example.UserNotFoundException",
		ExceptionMessage: "user not found",
	})

	require.True(t, ok)
	require.IsType(t, &java_exception.DubboGenericException{}, exception)
	assert.Equal(t, "com.example.UserNotFoundException", exception.ExceptionClass)
	assert.Equal(t, "user not found", exception.ExceptionMessage)
}

func TestToGenericExceptionConversions(t *testing.T) {
	pointerException := java_exception.NewDubboGenericException("com.example.PointerException", "pointer message")

	tests := []struct {
		name             string
		input            any
		wantOK           bool
		wantClass        string
		wantMessage      string
		wantDetailString string
	}{
		{
			name:        "generic exception pointer",
			input:       pointerException,
			wantOK:      true,
			wantClass:   "com.example.PointerException",
			wantMessage: "pointer message",
		},
		{
			name:             "throwable",
			input:            java_exception.NewThrowable("throwable message"),
			wantOK:           true,
			wantClass:        "java.lang.Throwable",
			wantMessage:      "throwable message",
			wantDetailString: "java exception: java.lang.Throwable - throwable message",
		},
		{
			name:             "legacy exception string",
			input:            "java exception: user not found",
			wantOK:           true,
			wantClass:        "java.lang.Exception",
			wantMessage:      "user not found",
			wantDetailString: "java exception: java.lang.Exception - user not found",
		},
		{
			name:             "plain string",
			input:            "plain failure",
			wantOK:           true,
			wantClass:        "java.lang.Exception",
			wantMessage:      "plain failure",
			wantDetailString: "java exception: java.lang.Exception - plain failure",
		},
		{
			name:   "unsupported type",
			input:  42,
			wantOK: false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			exception, ok := ToGenericException(test.input)
			assert.Equal(t, test.wantOK, ok)
			if !test.wantOK {
				assert.Nil(t, exception)
				return
			}

			require.NotNil(t, exception)
			assert.Equal(t, test.wantClass, exception.ExceptionClass)
			assert.Equal(t, test.wantMessage, exception.ExceptionMessage)
			if test.wantDetailString != "" {
				assert.Equal(t, test.wantDetailString, exception.Error())
			}
		})
	}
}

func TestNewGenericExceptionDetailMessage(t *testing.T) {
	tests := []struct {
		name        string
		class       string
		message     string
		wantDetails string
	}{
		{
			name:        "message only",
			message:     "message only",
			wantDetails: "message only",
		},
		{
			name:        "class only",
			class:       "com.example.EmptyMessage",
			wantDetails: "com.example.EmptyMessage",
		},
		{
			name:        "class and message",
			class:       "com.example.UserNotFoundException",
			message:     "user not found",
			wantDetails: "java exception: com.example.UserNotFoundException - user not found",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			exception := newGenericException(test.class, test.message)
			assert.Equal(t, test.class, exception.ExceptionClass)
			assert.Equal(t, test.message, exception.ExceptionMessage)
			assert.Equal(t, test.wantDetails, exception.Error())
		})
	}
}

func TestPackResponseWithGenericExceptionPointer(t *testing.T) {
	tests := []struct {
		name        string
		exception   error
		wantClass   string
		wantMessage string
	}{
		{
			name: "generic exception pointer",
			exception: java_exception.NewDubboGenericException(
				"com.example.UserNotFoundException",
				"user not found",
			),
			wantClass:   "com.example.UserNotFoundException",
			wantMessage: "user not found",
		},
		{
			name: "generic exception value",
			exception: java_exception.DubboGenericException{
				ExceptionClass:   "com.example.IllegalStateException",
				ExceptionMessage: "illegal state",
			},
			wantClass:   "com.example.IllegalStateException",
			wantMessage: "illegal state",
		},
		{
			name:        "throwable",
			exception:   java_exception.NewThrowable("throwable message"),
			wantClass:   "java.lang.Throwable",
			wantMessage: "throwable message",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			body := NewResponse(nil, test.exception, map[string]any{})
			data, err := packResponse(DubboHeader{
				SerialID:       2,
				Type:           PackageResponse,
				ID:             1,
				ResponseStatus: Response_OK,
			}, body)
			require.NoError(t, err)

			codecR := NewHessianCodec(bufio.NewReader(bytes.NewReader(data)))
			header := &DubboHeader{}
			require.NoError(t, codecR.ReadHeader(header))
			assert.Equal(t, PackageResponse, header.Type&PackageResponse)
			assert.Equal(t, Response_OK, header.ResponseStatus)

			decodedResponse := &DubboResponse{}
			require.NoError(t, codecR.ReadBody(decodedResponse))

			ge, ok := decodedResponse.Exception.(*java_exception.DubboGenericException)
			require.True(t, ok)
			assert.Equal(t, test.wantClass, ge.ExceptionClass)
			assert.Equal(t, test.wantMessage, ge.ExceptionMessage)
		})
	}
}

func TestVersion2Int(t *testing.T) {
	v := version2Int("2.1.3")
	assert.Equal(t, 2010300, v)

	v = version2Int("22.11.33")
	assert.Equal(t, 22113300, v)

	v = version2Int("222.111.333")
	assert.Equal(t, 223143300, v)

	v = version2Int("220.110.333")
	assert.Equal(t, 221133300, v)

	v = version2Int("229.119.333")
	assert.Equal(t, 230223300, v)

	v = version2Int("2222.1111.3333")
	assert.Equal(t, 2233443300, v)

	v = version2Int("2.11")
	assert.Equal(t, 211, v)

	v = version2Int("2.1.3.4")
	assert.Equal(t, 2010304, v)

	v = version2Int("2.1.3.4.5")
	assert.Equal(t, 201030405, v)
}
