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

package auth

import (
	"testing"
)

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIsEmpty(t *testing.T) {
	type args struct {
		s          string
		allowSpace bool
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{"whitespace_false", args{s: "   ", allowSpace: false}, true},
		{"whitespace_true", args{s: "   ", allowSpace: true}, false},
		{"normal_false", args{s: "hello,dubbo", allowSpace: false}, false},
		{"normal_true", args{s: "hello,dubbo", allowSpace: true}, false},
		{"empty_true", args{s: "", allowSpace: true}, true},
		{"empty_false", args{s: "", allowSpace: false}, true},
		{"single_space_false", args{s: " ", allowSpace: false}, true},
		{"single_space_true", args{s: " ", allowSpace: true}, false},
		{"single_tab_false", args{s: "\t", allowSpace: false}, true},
		{"single_tab_true", args{s: "\t", allowSpace: true}, false},
		{"mixed_whitespace_false", args{s: " \t\n\r ", allowSpace: false}, true},
		{"mixed_whitespace_true", args{s: " \t\n\r ", allowSpace: true}, false},
		{"leading_space_false", args{s: "  hello", allowSpace: false}, false},
		{"leading_space_true", args{s: "  hello", allowSpace: true}, false},
		{"trailing_space_false", args{s: "hello  ", allowSpace: false}, false},
		{"trailing_space_true", args{s: "hello  ", allowSpace: true}, false},
		{"both_ends_false", args{s: "  hello  ", allowSpace: false}, false},
		{"both_ends_true", args{s: "  hello  ", allowSpace: true}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsEmpty(tt.args.s, tt.args.allowSpace); got != tt.want {
				t.Errorf("IsEmpty() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSign(t *testing.T) {
	metadata := "com.ikurento.user.UserProvider::sayHi"
	key := "key"
	signature := Sign(metadata, key)
	assert.NotNil(t, signature)
}

func TestSignWithParams(t *testing.T) {
	metadata := "com.ikurento.user.UserProvider::sayHi"
	key := "key"
	params := []any{
		"a", 1, struct {
			Name string
			ID   int64
		}{"YuYu", 1},
	}
	signature, err := SignWithParams(params, metadata, key)
	require.NoError(t, err)
	assert.False(t, IsEmpty(signature, false))
}

func Test_doSign(t *testing.T) {
	sign := doSign([]byte("DubboGo"), "key")
	sign1 := doSign([]byte("DubboGo"), "key")
	sign2 := doSign([]byte("DubboGo"), "key2")
	assert.NotNil(t, sign)
	assert.Equal(t, sign1, sign)
	assert.NotEqual(t, sign1, sign2)
}

func Test_toBytes(t *testing.T) {
	params := []any{
		"a", 1, struct {
			Name string
			ID   int64
		}{"YuYu", 1},
	}
	params2 := []any{
		"a", 1, struct {
			Name string
			ID   int64
		}{"YuYu", 1},
	}
	jsonBytes, err := toBytes(params)
	require.NoError(t, err)
	assert.NotNil(t, jsonBytes)
	jsonBytes2, err2 := toBytes(params2)
	require.NoError(t, err2)
	assert.NotNil(t, jsonBytes2)
	assert.JSONEq(t, string(jsonBytes), string(jsonBytes2))
}
