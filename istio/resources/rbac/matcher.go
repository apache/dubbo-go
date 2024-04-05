package rbac

import (
	"fmt"
	"reflect"
	"regexp"
	"strconv"
	"strings"

	envoyroutev3 "github.com/envoyproxy/go-control-plane/envoy/config/route/v3"
	envoymatcherv3 "github.com/envoyproxy/go-control-plane/envoy/type/matcher/v3"
)

type StringMatcher interface {
	//*StringMatcher_Exact
	//*StringMatcher_Prefix
	//*StringMatcher_Suffix
	//*StringMatcher_SafeRegex
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

type SuffixStringMatcher struct {
	SuffixMatch string
}

func (m *SuffixStringMatcher) isStringMatcher() {}

func (m *SuffixStringMatcher) Match(targetValue string) bool {
	return strings.HasSuffix(targetValue, m.SuffixMatch)
}

type ContainsStringMatcher struct {
	ContainsMatch string
}

func (m *ContainsStringMatcher) isStringMatcher() {}

func (m *ContainsStringMatcher) Match(targetValue string) bool {
	return strings.Contains(targetValue, m.ContainsMatch)
}

type RegexStringMatcher struct {
	RegexMatch *regexp.Regexp
}

func (m *RegexStringMatcher) isStringMatcher() {}

func (m *RegexStringMatcher) Match(targetValue string) bool {
	return m.RegexMatch.MatchString(targetValue)
}

func NewStringMatcher(match *envoymatcherv3.StringMatcher) (StringMatcher, error) {
	switch match.MatchPattern.(type) {
	case *envoymatcherv3.StringMatcher_Exact:
		return &ExactStringMatcher{
			ExactMatch: match.MatchPattern.(*envoymatcherv3.StringMatcher_Exact).Exact,
		}, nil
	case *envoymatcherv3.StringMatcher_Prefix:
		return &PrefixStringMatcher{
			PrefixMatch: match.MatchPattern.(*envoymatcherv3.StringMatcher_Prefix).Prefix,
		}, nil
	case *envoymatcherv3.StringMatcher_Suffix:
		return &SuffixStringMatcher{
			SuffixMatch: match.MatchPattern.(*envoymatcherv3.StringMatcher_Suffix).Suffix,
		}, nil
	case *envoymatcherv3.StringMatcher_SafeRegex:
		return &RegexStringMatcher{
			RegexMatch: regexp.MustCompile(match.MatchPattern.(*envoymatcherv3.StringMatcher_SafeRegex).SafeRegex.Regex),
		}, nil
	case *envoymatcherv3.StringMatcher_Contains:
		return &ContainsStringMatcher{
			ContainsMatch: match.MatchPattern.(*envoymatcherv3.StringMatcher_Contains).Contains,
		}, nil

	default:
		return nil, fmt.Errorf(
			"[NewStringMatcher] not support StringMatcher type found, detail: %v",
			reflect.TypeOf(match.MatchPattern))
	}

	return nil, nil
}

type HeaderMatcher interface {
	//*HeaderMatcher_ExactMatch
	//*HeaderMatcher_RegexMatch
	//*HeaderMatcher_RangeMatch
	//*HeaderMatcher_PresentMatch
	//*HeaderMatcher_PrefixMatch
	//*HeaderMatcher_SuffixMatch
	//*HeaderMatcher_SafeRegexMatch
	isHeaderMatcher()
	Match(string) bool
}

type HeaderMatcherPresentMatch struct {
	PresentMatch bool
}

func (m *HeaderMatcherPresentMatch) Match(targetValue string) bool {
	return m.PresentMatch
}

func (m *HeaderMatcherPresentMatch) isHeaderMatcher() {}

type HeaderMatcherRangeMatch struct {
	Start int64 // inclusive
	End   int64 // exclusive
}

func (m *HeaderMatcherRangeMatch) isHeaderMatcher() {}

func (m *HeaderMatcherRangeMatch) Match(targetValue string) bool {
	if intValue, err := strconv.ParseInt(targetValue, 10, 64); err != nil {
		// return not match if target value is not a integer
		return false
	} else {
		return intValue >= m.Start && intValue < m.End
	}
}

func (m *ExactStringMatcher) isHeaderMatcher()  {}
func (m *PrefixStringMatcher) isHeaderMatcher() {}
func (m *SuffixStringMatcher) isHeaderMatcher() {}
func (m *RegexStringMatcher) isHeaderMatcher()  {}

func NewHeaderMatcher(header *envoyroutev3.HeaderMatcher) (HeaderMatcher, error) {
	switch header.HeaderMatchSpecifier.(type) {
	case *envoyroutev3.HeaderMatcher_ExactMatch:
		return &HeaderMatcherPresentMatch{}, nil
	case *envoyroutev3.HeaderMatcher_PrefixMatch:
		return &PrefixStringMatcher{
			PrefixMatch: header.HeaderMatchSpecifier.(*envoyroutev3.HeaderMatcher_PrefixMatch).PrefixMatch,
		}, nil
	case *envoyroutev3.HeaderMatcher_SuffixMatch:
		return &SuffixStringMatcher{
			SuffixMatch: header.HeaderMatchSpecifier.(*envoyroutev3.HeaderMatcher_SuffixMatch).SuffixMatch,
		}, nil
	case *envoyroutev3.HeaderMatcher_SafeRegexMatch:
		re := header.HeaderMatchSpecifier.(*envoyroutev3.HeaderMatcher_SafeRegexMatch).SafeRegexMatch
		if _, ok := re.EngineType.(*envoymatcherv3.RegexMatcher_GoogleRe2); !ok {
			return nil, fmt.Errorf("[NewHeaderMatcher] failed to build regex, unsupported engine type: %v", reflect.TypeOf(re.EngineType))
		}
		if rePattern, err := regexp.Compile(re.Regex); err != nil {
			return nil, fmt.Errorf("[NewHeaderMatcher] failed to build regex, error: %v", err)
		} else {
			return &RegexStringMatcher{
				RegexMatch: rePattern,
			}, nil
		}
	case *envoyroutev3.HeaderMatcher_PresentMatch:
		return &HeaderMatcherPresentMatch{
			PresentMatch: header.HeaderMatchSpecifier.(*envoyroutev3.HeaderMatcher_PresentMatch).PresentMatch,
		}, nil
	case *envoyroutev3.HeaderMatcher_RangeMatch:
		return &HeaderMatcherRangeMatch{
			Start: header.HeaderMatchSpecifier.(*envoyroutev3.HeaderMatcher_RangeMatch).RangeMatch.Start,
			End:   header.HeaderMatchSpecifier.(*envoyroutev3.HeaderMatcher_RangeMatch).RangeMatch.End,
		}, nil
	default:
		return nil, fmt.Errorf(
			"[NewHeaderMatcher] not support HeaderMatchSpecifier type found, error: %v",
			reflect.TypeOf(header.HeaderMatchSpecifier))
	}

	return nil, nil
}

type UrlPathMatcher interface {
	//	*PathMatcher_Path (supported)
	isUrlPathMatcher()
	Match(string) bool
}

type DefaultUrlPathMatcher struct {
	Matcher StringMatcher
}

func (m *DefaultUrlPathMatcher) isUrlPathMatcher() {}

func (m *DefaultUrlPathMatcher) Match(targetValue string) bool {
	return m.Matcher.Match(targetValue)
}

func NewUrlPathMatcher(urlPath *envoymatcherv3.PathMatcher) (UrlPathMatcher, error) {
	switch urlPath.Rule.(type) {
	case *envoymatcherv3.PathMatcher_Path:
		m, err := NewStringMatcher(urlPath.Rule.(*envoymatcherv3.PathMatcher_Path).Path)
		return &DefaultUrlPathMatcher{
			Matcher: m,
		}, err
	default:
		return nil, fmt.Errorf(
			"[NewUrlPathMatcher] not support PathMatcher type found, error: %v",
			reflect.TypeOf(urlPath.Rule))
	}
}
