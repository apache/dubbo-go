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
	"bytes"
	"context"
	"fmt"
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

func TestNewIncomingContextClonesHeaders(t *testing.T) {
	baseCtx := NewOutgoingContext(context.Background(), http.Header{
		"Request-Id": []string{"outgoing"},
	})
	inputValues := []string{"incoming"}
	input := http.Header{
		"request-id": inputValues,
	}

	ctx := newIncomingContext(baseCtx, input)
	incoming, ok := FromIncomingContext(ctx)
	assert.True(t, ok)
	incoming.Values("Request-Id")[0] = "changed"
	incoming.Add("Another", "value")

	assert.Equal(t, []string{"incoming"}, inputValues)
	outgoing := ExtractFromOutgoingContext(baseCtx)
	assert.Equal(t, []string{"outgoing"}, outgoing.Values("Request-Id"))
}

func ExampleNewOutgoingContext() {
	ctx := NewOutgoingContext(context.Background(), http.Header{
		"hello": []string{"triple"},
	})
	ctx = AppendToOutgoingContext(ctx, "hello", "dubbo", "hey", "hessian")

	headers := ExtractFromOutgoingContext(ctx)
	fmt.Println(headers.Values("hello"))
	fmt.Println(headers.Get("hey"))

	// Output:
	// [triple dubbo]
	// hessian
}
