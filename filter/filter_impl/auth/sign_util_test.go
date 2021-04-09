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
		// TODO: Add test cases.
		{"test1", args{s: "   ", allowSpace: false}, true},
		{"test2", args{s: "   ", allowSpace: true}, false},
		{"test3", args{s: "hello,dubbo", allowSpace: false}, false},
		{"test4", args{s: "hello,dubbo", allowSpace: true}, false},
		{"test5", args{s: "", allowSpace: true}, true},
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
	params := []interface{}{
		"a", 1, struct {
			Name string
			ID   int64
		}{"YuYu", 1},
	}
	signature, _ := SignWithParams(params, metadata, key)
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
	params := []interface{}{
		"a", 1, struct {
			Name string
			ID   int64
		}{"YuYu", 1},
	}
	params2 := []interface{}{
		"a", 1, struct {
			Name string
			ID   int64
		}{"YuYu", 1},
	}
	jsonBytes, _ := toBytes(params)
	jsonBytes2, _ := toBytes(params2)
	assert.NotNil(t, jsonBytes)
	assert.Equal(t, jsonBytes, jsonBytes2)
}
