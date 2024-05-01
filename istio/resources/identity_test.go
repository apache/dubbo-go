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

package resources

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseIdentity(t *testing.T) {
	tests := []struct {
		name           string
		input          string
		expectedOutput Identity
		expectedError  error
	}{
		{
			name:  "Valid SPIFFE identity with custom trust domain",
			input: "spiffe://example.org/ns/default/sa/my-service-account",
			expectedOutput: Identity{
				TrustDomain:    "example.org",
				Namespace:      "default",
				ServiceAccount: "my-service-account",
			},
			expectedError: nil,
		},
		{
			name:  "Valid SPIFFE identity with default trust domain",
			input: "spiffe://cluster.local/ns/my-namespace/sa/my-sa",
			expectedOutput: Identity{
				TrustDomain:    "cluster.local",
				Namespace:      "my-namespace",
				ServiceAccount: "my-sa",
			},
			expectedError: nil,
		},
		{
			name:           "Invalid SPIFFE identity - missing segments",
			input:          "spiffe://not-a-spiffe-format",
			expectedOutput: Identity{},
			expectedError:  fmt.Errorf("identity is not a spiffe format"),
		},
		{
			name:           "Invalid SPIFFE identity - incorrect segment order",
			input:          "spiffe://example.org/ns/default/invalid-segment/sa/my-sa",
			expectedOutput: Identity{},
			expectedError:  fmt.Errorf("identity is not a spiffe format"),
		},
		{
			name:           "Invalid SPIFFE identity - wrong scheme",
			input:          "not-a-spiffe://example.org/ns/default/sa/my-sa",
			expectedOutput: Identity{},
			expectedError:  fmt.Errorf("identity is not a spiffe format"),
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			output, err := ParseIdentity(tc.input)
			assert.Equal(t, tc.expectedOutput, output)
			assert.Equal(t, tc.expectedError, err)
		})
	}
}
