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

package common

import (
	"testing"
)

import (
	"github.com/stretchr/testify/assert"
)

func TestParamMatchIsMatch(t *testing.T) {
	u, _ := NewURL("dubbo://127.0.0.1:20000?app=test&version=1.0")

	tests := []struct {
		name     string
		param    ParamMatch
		expected bool
	}{
		{
			name:     "exact match",
			param:    ParamMatch{Key: "app", Value: StringMatch{Exact: "test"}},
			expected: true,
		},
		{
			name:     "exact not match",
			param:    ParamMatch{Key: "app", Value: StringMatch{Exact: "other"}},
			expected: false,
		},
		{
			name:     "key not exists",
			param:    ParamMatch{Key: "nonexistent", Value: StringMatch{Exact: ""}},
			expected: false,
		},
		{
			name:     "prefix match",
			param:    ParamMatch{Key: "version", Value: StringMatch{Prefix: "1."}},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.param.IsMatch(u))
		})
	}
}

func TestStringMatchIsMatch(t *testing.T) {
	tests := []struct {
		name     string
		match    StringMatch
		value    string
		expected bool
	}{
		// Exact match
		{"exact match", StringMatch{Exact: "hello"}, "hello", true},
		{"exact not match", StringMatch{Exact: "hello"}, "world", false},
		{"exact empty value", StringMatch{Exact: "hello"}, "", false},

		// Prefix match
		{"prefix match", StringMatch{Prefix: "hello"}, "hello world", true},
		{"prefix exact", StringMatch{Prefix: "hello"}, "hello", true},
		{"prefix not match", StringMatch{Prefix: "hello"}, "world", false},
		{"prefix empty value", StringMatch{Prefix: "hello"}, "", false},

		// Regex match
		{"regex match", StringMatch{Regex: "^hello.*"}, "hello world", true},
		{"regex not match", StringMatch{Regex: "^hello.*"}, "world", false},
		{"regex invalid pattern", StringMatch{Regex: "[invalid"}, "test", false},

		// Wildcard match
		{"wildcard exact", StringMatch{Wildcard: "hello"}, "hello", true},
		{"wildcard any", StringMatch{Wildcard: "*"}, "anything", true},
		{"wildcard not match", StringMatch{Wildcard: "hello"}, "world", false},

		// Empty match
		{"empty match", StringMatch{Empty: "true"}, "", true},
		{"empty not match", StringMatch{Empty: "true"}, "value", false},

		// Noempty match
		{"noempty match", StringMatch{Noempty: "true"}, "value", true},
		{"noempty not match", StringMatch{Noempty: "true"}, "", false},

		// No match condition
		{"no condition", StringMatch{}, "value", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.match.IsMatch(tt.value))
		})
	}
}

func TestAddressMatchIsMatch(t *testing.T) {
	tests := []struct {
		name     string
		match    AddressMatch
		value    string
		expected bool
	}{
		// CIDR match
		{"cidr match", AddressMatch{Cird: "192.168.1.0/24"}, "192.168.1.100", true},
		{"cidr not match", AddressMatch{Cird: "192.168.1.0/24"}, "192.168.2.100", false},
		{"cidr invalid", AddressMatch{Cird: "invalid"}, "192.168.1.1", false},
		{"cidr empty value", AddressMatch{Cird: "192.168.1.0/24"}, "", false},

		// Wildcard match
		{"wildcard any value *", AddressMatch{Wildcard: "192.*"}, "*", true},
		{"wildcard any host 0.0.0.0", AddressMatch{Wildcard: "192.*"}, "0.0.0.0", true},
		{"wildcard pattern match", AddressMatch{Wildcard: "192.168.*"}, "192.168.1.1", true},
		{"wildcard pattern not match", AddressMatch{Wildcard: "192.168.*"}, "10.0.0.1", false},
		{"wildcard empty value", AddressMatch{Wildcard: "192.*"}, "", false},

		// Exact match
		{"exact match", AddressMatch{Exact: "192.168.1.1"}, "192.168.1.1", true},
		{"exact not match", AddressMatch{Exact: "192.168.1.1"}, "192.168.1.2", false},
		{"exact empty value", AddressMatch{Exact: "192.168.1.1"}, "", false},

		// No condition
		{"no condition", AddressMatch{}, "192.168.1.1", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.match.IsMatch(tt.value))
		})
	}
}

func TestListStringMatchIsMatch(t *testing.T) {
	tests := []struct {
		name     string
		match    ListStringMatch
		value    string
		expected bool
	}{
		{
			name: "match first",
			match: ListStringMatch{
				Oneof: []StringMatch{
					{Exact: "hello"},
					{Exact: "world"},
				},
			},
			value:    "hello",
			expected: true,
		},
		{
			name: "match second",
			match: ListStringMatch{
				Oneof: []StringMatch{
					{Exact: "hello"},
					{Exact: "world"},
				},
			},
			value:    "world",
			expected: true,
		},
		{
			name: "no match",
			match: ListStringMatch{
				Oneof: []StringMatch{
					{Exact: "hello"},
					{Exact: "world"},
				},
			},
			value:    "other",
			expected: false,
		},
		{
			name:     "empty list",
			match:    ListStringMatch{Oneof: []StringMatch{}},
			value:    "hello",
			expected: false,
		},
		{
			name: "mixed match types",
			match: ListStringMatch{
				Oneof: []StringMatch{
					{Prefix: "hello"},
					{Regex: "^world.*"},
				},
			},
			value:    "world123",
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.match.IsMatch(tt.value))
		})
	}
}
