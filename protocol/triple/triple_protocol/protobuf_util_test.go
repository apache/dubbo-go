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
	"testing"
)

import (
	"dubbo.apache.org/dubbo-go/v3/protocol/triple/triple_protocol/assert"
)

func TestParseProtobufURL(t *testing.T) {
	t.Parallel()
	assertExtractedProtoPath(
		t,
		// full URL
		"https://api.foo.com/grpc/foo.user.v1.UserService/GetUser",
		"/foo.user.v1.UserService/GetUser",
	)
	assertExtractedProtoPath(
		t,
		// rooted path
		"/foo.user.v1.UserService/GetUser",
		"/foo.user.v1.UserService/GetUser",
	)
	assertExtractedProtoPath(
		t,
		// path without leading or trailing slashes
		"foo.user.v1.UserService/GetUser",
		"/foo.user.v1.UserService/GetUser",
	)
	assertExtractedProtoPath(
		t,
		// path with trailing slash
		"/foo.user.v1.UserService.GetUser/",
		"/foo.user.v1.UserService.GetUser",
	)
	// edge cases
	assertExtractedProtoPath(t, "", "/")
	assertExtractedProtoPath(t, "//", "/")
}

func assertExtractedProtoPath(tb testing.TB, inputURL, expectPath string) {
	tb.Helper()
	assert.Equal(
		tb,
		extractProtoPath(inputURL),
		expectPath,
	)
}
