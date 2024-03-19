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
