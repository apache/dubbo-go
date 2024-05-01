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
	"strings"
	"testing"
)

func TestMatchSpiffe(t *testing.T) {
	tests := []struct {
		name     string
		spiffee  string
		action   string
		value    string
		expected bool
	}{
		{
			name:     "exact match",
			spiffee:  "spiffe://example.org/ns/my-ns/sa/default",
			action:   "exact",
			value:    "spiffe://example.org/ns/my-ns/sa/default",
			expected: true,
		},
		{
			name:     "prefix match",
			spiffee:  "spiffe://example.org/ns/my-ns/sa/default",
			action:   "prefix",
			value:    "spiffe://example.org/ns/my-ns/",
			expected: true,
		},
		{
			name:     "contains match",
			spiffee:  "spiffe://example.org/ns/my-ns/sa/default",
			action:   "contains",
			value:    "my-ns/sa",
			expected: true,
		},
		{
			name:     "invalid action",
			spiffee:  "spiffe://example.org/ns/my-ns/sa/default",
			action:   "invalid-action",
			value:    "something",
			expected: false,
		},
		{
			name:     "empty action",
			spiffee:  "spiffe://example.org/ns/my-ns/sa/default",
			action:   "",
			value:    "something",
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := MatchSpiffe(strings.ToLower(tt.spiffee), tt.action, strings.ToLower(tt.value))
			if result != tt.expected {
				t.Errorf("MatchSpiffe('%s', '%s', '%s') expected %t but got %t", tt.spiffee, tt.action, tt.value, tt.expected, result)
			}
		})
	}
}
