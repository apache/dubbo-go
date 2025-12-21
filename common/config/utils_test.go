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

package config

import (
	"testing"
)

import (
	"github.com/stretchr/testify/assert"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common/constant"
)

func TestTranslateIds(t *testing.T) {
	tests := []struct {
		name     string
		input    []string
		expected []string
	}{
		{"single id", []string{"nacos"}, []string{"nacos"}},
		{"comma separated", []string{"nacos,zk"}, []string{"nacos", "zk"}},
		{"multiple entries", []string{"nacos", "zk"}, []string{"nacos", "zk"}},
		{"mixed", []string{"nacos,zk", "etcd"}, []string{"nacos", "zk", "etcd"}},
		{"duplicates", []string{"nacos,nacos"}, []string{"nacos"}},
		{"empty string", []string{""}, []string{}},
		{"empty slice", []string{}, []string{}},
		{"with empty parts", []string{"nacos,,zk"}, []string{"nacos", "zk"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := TranslateIds(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestRemoveDuplicateElement(t *testing.T) {
	tests := []struct {
		name     string
		input    []string
		expected []string
	}{
		{"no duplicates", []string{"a", "b", "c"}, []string{"a", "b", "c"}},
		{"with duplicates", []string{"a", "b", "a", "c", "b"}, []string{"a", "b", "c"}},
		{"all same", []string{"a", "a", "a"}, []string{"a"}},
		{"empty", []string{}, []string{}},
		{"with empty strings", []string{"a", "", "b", ""}, []string{"a", "b"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := removeDuplicateElement(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

type testStruct struct {
	Name string `validate:"required"`
	Age  int    `validate:"gte=0"`
}

func TestVerify(t *testing.T) {
	// Valid struct
	valid := &testStruct{Name: "test", Age: 10}
	err := Verify(valid)
	assert.NoError(t, err)

	// Invalid struct - missing required field
	invalid := &testStruct{Name: "", Age: 10}
	err = Verify(invalid)
	assert.Error(t, err)

	// Invalid struct - negative age
	invalid2 := &testStruct{Name: "test", Age: -1}
	err = Verify(invalid2)
	assert.Error(t, err)
}

func TestMergeValue(t *testing.T) {
	tests := []struct {
		name     string
		str1     string
		str2     string
		def      string
		expected string
	}{
		{"both empty use default", "", "", "default", "default"},
		{"str1 only", "a,b", "", "default", "default,a,b"},
		{"str2 only", "", "c,d", "default", "default,c,d"},
		{"both have values", "a", "b", "default", "default,a,b"},
		{"with default key", constant.DefaultKey + ",a", "b", "mydef", "mydef,a,b"},
		{"with minus", "a,-b", "b,c", "default", "default,a,c"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := MergeValue(tt.str1, tt.str2, tt.def)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestRemoveMinus(t *testing.T) {
	tests := []struct {
		name     string
		input    []string
		expected string
	}{
		{"empty", []string{}, ""},
		{"no minus", []string{"a", "b", "c"}, "a,b,c"},
		{"with minus", []string{"a", "-b", "b", "c"}, "a,c"},
		{"all minus", []string{"-a", "-b"}, ""},
		{"minus not found", []string{"a", "-x", "b"}, "a,b"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := removeMinus(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestIsValid(t *testing.T) {
	assert.True(t, IsValid("127.0.0.1:8080"))
	assert.True(t, IsValid("localhost"))
	assert.False(t, IsValid(""))
	assert.False(t, IsValid(constant.NotAvailable))
}
