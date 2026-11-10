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

package dubboutil

import (
	"encoding/json"
	"testing"
)

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEncodeJSON(t *testing.T) {
	encoded, err := EncodeJSON(struct {
		Name  string `json:"name"`
		Count int    `json:"count"`
	}{
		Name:  "dubbo-go",
		Count: 2,
	})
	require.NoError(t, err)
	assert.JSONEq(t, `{"name":"dubbo-go","count":2}`, string(encoded))
}

func TestEncodeJSONNilInput(t *testing.T) {
	encoded, err := EncodeJSON(nil)
	require.EqualError(t, err, "input for encoding is nil")
	assert.Nil(t, encoded)
}

func TestEncodeJSONMarshalError(t *testing.T) {
	encoded, err := EncodeJSON(make(chan int))
	require.Error(t, err)
	require.ErrorContains(t, err, "encode JSON: json: unsupported type: chan int")
	assert.Nil(t, encoded)
}

func TestDecodeJSON(t *testing.T) {
	var decoded struct {
		Name  string `json:"name"`
		Count int    `json:"count"`
	}
	err := DecodeJSON([]byte(`{"name":"dubbo-go","count":2}`), &decoded)
	require.NoError(t, err)
	assert.Equal(t, "dubbo-go", decoded.Name)
	assert.Equal(t, 2, decoded.Count)
}

func TestDecodeJSONUseNumber(t *testing.T) {
	const largeInteger = int64(1<<53 + 1)

	var decoded map[string]any
	err := DecodeJSON([]byte(`{"int":9007199254740993,"float":1.25,"nullVal":null}`), &decoded)
	require.NoError(t, err)
	assert.Equal(t, json.Number("9007199254740993"), decoded["int"])
	decodedInteger, err := decoded["int"].(json.Number).Int64()
	require.NoError(t, err)
	assert.Equal(t, largeInteger, decodedInteger)
	assert.Equal(t, json.Number("1.25"), decoded["float"])
	assert.Nil(t, decoded["nullVal"])
}

func TestDecodeJSONInvalidArguments(t *testing.T) {
	tests := []struct {
		name string
		data []byte
		out  any
		err  string
	}{
		{
			name: "nil data",
			out:  &map[string]any{},
			err:  "'data' being decoded is nil",
		},
		{
			name: "empty data",
			data: []byte{},
			out:  &map[string]any{},
			err:  "'data' being decoded is nil",
		},
		{
			name: "nil output",
			data: []byte(`{}`),
			err:  "output parameter 'out' is nil",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := DecodeJSON(test.data, test.out)
			require.EqualError(t, err, test.err)
		})
	}
}

func TestDecodeJSONInvalidData(t *testing.T) {
	var decoded map[string]any
	err := DecodeJSON([]byte(`{"value":`), &decoded)
	require.Error(t, err)
	assert.ErrorContains(t, err, "unexpected EOF")
}

func TestDecodeJSONTypeError(t *testing.T) {
	var decoded struct {
		Value int `json:"value"`
	}
	err := DecodeJSON([]byte(`{"value":3.14159}`), &decoded)
	require.Error(t, err)
	assert.ErrorContains(t, err, "cannot unmarshal number")
}
