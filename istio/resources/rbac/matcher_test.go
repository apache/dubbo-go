package rbac

import "testing"

func TestExactStringMatcher_Match(t *testing.T) {
	testCases := []struct {
		name         string
		matcher      *ExactStringMatcher
		targetValue  string
		expectedBool bool
	}{
		{
			name: "Match found",
			matcher: &ExactStringMatcher{
				ExactMatch: "test",
			},
			targetValue:  "test",
			expectedBool: true,
		},
		{
			name: "No match found",
			matcher: &ExactStringMatcher{
				ExactMatch: "apple",
			},
			targetValue:  "banana",
			expectedBool: false,
		},
		{
			name: "Empty exact match",
			matcher: &ExactStringMatcher{
				ExactMatch: "",
			},
			targetValue:  "empty",
			expectedBool: false,
		},
		{
			name: "Target value empty",
			matcher: &ExactStringMatcher{
				ExactMatch: "notEmpty",
			},
			targetValue:  "",
			expectedBool: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			actualBool := tc.matcher.Match(false, tc.targetValue)
			if actualBool != tc.expectedBool {
				t.Errorf("Expected %t for targetValue '%s' and ExactMatch '%s', but got %t", tc.expectedBool, tc.targetValue, tc.matcher.ExactMatch, actualBool)
			}
		})
	}
}

func TestPrefixStringMatcher_Match(t *testing.T) {
	testCases := []struct {
		name        string
		prefixField PrefixStringMatcher
		target      string
		expected    bool
	}{
		{
			name:        "Empty prefix and target",
			prefixField: PrefixStringMatcher{PrefixMatch: ""},
			target:      "",
			expected:    true,
		},
		{
			name:        "Empty prefix",
			prefixField: PrefixStringMatcher{PrefixMatch: ""},
			target:      "abc",
			expected:    true,
		},
		{
			name:        "Exact match",
			prefixField: PrefixStringMatcher{PrefixMatch: "pre"},
			target:      "prefix",
			expected:    true,
		},
		{
			name:        "Non-matching prefix",
			prefixField: PrefixStringMatcher{PrefixMatch: "pre"},
			target:      "postfix",
			expected:    false,
		},
		{
			name:        "Valid prefix",
			prefixField: PrefixStringMatcher{PrefixMatch: "hello"},
			target:      "hello world",
			expected:    true,
		},
		{
			name:        "Invalid prefix",
			prefixField: PrefixStringMatcher{PrefixMatch: "hello"},
			target:      "world hello",
			expected:    false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := tc.prefixField.Match(false, tc.target)

			if result != tc.expected {
				t.Errorf("PrefixStringMatcher.Match(%v, %s) = %t, expected %t", tc.prefixField, tc.target, result, tc.expected)
			}
		})
	}
}

func TestSuffixStringMatcher_Match(t *testing.T) {
	testCases := []struct {
		name         string
		suffix       string
		targetValue  string
		expectedBool bool
	}{
		{
			name:         "Target value with matching suffix",
			suffix:       "suffix",
			targetValue:  "example suffix",
			expectedBool: true,
		},
		{
			name:         "Target value without matching suffix",
			suffix:       "suffix",
			targetValue:  "example",
			expectedBool: false,
		},
		{
			name:         "Empty target value",
			suffix:       "suffix",
			targetValue:  "",
			expectedBool: false,
		},
		{
			name:         "Target value equals suffix",
			suffix:       "suffix",
			targetValue:  "suffix",
			expectedBool: true,
		},
		{
			name:         "Target value longer than suffix, no match",
			suffix:       "abc",
			targetValue:  "xyzabcde",
			expectedBool: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			matcher := &SuffixStringMatcher{SuffixMatch: tc.suffix}
			actualBool := matcher.Match(false, tc.targetValue)

			if actualBool != tc.expectedBool {
				t.Errorf("Match() failed for case '%s'. Expected: %v, but got: %v", tc.name, tc.expectedBool, actualBool)
			}
		})
	}
}

func TestContainsStringMatcher_Match(t *testing.T) {
	testCases := []struct {
		name  string
		m     *ContainsStringMatcher
		value string
		want  bool
	}{
		{
			name:  "Match",
			m:     &ContainsStringMatcher{ContainsMatch: "abc"},
			value: "abcdefg",
			want:  true,
		},
		{
			name:  "No Match",
			m:     &ContainsStringMatcher{ContainsMatch: "xyz"},
			value: "abcdefg",
			want:  false,
		},
		{
			name:  "Empty String",
			m:     &ContainsStringMatcher{ContainsMatch: "abc"},
			value: "",
			want:  false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := tc.m.Match(false, tc.value)
			if got != tc.want {
				t.Errorf("Match() = %v, want %v", got, tc.want)
			}
		})
	}
}
