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

package jsonrpc

import (
	"encoding/json"
	"testing"
)

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type TestData struct {
	Test string
}

func TestJsonClientCodecWrite(t *testing.T) {
	cd := &CodecData{
		ID:     1,
		Method: "GetUser",
		Args:   []any{"args", 2},
	}
	codec := newJsonClientCodec()
	data, err := codec.Write(cd)
	require.NoError(t, err)
	assert.JSONEq(t, "{\"jsonrpc\":\"2.0\",\"method\":\"GetUser\",\"params\":[\"args\",2],\"id\":1}\n", string(data))

	cd.Args = 1
	_, err = codec.Write(cd)
	require.EqualError(t, err, "unsupported param type: int")
}

func TestJsonClientCodecRead(t *testing.T) {
	codec := newJsonClientCodec()
	codec.pending[1] = "GetUser"
	rsp := &TestData{}
	err := codec.Read([]byte("{\"jsonrpc\":\"2.0\",\"id\":1,\"result\":{\"Test\":\"test\"}}\n"), rsp)
	require.NoError(t, err)
	assert.Equal(t, "test", rsp.Test)

	// error
	codec.pending[1] = "GetUser"
	err = codec.Read([]byte("{\"jsonrpc\":\"2.0\",\"id\":1,\"error\":{\"code\":-32000,\"message\":\"error\"}}\n"), rsp)
	require.EqualError(t, err, "{\"code\":-32000,\"message\":\"error\"}")
}

func TestServerCodecWrite(t *testing.T) {
	codec := newServerCodec()
	a := json.RawMessage([]byte("1"))
	codec.req = serverRequest{Version: "1.0", Method: "GetUser", ID: &a}
	data, err := codec.Write("error", &TestData{Test: "test"})
	require.NoError(t, err)
	assert.JSONEq(t, "{\"jsonrpc\":\"2.0\",\"id\":1,\"result\":{\"Test\":\"test\"},\"error\":{\"code\":-32000,\"message\":\"error\"}}\n", string(data))

	data, err = codec.Write("{\"code\":-32000,\"message\":\"error\"}", &TestData{Test: "test"})
	require.NoError(t, err)
	assert.JSONEq(t, "{\"jsonrpc\":\"2.0\",\"id\":1,\"result\":{\"Test\":\"test\"},\"error\":{\"code\":-32000,\"message\":\"error\"}}\n", string(data))
}

func TestServerCodecRead(t *testing.T) {
	codec := newServerCodec()
	header := map[string]string{}
	err := codec.ReadHeader(header, []byte("{\"jsonrpc\":\"2.0\",\"method\":\"GetUser\",\"params\":[\"args\",2],\"id\":1}\n"))
	require.EqualError(t, err, "{\"code\":-32601,\"message\":\"Method not found\"}")

	header["HttpMethod"] = "POST"
	err = codec.ReadHeader(header, []byte("{\"jsonrpc\":\"2.0\",\"method\":\"GetUser\",\"params\":[\"args\",2],\"id\":1}\n"))
	require.NoError(t, err)
	assert.Equal(t, "1", string([]byte(*codec.req.ID)))
	assert.Equal(t, "GetUser", codec.req.Method)
	assert.Equal(t, "2.0", codec.req.Version)
	assert.Equal(t, "[\"args\",2]", string([]byte(*codec.req.Params)))

	req := []any{}
	err = codec.ReadBody(&req)
	require.NoError(t, err)
	assert.Equal(t, "args", req[0])
	assert.InEpsilon(t, 2.0, req[1], 1e-9)
}
