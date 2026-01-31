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
	"errors"
	"testing"
	"time"
)

import (
	hessian "github.com/apache/dubbo-go-hessian2"
	"github.com/apache/dubbo-go-hessian2/java_exception"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/protocol/dubbo/hessian2"
)

const (
	dubboParam = "org.apache.dubbo.param"
	dubboPojo  = "org.apache.dubbo.pojo"
)

type Param struct {
}

func (p *Param) JavaClassName() string {
	return dubboPojo
}

func (p *Param) JavaParamName() string {
	return dubboParam
}

type Pojo struct {
}

func (p *Pojo) JavaClassName() string {
	return dubboPojo
}

func TestGetArgType(t *testing.T) {

	t.Run("pojo", func(t *testing.T) {
		assert.Equal(t, dubboPojo, getArgType(&Pojo{}))
	})

	t.Run("param", func(t *testing.T) {
		assert.Equal(t, dubboParam, getArgType(&Param{}))
	})

	t.Run("nil", func(t *testing.T) {
		assert.Equal(t, "V", getArgType(nil))
	})

	t.Run("bool", func(t *testing.T) {
		assert.Equal(t, "Z", getArgType(true))
	})

	t.Run("bool array", func(t *testing.T) {
		assert.Equal(t, "[Z", getArgType([]bool{true, false}))
	})

	t.Run("byte", func(t *testing.T) {
		assert.Equal(t, "B", getArgType(byte(1)))
	})

	t.Run("byte array", func(t *testing.T) {
		assert.Equal(t, "[B", getArgType([]byte{1, 2, 3}))
	})

	t.Run("int8", func(t *testing.T) {
		assert.Equal(t, "B", getArgType(int8(1)))
	})

	t.Run("int8 array", func(t *testing.T) {
		assert.Equal(t, "[B", getArgType([]int8{1, 2, 3}))
	})

	t.Run("int16", func(t *testing.T) {
		assert.Equal(t, "S", getArgType(int16(1)))
	})

	t.Run("int16 array", func(t *testing.T) {
		assert.Equal(t, "[S", getArgType([]int16{1, 2, 3}))
	})

	t.Run("uint16", func(t *testing.T) {
		assert.Equal(t, "C", getArgType(uint16(1)))
	})

	t.Run("uint16 array", func(t *testing.T) {
		assert.Equal(t, "[C", getArgType([]uint16{1, 2, 3}))
	})

	t.Run("int", func(t *testing.T) {
		assert.Equal(t, "J", getArgType(int(1)))
	})

	t.Run("int array", func(t *testing.T) {
		assert.Equal(t, "[J", getArgType([]int{1, 2, 3}))
	})

	t.Run("int32", func(t *testing.T) {
		assert.Equal(t, "I", getArgType(int32(1)))
	})

	t.Run("int32 array", func(t *testing.T) {
		assert.Equal(t, "[I", getArgType([]int32{1, 2, 3}))
	})

	t.Run("int64", func(t *testing.T) {
		assert.Equal(t, "J", getArgType(int64(1)))
	})

	t.Run("int64 array", func(t *testing.T) {
		assert.Equal(t, "[J", getArgType([]int64{1, 2, 3}))
	})

	t.Run("time.Time", func(t *testing.T) {
		assert.Equal(t, "java.util.Date", getArgType(time.Now()))
	})

	t.Run("time.Time array", func(t *testing.T) {
		assert.Equal(t, "[Ljava.util.Date", getArgType([]time.Time{time.Now()}))
	})

	t.Run("float32", func(t *testing.T) {
		assert.Equal(t, "F", getArgType(float32(1.0)))
	})

	t.Run("float32 array", func(t *testing.T) {
		assert.Equal(t, "[F", getArgType([]float32{1.0, 2.0}))
	})

	t.Run("float64", func(t *testing.T) {
		assert.Equal(t, "D", getArgType(float64(1.0)))
	})

	t.Run("float64 array", func(t *testing.T) {
		assert.Equal(t, "[D", getArgType([]float64{1.0, 2.0}))
	})

	t.Run("string", func(t *testing.T) {
		assert.Equal(t, "java.lang.String", getArgType("test"))
	})

	t.Run("string array", func(t *testing.T) {
		assert.Equal(t, "[Ljava.lang.String;", getArgType([]string{"test1", "test2"}))
	})

	t.Run("map[any]any", func(t *testing.T) {
		assert.Equal(t, "java.util.Map", getArgType(map[any]any{"key": "value"}))
	})

	t.Run("map[string]int", func(t *testing.T) {
		assert.Equal(t, "java.util.Map", getArgType(map[string]int{"key": 1}))
	})

	t.Run("pointer types", func(t *testing.T) {
		v := int8(1)
		assert.Equal(t, "java.lang.Byte", getArgType(&v))

		v2 := int16(1)
		assert.Equal(t, "java.lang.Short", getArgType(&v2))

		v3 := uint16(1)
		assert.Equal(t, "java.lang.Character", getArgType(&v3))

		v4 := int(1)
		assert.Equal(t, "java.lang.Long", getArgType(&v4))

		v5 := int32(1)
		assert.Equal(t, "java.lang.Integer", getArgType(&v5))

		v6 := int64(1)
		assert.Equal(t, "java.lang.Long", getArgType(&v6))

		v7 := float32(1.0)
		assert.Equal(t, "java.lang.Float", getArgType(&v7))

		v8 := float64(1.0)
		assert.Equal(t, "java.lang.Double", getArgType(&v8))
	})

	t.Run("slice of structs", func(t *testing.T) {
		assert.Equal(t, "[Ljava.lang.Object;", getArgType([]Pojo{{}, {}}))
	})

	t.Run("slice of string", func(t *testing.T) {
		assert.Equal(t, "[Ljava.lang.String;", getArgType([]string{"a", "b"}))
	})

	t.Run("generic object", func(t *testing.T) {
		type GenericStruct struct {
			Field string
		}
		assert.Equal(t, "java.lang.Object", getArgType(&GenericStruct{Field: "test"}))
	})
}

func TestGetArgsTypeList(t *testing.T) {
	t.Run("multiple types", func(t *testing.T) {
		args := []any{int32(1), "test", true, float64(1.0)}
		types, err := GetArgsTypeList(args)
		require.NoError(t, err)
		assert.Equal(t, "ILjava/lang/String;ZD", types)
	})

	t.Run("with pojo", func(t *testing.T) {
		args := []any{&Pojo{}, "test"}
		types, err := GetArgsTypeList(args)
		require.NoError(t, err)
		assert.Equal(t, "Lorg/apache/dubbo/pojo;Ljava/lang/String;", types)
	})

	t.Run("with array", func(t *testing.T) {
		args := []any{[]string{"a", "b"}}
		types, err := GetArgsTypeList(args)
		require.NoError(t, err)
		assert.Equal(t, "[Ljava/lang/String;", types)
	})

	t.Run("empty args", func(t *testing.T) {
		args := []any{}
		types, err := GetArgsTypeList(args)
		require.NoError(t, err)
		assert.Empty(t, types)
	})
}

func TestToMapStringInterface(t *testing.T) {
	t.Run("normal map", func(t *testing.T) {
		origin := map[any]any{
			"key1": "value1",
			"key2": 123,
		}
		result := ToMapStringInterface(origin)
		assert.Equal(t, "value1", result["key1"])
		assert.Equal(t, 123, result["key2"])
	})

	t.Run("with nil value", func(t *testing.T) {
		origin := map[any]any{
			"key1": nil,
			"key2": "value2",
		}
		result := ToMapStringInterface(origin)
		assert.Empty(t, result["key1"])
		assert.Equal(t, "value2", result["key2"])
	})

	t.Run("with non-string key", func(t *testing.T) {
		origin := map[any]any{
			123:    "value1",
			"key2": "value2",
		}
		result := ToMapStringInterface(origin)
		assert.Len(t, result, 1)
		assert.Equal(t, "value2", result["key2"])
	})
}

func TestVersion2Int(t *testing.T) {
	tests := []struct {
		name     string
		version  string
		expected int
	}{
		{"version 2.0.10", "2.0.10", 2001000},
		{"version 2.6.2", "2.6.2", 2060200},
		{"version 2.7.0", "2.7.0", 2070000},
		{"version 2.0.1", "2.0.1", 2000100},
		{"version 3.0.0", "3.0.0", 3000000},
		{"invalid version", "invalid", -1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := version2Int(tt.version)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestIsSupportResponseAttachment(t *testing.T) {
	t.Run("empty version", func(t *testing.T) {
		assert.False(t, isSupportResponseAttachment(""))
	})

	t.Run("version 2.0.10", func(t *testing.T) {
		assert.False(t, isSupportResponseAttachment("2.0.10"))
	})

	t.Run("version 2.6.2", func(t *testing.T) {
		assert.False(t, isSupportResponseAttachment("2.6.2"))
	})

	t.Run("version 2.7.0", func(t *testing.T) {
		assert.True(t, isSupportResponseAttachment("2.7.0"))
	})

	t.Run("version 3.0.0", func(t *testing.T) {
		assert.True(t, isSupportResponseAttachment("3.0.0"))
	})

	t.Run("invalid version", func(t *testing.T) {
		assert.False(t, isSupportResponseAttachment("invalid"))
	})
}

func TestMarshalRequestWithTypedNilPointer(t *testing.T) {
	encoder := hessian.NewEncoder()
	pkg := DubboPackage{
		Service: Service{
			Path:    "test.Path",
			Version: "1.0.0",
			Method:  "Echo",
		},
		Body: &RequestPayload{
			Params: []any{(*int32)(nil)},
			Attachments: map[string]any{
				"key": "value",
			},
		},
	}

	data, err := marshalRequest(encoder, pkg)
	require.NoError(t, err)
	assert.NotNil(t, data)
}

func TestMarshalRequestWithNonNilPointer(t *testing.T) {
	val := int32(42)
	encoder := hessian.NewEncoder()
	pkg := DubboPackage{
		Service: Service{
			Path:    "test.Path",
			Version: "1.0.0",
			Method:  "Echo",
		},
		Body: &RequestPayload{
			Params: []any{&val},
			Attachments: map[string]any{
				"key": "value",
			},
		},
	}

	data, err := marshalRequest(encoder, pkg)
	require.NoError(t, err)
	assert.NotNil(t, data)
}

func TestMarshalRequest(t *testing.T) {
	t.Run("basic request", func(t *testing.T) {
		encoder := hessian.NewEncoder()
		pkg := DubboPackage{
			Service: Service{
				Path:      "com.test.Service",
				Interface: "com.test.Service",
				Version:   "1.0.0",
				Method:    "test",
				Group:     "testGroup",
				Timeout:   time.Second * 5,
			},
			Body: &RequestPayload{
				Params:      []any{int32(123), "test"},
				Attachments: map[string]any{},
			},
		}

		data, err := marshalRequest(encoder, pkg)
		require.NoError(t, err)
		assert.NotNil(t, data)
	})

	t.Run("request with invalid params", func(t *testing.T) {
		encoder := hessian.NewEncoder()
		pkg := DubboPackage{
			Service: Service{
				Path:    "test.Path",
				Version: "1.0.0",
				Method:  "Echo",
			},
			Body: &RequestPayload{
				Params:      "not a slice",
				Attachments: map[string]any{},
			},
		}

		_, err := marshalRequest(encoder, pkg)
		assert.Error(t, err)
	})
}

func TestMarshalResponse(t *testing.T) {
	t.Run("heartbeat response", func(t *testing.T) {
		encoder := hessian.NewEncoder()
		pkg := DubboPackage{
			Header: DubboHeader{
				Type:           PackageHeartbeat | PackageResponse,
				ResponseStatus: Response_OK,
			},
			Body: &ResponsePayload{
				Attachments: map[string]any{},
			},
		}

		data, err := marshalResponse(encoder, pkg)
		require.NoError(t, err)
		assert.NotNil(t, data)
	})

	t.Run("response with value", func(t *testing.T) {
		encoder := hessian.NewEncoder()
		pkg := DubboPackage{
			Header: DubboHeader{
				Type:           PackageResponse,
				ResponseStatus: Response_OK,
			},
			Body: &ResponsePayload{
				RspObj:      "test response",
				Attachments: map[string]any{},
			},
		}

		data, err := marshalResponse(encoder, pkg)
		require.NoError(t, err)
		assert.NotNil(t, data)
	})

	t.Run("response with null value", func(t *testing.T) {
		encoder := hessian.NewEncoder()
		pkg := DubboPackage{
			Header: DubboHeader{
				Type:           PackageResponse,
				ResponseStatus: Response_OK,
			},
			Body: &ResponsePayload{
				RspObj:      nil,
				Attachments: map[string]any{},
			},
		}

		data, err := marshalResponse(encoder, pkg)
		require.NoError(t, err)
		assert.NotNil(t, data)
	})

	t.Run("response with exception", func(t *testing.T) {
		encoder := hessian.NewEncoder()
		pkg := DubboPackage{
			Header: DubboHeader{
				Type:           PackageResponse,
				ResponseStatus: Response_OK,
			},
			Body: &ResponsePayload{
				Exception:   errors.New("test error"),
				Attachments: map[string]any{},
			},
		}

		data, err := marshalResponse(encoder, pkg)
		require.NoError(t, err)
		assert.NotNil(t, data)
	})

	t.Run("response with generic exception", func(t *testing.T) {
		encoder := hessian.NewEncoder()
		pkg := DubboPackage{
			Header: DubboHeader{
				Type:           PackageResponse,
				ResponseStatus: Response_OK,
			},
			Body: &ResponsePayload{
				Exception: hessian2.GenericException{
					ExceptionClass:   "com.example.UserNotFoundException",
					ExceptionMessage: "user not found",
				},
				Attachments: map[string]any{},
			},
		}

		data, err := marshalResponse(encoder, pkg)
		require.NoError(t, err)
		assert.NotNil(t, data)

		decoder := hessian.NewDecoder(data)
		rspType, err := decoder.Decode()
		require.NoError(t, err)
		assert.EqualValues(t, RESPONSE_WITH_EXCEPTION, rspType)

		expt, err := decoder.Decode()
		require.NoError(t, err)
		switch ge := expt.(type) {
		case *java_exception.DubboGenericException:
			assert.Equal(t, "com.example.UserNotFoundException", ge.ExceptionClass)
			assert.Equal(t, "user not found", ge.ExceptionMessage)
		case java_exception.DubboGenericException:
			assert.Equal(t, "com.example.UserNotFoundException", ge.ExceptionClass)
			assert.Equal(t, "user not found", ge.ExceptionMessage)
		default:
			require.Failf(t, "unexpected exception type", "%T", expt)
		}
	})

	t.Run("response with throwable exception", func(t *testing.T) {
		encoder := hessian.NewEncoder()
		pkg := DubboPackage{
			Header: DubboHeader{
				Type:           PackageResponse,
				ResponseStatus: Response_OK,
			},
			Body: &ResponsePayload{
				Exception:   java_exception.NewThrowable("test throwable"),
				Attachments: map[string]any{},
			},
		}

		data, err := marshalResponse(encoder, pkg)
		require.NoError(t, err)
		assert.NotNil(t, data)
	})

	t.Run("response with attachments", func(t *testing.T) {
		encoder := hessian.NewEncoder()
		pkg := DubboPackage{
			Header: DubboHeader{
				Type:           PackageResponse,
				ResponseStatus: Response_OK,
			},
			Body: &ResponsePayload{
				RspObj: "test",
				Attachments: map[string]any{
					DUBBO_VERSION_KEY: "2.7.0",
					"custom":          "value",
				},
			},
		}

		data, err := marshalResponse(encoder, pkg)
		require.NoError(t, err)
		assert.NotNil(t, data)
	})

	t.Run("response with error status", func(t *testing.T) {
		encoder := hessian.NewEncoder()
		pkg := DubboPackage{
			Header: DubboHeader{
				Type:           PackageResponse,
				ResponseStatus: Response_SERVER_ERROR,
			},
			Body: &ResponsePayload{
				Exception: errors.New("server error"),
			},
		}

		data, err := marshalResponse(encoder, pkg)
		require.NoError(t, err)
		assert.NotNil(t, data)
	})
}

func TestUnmarshalRequestBody(t *testing.T) {
	t.Run("basic request unmarshal", func(t *testing.T) {
		// Prepare encoded request
		encoder := hessian.NewEncoder()
		_ = encoder.Encode("2.0.2")                     // dubbo version
		_ = encoder.Encode("com.test.Service")          // target
		_ = encoder.Encode("1.0.0")                     // service version
		_ = encoder.Encode("test")                      // method
		_ = encoder.Encode("Ljava/lang/String;")        // args types
		_ = encoder.Encode("hello")                     // args
		_ = encoder.Encode(map[any]any{"key": "value"}) // attachments

		pkg := &DubboPackage{}
		err := unmarshalRequestBody(encoder.Buffer(), pkg)
		require.NoError(t, err)
		assert.NotNil(t, pkg.Body)

		body, ok := pkg.Body.(map[string]any)
		assert.True(t, ok)
		assert.Equal(t, "2.0.2", body["dubboVersion"])
		assert.Equal(t, "Ljava/lang/String;", body["argsTypes"])
		assert.NotNil(t, body["attachments"])
	})

	t.Run("request with multiple args", func(t *testing.T) {
		encoder := hessian.NewEncoder()
		_ = encoder.Encode("2.0.2")
		_ = encoder.Encode("com.test.Service")
		_ = encoder.Encode("1.0.0")
		_ = encoder.Encode("test")
		_ = encoder.Encode("ILjava/lang/String;")
		_ = encoder.Encode(int32(123))
		_ = encoder.Encode("test")
		_ = encoder.Encode(map[any]any{"key": "value"})

		pkg := &DubboPackage{}
		err := unmarshalRequestBody(encoder.Buffer(), pkg)
		require.NoError(t, err)

		body, ok := pkg.GetBody().(map[string]any)
		assert.True(t, ok)
		args := body["args"].([]any)
		assert.Len(t, args, 2)
	})

	t.Run("request with nil attachments", func(t *testing.T) {
		encoder := hessian.NewEncoder()
		_ = encoder.Encode("2.0.2")
		_ = encoder.Encode("com.test.Service")
		_ = encoder.Encode("1.0.0")
		_ = encoder.Encode("test")
		_ = encoder.Encode("")
		_ = encoder.Encode(nil)

		pkg := &DubboPackage{}
		err := unmarshalRequestBody(encoder.Buffer(), pkg)
		assert.NoError(t, err)
	})
}

func TestUnmarshalResponseBody(t *testing.T) {
	t.Run("response with value", func(t *testing.T) {
		encoder := hessian.NewEncoder()
		_ = encoder.Encode(RESPONSE_VALUE)
		_ = encoder.Encode("test response")

		// Need to set RspObj for ReflectResponse to work
		rspObj := ""
		pkg := &DubboPackage{
			Body: &ResponsePayload{
				RspObj: &rspObj,
			},
		}
		err := unmarshalResponseBody(encoder.Buffer(), pkg)
		require.NoError(t, err)
		assert.Equal(t, "test response", rspObj)
	})

	t.Run("response with value and attachments", func(t *testing.T) {
		encoder := hessian.NewEncoder()
		_ = encoder.Encode(RESPONSE_VALUE_WITH_ATTACHMENTS)
		_ = encoder.Encode("test response")
		_ = encoder.Encode(map[any]any{"key": "value"})

		rspObj := ""
		pkg := &DubboPackage{
			Body: &ResponsePayload{
				RspObj: &rspObj,
			},
		}
		err := unmarshalResponseBody(encoder.Buffer(), pkg)
		require.NoError(t, err)

		response := EnsureResponsePayload(pkg.Body)
		assert.NotNil(t, response.Attachments)
		assert.Equal(t, "value", response.Attachments["key"])
	})

	t.Run("response with exception", func(t *testing.T) {
		encoder := hessian.NewEncoder()
		_ = encoder.Encode(RESPONSE_WITH_EXCEPTION)
		_ = encoder.Encode(java_exception.NewThrowable("test error"))

		pkg := &DubboPackage{}
		err := unmarshalResponseBody(encoder.Buffer(), pkg)
		require.NoError(t, err)

		response := EnsureResponsePayload(pkg.Body)
		assert.Error(t, response.Exception)
	})

	t.Run("response with generic exception", func(t *testing.T) {
		encoder := hessian.NewEncoder()
		_ = encoder.Encode(RESPONSE_WITH_EXCEPTION)
		_ = encoder.Encode(java_exception.NewDubboGenericException("com.example.UserNotFoundException", "user not found"))

		pkg := &DubboPackage{}
		err := unmarshalResponseBody(encoder.Buffer(), pkg)
		require.NoError(t, err)

		response := EnsureResponsePayload(pkg.Body)
		ge, ok := response.Exception.(*hessian2.GenericException)
		require.True(t, ok)
		assert.Equal(t, "com.example.UserNotFoundException", ge.ExceptionClass)
		assert.Equal(t, "user not found", ge.ExceptionMessage)
	})

	t.Run("response with exception and attachments", func(t *testing.T) {
		encoder := hessian.NewEncoder()
		_ = encoder.Encode(RESPONSE_WITH_EXCEPTION_WITH_ATTACHMENTS)
		_ = encoder.Encode(java_exception.NewThrowable("test error"))
		_ = encoder.Encode(map[any]any{"key": "value"})

		pkg := &DubboPackage{}
		err := unmarshalResponseBody(encoder.Buffer(), pkg)
		require.NoError(t, err)

		response := EnsureResponsePayload(pkg.Body)
		require.Error(t, response.Exception)
		assert.NotNil(t, response.Attachments)
	})

	t.Run("response with null value", func(t *testing.T) {
		encoder := hessian.NewEncoder()
		_ = encoder.Encode(RESPONSE_NULL_VALUE)

		pkg := &DubboPackage{}
		err := unmarshalResponseBody(encoder.Buffer(), pkg)
		assert.NoError(t, err)
	})

	t.Run("response with null value and attachments", func(t *testing.T) {
		encoder := hessian.NewEncoder()
		_ = encoder.Encode(RESPONSE_NULL_VALUE_WITH_ATTACHMENTS)
		_ = encoder.Encode(map[any]any{"key": "value"})

		pkg := &DubboPackage{}
		err := unmarshalResponseBody(encoder.Buffer(), pkg)
		require.NoError(t, err)

		response := EnsureResponsePayload(pkg.Body)
		assert.NotNil(t, response.Attachments)
	})

	t.Run("response with non-error exception", func(t *testing.T) {
		encoder := hessian.NewEncoder()
		_ = encoder.Encode(RESPONSE_WITH_EXCEPTION)
		_ = encoder.Encode("string exception")

		pkg := &DubboPackage{}
		err := unmarshalResponseBody(encoder.Buffer(), pkg)
		require.NoError(t, err)

		response := EnsureResponsePayload(pkg.Body)
		assert.Error(t, response.Exception)
	})

	t.Run("response with legacy exception string", func(t *testing.T) {
		encoder := hessian.NewEncoder()
		_ = encoder.Encode(RESPONSE_WITH_EXCEPTION)
		_ = encoder.Encode("java exception: user not found")

		pkg := &DubboPackage{}
		err := unmarshalResponseBody(encoder.Buffer(), pkg)
		require.NoError(t, err)

		response := EnsureResponsePayload(pkg.Body)
		ge, ok := response.Exception.(*hessian2.GenericException)
		require.True(t, ok)
		assert.Equal(t, "java.lang.Exception", ge.ExceptionClass)
		assert.Equal(t, "user not found", ge.ExceptionMessage)
	})

	t.Run("response with invalid attachments", func(t *testing.T) {
		encoder := hessian.NewEncoder()
		_ = encoder.Encode(RESPONSE_VALUE_WITH_ATTACHMENTS)
		_ = encoder.Encode("test")
		_ = encoder.Encode("invalid attachments")

		pkg := &DubboPackage{}
		err := unmarshalResponseBody(encoder.Buffer(), pkg)
		assert.Error(t, err)
	})
}

func TestBuildServerSidePackageBody(t *testing.T) {
	// Register a test service
	common.ServiceMap.Register("com.test.Interface", "dubbo", "testGroup", "1.0.0", &TestService{})

	t.Run("build package body", func(t *testing.T) {
		req := []any{
			"2.0.2",
			"com.test.Path",
			"1.0.0",
			"test",
			"Ljava/lang/String;",
			[]any{"arg1"},
			map[string]any{
				"path":      "com.test.Path",
				"interface": "com.test.Interface",
				"group":     "testGroup",
			},
		}

		pkg := &DubboPackage{
			Body: req,
		}

		buildServerSidePackageBody(pkg)

		body, ok := pkg.GetBody().(map[string]any)
		assert.True(t, ok)
		assert.Equal(t, "2.0.2", body["dubboVersion"])
		assert.Equal(t, "Ljava/lang/String;", body["argsTypes"])
		assert.NotNil(t, body["attachments"])
	})

	t.Run("empty body", func(t *testing.T) {
		pkg := &DubboPackage{
			Body: []any{},
		}

		buildServerSidePackageBody(pkg)
		// Should not panic
	})
}

type TestService struct{}

func TestHessianSerializer_Marshal(t *testing.T) {
	serializer := HessianSerializer{}

	t.Run("marshal request", func(t *testing.T) {
		pkg := DubboPackage{
			Header: DubboHeader{
				Type: PackageRequest,
			},
			Service: Service{
				Path:    "test.Service",
				Version: "1.0.0",
				Method:  "test",
			},
			Body: &RequestPayload{
				Params:      []any{int32(123)},
				Attachments: map[string]any{},
			},
		}

		data, err := serializer.Marshal(pkg)
		require.NoError(t, err)
		assert.NotNil(t, data)
	})

	t.Run("marshal response", func(t *testing.T) {
		pkg := DubboPackage{
			Header: DubboHeader{
				Type:           PackageResponse,
				ResponseStatus: Response_OK,
			},
			Body: &ResponsePayload{
				RspObj:      "test",
				Attachments: map[string]any{},
			},
		}

		data, err := serializer.Marshal(pkg)
		require.NoError(t, err)
		assert.NotNil(t, data)
	})
}

func TestHessianSerializer_Unmarshal(t *testing.T) {
	serializer := HessianSerializer{}

	t.Run("unmarshal heartbeat", func(t *testing.T) {
		pkg := &DubboPackage{
			Header: DubboHeader{
				Type: PackageHeartbeat | PackageResponse,
			},
		}

		err := serializer.Unmarshal([]byte{}, pkg)
		assert.NoError(t, err)
	})

	t.Run("unmarshal request", func(t *testing.T) {
		encoder := hessian.NewEncoder()
		_ = encoder.Encode("2.0.2")
		_ = encoder.Encode("com.test.Service")
		_ = encoder.Encode("1.0.0")
		_ = encoder.Encode("test")
		_ = encoder.Encode("")
		_ = encoder.Encode(map[any]any{"key": "value"})

		pkg := &DubboPackage{
			Header: DubboHeader{
				Type: PackageRequest,
			},
		}

		err := serializer.Unmarshal(encoder.Buffer(), pkg)
		assert.NoError(t, err)
	})

	t.Run("unmarshal response", func(t *testing.T) {
		encoder := hessian.NewEncoder()
		_ = encoder.Encode(RESPONSE_VALUE)
		_ = encoder.Encode("test")

		rspObj := ""
		pkg := &DubboPackage{
			Header: DubboHeader{
				Type: PackageResponse,
			},
			Body: &ResponsePayload{
				RspObj: &rspObj,
			},
		}

		err := serializer.Unmarshal(encoder.Buffer(), pkg)
		require.NoError(t, err)
		assert.Equal(t, "test", rspObj)
	})
}
