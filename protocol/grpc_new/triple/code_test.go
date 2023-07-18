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

package triple

import (
	"strconv"
	"strings"
	"testing"

	"dubbo.apache.org/dubbo-go/v3/protocol/grpc_new/connect/assert"
)

func TestCode(t *testing.T) {
	t.Parallel()
	var valid []Code
	for code := minCode; code <= maxCode; code++ {
		valid = append(valid, code)
	}
	// Ensures that we don't forget to update the mapping in the Stringer
	// implementation.
	for _, code := range valid {
		assert.False(
			t,
			strings.HasPrefix(code.String(), "code_"),
			assert.Sprintf("update Code.String() method for new code %v", code),
		)
		assertCodeRoundTrips(t, code)
	}
	assertCodeRoundTrips(t, Code(999))
}

func assertCodeRoundTrips(tb testing.TB, code Code) {
	tb.Helper()
	encoded, err := code.MarshalText()
	assert.Nil(tb, err)
	var decoded Code
	assert.Nil(tb, decoded.UnmarshalText(encoded))
	assert.Equal(tb, decoded, code)
	if code >= minCode && code <= maxCode {
		var invalid Code
		// For the known codes, we only accept the canonical string representation: "canceled", not "code_1".
		assert.NotNil(tb, invalid.UnmarshalText([]byte("code_"+strconv.Itoa(int(code)))))
	}
}
