package rbac

import (
	"fmt"
	"reflect"

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
	Equal(string) bool
}

type ExactStringMatcher struct {
	ExactMatch string
}

func (m *ExactStringMatcher) Equal(targetValue string) bool {
	return m.ExactMatch == targetValue
}

func (m *ExactStringMatcher) isStringMatcher() {}

func NewStringMatcher(match *matcherv3.StringMatcher) (StringMatcher, error) {
	switch match.MatchPattern.(type) {
	case *matcherv3.StringMatcher_Exact:
		return &ExactStringMatcher{
			ExactMatch: match.MatchPattern.(*matcherv3.StringMatcher_Exact).Exact,
		}, nil
	case *matcherv3.StringMatcher_Prefix:
	case *matcherv3.StringMatcher_Suffix:
	case *matcherv3.StringMatcher_SafeRegex:

	default:
		return nil, fmt.Errorf(
			"[NewStringMatcher] not support StringMatcher type found, detail: %v",
			reflect.TypeOf(match.MatchPattern))
	}

	return nil, nil
}
