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

package triple_protocol

import (
	"testing"
)

import (
	"dubbo.apache.org/dubbo-go/v3/protocol/triple/triple_protocol/internal/assert"
)

func TestCanonicalizeContentType(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		arg  string
		want string
	}{
		// slow path: base type needs case normalization
		{name: "uppercase should be normalized", arg: "APPLICATION/json", want: "application/json"},
		{name: "no parameters should be normalized", arg: "APPLICATION/json;  ", want: "application/json"},
		// slow path: non-charset parameter
		{name: "non charset param should not be changed", arg: "multipart/form-data; boundary=fooBar", want: "multipart/form-data; boundary=fooBar"},
		// fast path: charset parameter normalization
		{name: "charset param uppercase normalized via fast path", arg: "application/json; charset=UTF-8", want: "application/json; charset=utf-8"},
		{name: "charset param already lowercase returned as-is", arg: "application/json; charset=utf-8", want: "application/json; charset=utf-8"},
		// malformed: must fall through to slow path unchanged (not rewritten by fast path)
		{name: "malformed missing type", arg: "/json; charset=UTF-8", want: "/json; charset=UTF-8"},
		{name: "malformed missing subtype", arg: "application/; charset=UTF-8", want: "application/; charset=UTF-8"},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, canonicalizeContentType(tt.arg), tt.want)
		})
	}
}

func BenchmarkCanonicalizeContentType(b *testing.B) {
	b.Run("simple", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			_ = canonicalizeContentType("application/json")
		}
	})

	b.Run("charset canonical", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			_ = canonicalizeContentType("application/json; charset=utf-8")
		}
	})

	b.Run("charset uppercase", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			_ = canonicalizeContentType("application/json; charset=UTF-8")
		}
	})

	b.Run("non-charset param", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			_ = canonicalizeContentType("application/json; foo=utf-8")
		}
	})
}
