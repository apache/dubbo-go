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

package connect

import (
	"bytes"
	"net/http"
	"testing"
	"testing/quick"

	"dubbo.apache.org/dubbo-go/v3/protocol/grpc_new/connect/assert"
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
