// Copyright 2021-2023 Buf Technologies, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package triple_protocol

import (
	"bytes"
	"context"
	"net/http"
	"testing"
	"testing/quick"
)

import (
	"dubbo.apache.org/dubbo-go/v3/protocol/triple/triple_protocol/internal/assert"
)

func TestBinaryEncodingQuick(t *testing.T) {
	t.Parallel()
	roundtrip := func(binary []byte) bool {
		encoded := EncodeBinaryHeader(binary)
		decoded, err := DecodeBinaryHeader(encoded)
		if err != nil {
			// We want to abort immediately. Don't use our assert package.
			t.Fatalf("decode error: %v", err)
		}
		return bytes.Equal(decoded, binary)
	}
	if err := quick.Check(roundtrip, nil /* config */); err != nil {
		t.Error(err)
	}
}

func TestHeaderMerge(t *testing.T) {
	t.Parallel()
	header := http.Header{
		"Foo": []string{"one"},
	}
	mergeHeaders(header, http.Header{
		"Foo": []string{"two"},
		"Bar": []string{"one"},
		"Baz": nil,
	})
	expect := http.Header{
		"Foo": []string{"one", "two"},
		"Bar": []string{"one"},
		"Baz": nil,
	}
	assert.Equal(t, header, expect)
}

func TestNewOutgoingContext(t *testing.T) {
	tests := []struct {
		desc   string
		header http.Header
	}{
		{
			desc:   "nil",
			header: nil,
		},
		{
			desc:   "empty Header",
			header: http.Header{},
		},
		{
			desc: "normal Header",
			header: http.Header{
				"test-key": []string{
					"test-val",
				},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			ctx := context.Background()
			ctx = NewOutgoingContext(ctx, test.header)
			header := retrieveFromOutgoingContext(ctx)
			assert.Equal(t, header, test.header)
			mergeHeaders(test.header, header)
		})
	}
}

func TestAppendToOutgoingContext(t *testing.T) {
	tests := []struct {
		desc        string
		kvs         []string
		expect      http.Header
		expectPanic bool
	}{
		{
			desc: "keys and vals do not come in pairs",
			kvs: []string{
				"key",
			},
			expectPanic: true,
		},
		{
			desc:   "nil kvs",
			kvs:    nil,
			expect: nil,
		},
		{
			desc: "normal key and val",
			kvs: []string{
				"key",
				"val",
			},
			expect: http.Header{
				"key": []string{
					"val",
				},
			},
		},
		{
			desc: "a key with multiple vals",
			kvs: []string{
				"key",
				"val0",
				"key",
				"val1",
			},
			expect: http.Header{
				"key": []string{
					"val0",
					"val1",
				},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			defer func() {
				if e := recover(); e != nil && !test.expectPanic {
					t.Fatal(e)
				}
			}()

			ctx := AppendToOutgoingContext(context.Background(), test.kvs...)
			header := retrieveFromOutgoingContext(ctx)
			assert.Equal(t, header, test.expect)
		})
	}
}
