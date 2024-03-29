package rbac

import (
	"fmt"
	"reflect"
	"strings"

	matcherv3 "github.com/envoyproxy/go-control-plane/envoy/type/matcher/v3"
)

type StringMatcher interface {
	//*StringMatcher_Exact
	//*StringMatcher_Prefix
	//*StringMatcher_Suffix
	//*StringMatcher_SafeRegex
	//*StringMatcher_HiddenEnvoyDeprecatedRegex
	//*StringMatcher_Contains
	isStringMatcher()
	Match(string) bool
}

type ExactStringMatcher struct {
	ExactMatch string
}

func (m *ExactStringMatcher) Match(targetValue string) bool {
	return m.ExactMatch == targetValue
}

func (m *ExactStringMatcher) isStringMatcher() {}

type PrefixStringMatcher struct {
	PrefixMatch string
}

func (m *PrefixStringMatcher) Match(targetValue string) bool {
	return strings.HasPrefix(targetValue, m.PrefixMatch)
}

func (m *PrefixStringMatcher) isStringMatcher() {}

func NewStringMatcher(match *matcherv3.StringMatcher) (StringMatcher, error) {
	switch match.MatchPattern.(type) {
	case *matcherv3.StringMatcher_Exact:
		return &ExactStringMatcher{
			ExactMatch: match.MatchPattern.(*matcherv3.StringMatcher_Exact).Exact,
		}, nil
	case *matcherv3.StringMatcher_Prefix:
		return &PrefixStringMatcher{
			PrefixMatch: match.MatchPattern.(*matcherv3.StringMatcher_Prefix).Prefix,
		}, nil
	case *matcherv3.StringMatcher_Suffix:
	case *matcherv3.StringMatcher_SafeRegex:

	default:
		return nil, fmt.Errorf(
			"[NewStringMatcher] not support StringMatcher type found, detail: %v",
			reflect.TypeOf(match.MatchPattern))
	}

	return nil, nil
}
