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

package utils

import (
	"testing"
)

import (
	"github.com/stretchr/testify/assert"
)

func TestParseVersionSuccess(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected Version
	}{
		{
			name:     "lowercase v prefix",
			input:    "v1.0.0",
			expected: Version{1, 0, 0},
		},
		{
			name:     "uppercase V prefix",
			input:    "V2.1.3",
			expected: Version{2, 1, 3},
		},
		{
			name:     "two parts version",
			input:    "v3.1",
			expected: Version{3, 1},
		},
		{
			name:     "single part version",
			input:    "v5",
			expected: Version{5},
		},
		{
			name:     "four parts version",
			input:    "v1.2.3.4",
			expected: Version{1, 2, 3, 4},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			version, err := ParseVersion(tt.input)
			assert.NoError(t, err)
			assert.Equal(t, tt.expected, version)
		})
	}
}

func TestParseVersionError(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{
			name:  "no prefix",
			input: "1.0.0",
		},
		{
			name:  "illegal characters",
			input: "v1.0.0-alpha",
		},
		{
			name:  "empty string",
			input: "",
		},
		{
			name:  "only prefix",
			input: "v",
		},
		{
			name:  "with spaces",
			input: "v 1.0.0",
		},
		{
			name:  "invalid format",
			input: "version1.0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ParseVersion(tt.input)
			assert.Error(t, err)
		})
	}
}

func TestVersionEqual(t *testing.T) {
	tests := []struct {
		name     string
		v1       Version
		v2       Version
		expected bool
	}{
		{
			name:     "equal versions",
			v1:       Version{1, 0, 0},
			v2:       Version{1, 0, 0},
			expected: true,
		},
		{
			name:     "different versions",
			v1:       Version{1, 0, 0},
			v2:       Version{1, 0, 1},
			expected: false,
		},
		{
			name:     "different length not equal",
			v1:       Version{1, 0},
			v2:       Version{1, 0, 0},
			expected: false,
		},
		{
			name:     "empty versions",
			v1:       Version{},
			v2:       Version{},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.v1.Equal(tt.v2)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestVersionLess(t *testing.T) {
	tests := []struct {
		name     string
		v1       Version
		v2       Version
		expected bool
	}{
		{
			name:     "v1 less than v2",
			v1:       Version{1, 0, 0},
			v2:       Version{2, 0, 0},
			expected: true,
		},
		{
			name:     "v1 greater than v2",
			v1:       Version{2, 0, 0},
			v2:       Version{1, 0, 0},
			expected: false,
		},
		{
			name:     "equal versions",
			v1:       Version{1, 0, 0},
			v2:       Version{1, 0, 0},
			expected: false,
		},
		{
			name:     "minor version less",
			v1:       Version{1, 0, 0},
			v2:       Version{1, 1, 0},
			expected: true,
		},
		{
			name:     "patch version less",
			v1:       Version{1, 0, 0},
			v2:       Version{1, 0, 1},
			expected: true,
		},
		{
			name:     "shorter version less",
			v1:       Version{1, 0},
			v2:       Version{1, 0, 1},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.v1.Less(tt.v2)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestVersionGreater(t *testing.T) {
	tests := []struct {
		name     string
		v1       Version
		v2       Version
		expected bool
	}{
		{
			name:     "v1 greater than v2",
			v1:       Version{2, 0, 0},
			v2:       Version{1, 0, 0},
			expected: true,
		},
		{
			name:     "v1 less than v2",
			v1:       Version{1, 0, 0},
			v2:       Version{2, 0, 0},
			expected: false,
		},
		{
			name:     "equal versions",
			v1:       Version{1, 0, 0},
			v2:       Version{1, 0, 0},
			expected: false,
		},
		{
			name:     "minor version greater",
			v1:       Version{1, 1, 0},
			v2:       Version{1, 0, 0},
			expected: true,
		},
		{
			name:     "patch version greater",
			v1:       Version{1, 0, 1},
			v2:       Version{1, 0, 0},
			expected: true,
		},
		{
			name:     "longer version greater",
			v1:       Version{1, 0, 1},
			v2:       Version{1, 0},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.v1.Greater(tt.v2)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestVersionCompareVersions(t *testing.T) {
	tests := []struct {
		name     string
		v1       Version
		v2       Version
		expected int
	}{
		{
			name:     "v1 equals v2",
			v1:       Version{1, 0, 0},
			v2:       Version{1, 0, 0},
			expected: versionEqual,
		},
		{
			name:     "v1 less than v2",
			v1:       Version{1, 0, 0},
			v2:       Version{2, 0, 0},
			expected: versionLess,
		},
		{
			name:     "v1 greater than v2",
			v1:       Version{2, 0, 0},
			v2:       Version{1, 0, 0},
			expected: versionGreater,
		},
		{
			name:     "different length equal",
			v1:       Version{1, 0, 0},
			v2:       Version{1, 0},
			expected: versionGreater,
		},
		{
			name:     "different length less",
			v1:       Version{1, 0},
			v2:       Version{1, 0, 1},
			expected: versionLess,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.v1.compareVersions(tt.v2)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestVersionContainsIllegalCharacters(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{
			name:     "valid version string",
			input:    "v1.0.0",
			expected: false,
		},
		{
			name:     "valid uppercase",
			input:    "V2.1.3",
			expected: false,
		},
		{
			name:     "contains dash",
			input:    "v1.0.0-alpha",
			expected: true,
		},
		{
			name:     "contains letters",
			input:    "v1.0.0beta",
			expected: true,
		},
		{
			name:     "contains space",
			input:    "v1.0 .0",
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := versionContainsIllegalCharacters(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestV3_1Constant(t *testing.T) {
	assert.NotNil(t, V3_1)
	assert.Equal(t, Version{3, 1}, V3_1)
}
