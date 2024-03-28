package rbac

import (
	"fmt"
	"reflect"

	routev3 "github.com/envoyproxy/go-control-plane/envoy/config/route/v3"
)

type HeaderMatcher interface {
	//*HeaderMatcher_ExactMatch
	//*HeaderMatcher_RegexMatch
	//*HeaderMatcher_RangeMatch
	//*HeaderMatcher_PresentMatch
	//*HeaderMatcher_PrefixMatch
	//*HeaderMatcher_SuffixMatch
	//*HeaderMatcher_SafeRegexMatch
	isHeaderMatcher()
	Equal(string) bool
}

type HeaderMatcherPresentMatch struct {
	PresentMatch bool
}

func (m *HeaderMatcherPresentMatch) Equal(targetValue string) bool {
	return m.PresentMatch
}

func (m *HeaderMatcherPresentMatch) isHeaderMatcher() {}

func NewHeaderMatcher(header *routev3.HeaderMatcher) (HeaderMatcher, error) {
	switch header.HeaderMatchSpecifier.(type) {
	case *routev3.HeaderMatcher_ExactMatch:
		return &HeaderMatcherPresentMatch{}, nil
	case *routev3.HeaderMatcher_PrefixMatch:
	case *routev3.HeaderMatcher_SuffixMatch:
	case *routev3.HeaderMatcher_SafeRegexMatch:
	case *routev3.HeaderMatcher_PresentMatch:
	case *routev3.HeaderMatcher_RangeMatch:
	default:
		return nil, fmt.Errorf(
			"[NewHeaderMatcher] not support HeaderMatchSpecifier type found, detail: %v",
			reflect.TypeOf(header.HeaderMatchSpecifier))
	}

	return nil, nil
}
