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

package trace

import (
	"reflect"
	"testing"
)

func Test_metadataSupplier_Keys(t *testing.T) {
	tests := []struct {
		name     string
		metadata map[string]interface{}
		want     []string
	}{
		{
			name: "test",
			metadata: map[string]interface{}{
				"key1": nil,
			},
			want: []string{"key1"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &metadataSupplier{
				metadata: tt.metadata,
			}
			if got := s.Keys(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Keys() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_metadataSupplier_Set(t *testing.T) {
	tests := []struct {
		name     string
		metadata map[string]interface{}
		key      string
		value    string
	}{
		{
			name:     "test",
			metadata: nil,
			key:      "key",
			value:    "value",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &metadataSupplier{
				metadata: tt.metadata,
			}
			s.Set(tt.key, tt.value)
		})
	}
}

func Test_metadataSupplier_Get(t *testing.T) {
	tests := []struct {
		name     string
		metadata map[string]interface{}
		key      string
		want     string
	}{
		{
			name:     "nil metadata",
			metadata: nil,
			key:      "key",
			want:     "",
		},
		{
			name: "not exist",
			metadata: map[string]interface{}{
				"k": nil,
			},
			key:  "key",
			want: "",
		},
		{
			name: "nil slice",
			metadata: map[string]interface{}{
				"key": nil,
			},
			key:  "key",
			want: "",
		},
		{
			name: "empty slice",
			metadata: map[string]interface{}{
				"key": []string{},
			},
			key:  "key",
			want: "",
		},
		{
			name: "test",
			metadata: map[string]interface{}{
				"key": []string{"test"},
			},
			key:  "key",
			want: "test",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &metadataSupplier{
				metadata: tt.metadata,
			}
			if got := s.Get(tt.key); got != tt.want {
				t.Errorf("Get() = %v, want %v", got, tt.want)
			}
		})
	}
}
