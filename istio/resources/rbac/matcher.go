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

type StringMatcher struct {
	// Types that are assignable to MatchPattern:
	//	*StringMatcher_Exact
	//	*StringMatcher_Prefix
	//	*StringMatcher_Suffix
	//	*StringMatcher_SafeRegex
	//	*StringMatcher_Contains
	MatchPattern StringMatcherMatchPattern
	// If true, indicates the exact/prefix/suffix/contains matching should be case insensitive. This
	// has no effect for the safe_regex match.
	// For example, the matcher ``data`` will match both input string ``Data`` and ``data`` if set to true.
	IgnoreCase bool
}

func (s *StringMatcher) Match(targetValue string) bool {
	return s.MatchPattern.Match(s.IgnoreCase, targetValue)
}

type StringMatcherMatchPattern interface {
	//*StringMatcher_Exact
	//*StringMatcher_Prefix
	//*StringMatcher_Suffix
	//*StringMatcher_SafeRegex
	//*StringMatcher_Contains
	isStringMatcher()
	Match(bool, string) bool
}

type ExactStringMatcher struct {
	ExactMatch string
}

func (m *ExactStringMatcher) Match(IgnoreCase bool, targetValue string) bool {
	return m.ExactMatch == targetValue
}

func (m *ExactStringMatcher) isStringMatcher() {}

type PrefixStringMatcher struct {
	PrefixMatch string
}

func (m *PrefixStringMatcher) Match(IgnoreCase bool, targetValue string) bool {
	return strings.HasPrefix(targetValue, m.PrefixMatch)
}

func (m *PrefixStringMatcher) isStringMatcher() {}

type SuffixStringMatcher struct {
	SuffixMatch string
}

func (m *SuffixStringMatcher) isStringMatcher() {}

func (m *SuffixStringMatcher) Match(IgnoreCase bool, targetValue string) bool {
	return strings.HasSuffix(targetValue, m.SuffixMatch)
}

type ContainsStringMatcher struct {
	ContainsMatch string
}

func (m *ContainsStringMatcher) isStringMatcher() {}

func (m *ContainsStringMatcher) Match(IgnoreCase bool, targetValue string) bool {
	return strings.Contains(targetValue, m.ContainsMatch)
}

type RegexStringMatcher struct {
	RegexMatch *regexp.Regexp
}

func (m *RegexStringMatcher) isStringMatcher() {}

func (m *RegexStringMatcher) Match(IgnoreCase bool, targetValue string) bool {
	return m.RegexMatch.MatchString(targetValue)
}

func NewStringMatcherMatchPattern(mather *envoymatcherv3.StringMatcher) (*StringMatcher, error) {
	stringMatcher := &StringMatcher{
		IgnoreCase: mather.IgnoreCase,
	}
	switch mather.MatchPattern.(type) {
	case *envoymatcherv3.StringMatcher_Exact:
		stringMatcher.MatchPattern = &ExactStringMatcher{
			ExactMatch: mather.MatchPattern.(*envoymatcherv3.StringMatcher_Exact).Exact,
		}
	case *envoymatcherv3.StringMatcher_Prefix:
		stringMatcher.MatchPattern = &PrefixStringMatcher{
			PrefixMatch: mather.MatchPattern.(*envoymatcherv3.StringMatcher_Prefix).Prefix,
		}
	case *envoymatcherv3.StringMatcher_Suffix:
		stringMatcher.MatchPattern = &SuffixStringMatcher{
			SuffixMatch: mather.MatchPattern.(*envoymatcherv3.StringMatcher_Suffix).Suffix,
		}
	case *envoymatcherv3.StringMatcher_SafeRegex:
		stringMatcher.MatchPattern = &RegexStringMatcher{
			RegexMatch: regexp.MustCompile(mather.MatchPattern.(*envoymatcherv3.StringMatcher_SafeRegex).SafeRegex.Regex),
		}
	case *envoymatcherv3.StringMatcher_Contains:
		stringMatcher.MatchPattern = &ContainsStringMatcher{
			ContainsMatch: mather.MatchPattern.(*envoymatcherv3.StringMatcher_Contains).Contains,
		}
	default:
		return nil, fmt.Errorf(
			"[NewStringMatcher] not support StringMatcher type found, detail: %v",
			reflect.TypeOf(mather.MatchPattern))
	}

	return stringMatcher, nil
}

func NewStringMatcher(mather *envoymatcherv3.StringMatcher) (*StringMatcher, error) {
	stringMatcher := &StringMatcher{
		IgnoreCase: mather.IgnoreCase,
	}
	switch mather.MatchPattern.(type) {
	case *envoymatcherv3.StringMatcher_Exact:
		stringMatcher.MatchPattern = &ExactStringMatcher{
			ExactMatch: mather.MatchPattern.(*envoymatcherv3.StringMatcher_Exact).Exact,
		}
	case *envoymatcherv3.StringMatcher_Prefix:
		stringMatcher.MatchPattern = &PrefixStringMatcher{
			PrefixMatch: mather.MatchPattern.(*envoymatcherv3.StringMatcher_Prefix).Prefix,
		}
	case *envoymatcherv3.StringMatcher_Suffix:
		stringMatcher.MatchPattern = &SuffixStringMatcher{
			SuffixMatch: mather.MatchPattern.(*envoymatcherv3.StringMatcher_Suffix).Suffix,
		}
	case *envoymatcherv3.StringMatcher_SafeRegex:
		stringMatcher.MatchPattern = &RegexStringMatcher{
			RegexMatch: regexp.MustCompile(mather.MatchPattern.(*envoymatcherv3.StringMatcher_SafeRegex).SafeRegex.Regex),
		}
	case *envoymatcherv3.StringMatcher_Contains:
		stringMatcher.MatchPattern = &ContainsStringMatcher{
			ContainsMatch: mather.MatchPattern.(*envoymatcherv3.StringMatcher_Contains).Contains,
		}
	default:
		return nil, fmt.Errorf(
			"[NewStringMatcher] not support StringMatcher type found, detail: %v",
			reflect.TypeOf(mather.MatchPattern))
	}

	return stringMatcher, nil
}

type HeaderMatcher struct {
	// Specifies the name of the header in the request.
	Name string
	// Specifies how the header match will be performed to route the request.
	//
	// Types that are assignable to HeaderMatchSpecifier:
	//	*HeaderMatcher_ExactMatch
	//	*HeaderMatcher_SafeRegexMatch
	//	*HeaderMatcher_RangeMatch
	//	*HeaderMatcher_PresentMatch
	//	*HeaderMatcher_PrefixMatch
	//	*HeaderMatcher_SuffixMatch
	//	*HeaderMatcher_ContainsMatch
	//	*HeaderMatcher_StringMatch
	HeaderMatchSpecifier HeaderMatcherSpecifier
	// If specified, the match result will be inverted before checking. Defaults to false.
	//
	// Examples:
	//
	// * The regex ``\d{3}`` does not match the value ``1234``, so it will match when inverted.
	// * The range [-10,0) will match the value -1, so it will not match when inverted.
	InvertMatch bool
	// If specified, for any header match rule, if the header match rule specified header
	// does not exist, this header value will be treated as empty. Defaults to false.
	//
	// Examples:
	//
	TreatMissingHeaderAsEmpty bool
}

func (h *HeaderMatcher) Match(headers map[string]string) bool {
	targetValue, found := getHeader(h.Name, headers)
	// HeaderMatcherPresent is a little special
	if matcher, ok := h.HeaderMatchSpecifier.(*HeaderMatcherPresent); ok {
		isMatch := found && matcher.PresentMatch
		return h.InvertMatch != isMatch
	}
	// return false when targetValue is not found, except matcher is `HeaderMatcherPresent`
	if !found {
		return false
	} else {
		isMatch := h.HeaderMatchSpecifier.Match(targetValue)
		// permission.InvertMatch xor isMatch
		return h.InvertMatch != isMatch
	}
}

type HeaderMatcherSpecifier interface {
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

type HeaderMatcherPresent struct {
	PresentMatch bool
}

func (m *HeaderMatcherPresent) Match(targetValue string) bool {
	return m.PresentMatch
}

func (m *HeaderMatcherPresent) isHeaderMatcher() {}

type HeaderMatcherRange struct {
	Start int64 // inclusive
	End   int64 // exclusive
}

func (m *HeaderMatcherRange) isHeaderMatcher() {}

func (m *HeaderMatcherRange) Match(targetValue string) bool {
	if intValue, err := strconv.ParseInt(targetValue, 10, 64); err != nil {
		// return not match if target value is not a integer
		return false
	} else {
		return intValue >= m.Start && intValue < m.End
	}
}

type HeaderMatcherString struct {
	StringMatch *StringMatcher
}

func (m *HeaderMatcherString) isHeaderMatcher() {}

func (m *HeaderMatcherString) Match(targetValue string) bool {
	return m.StringMatch.Match(targetValue)
}

type HeaderMatcherExact struct {
	ExactMatch string
}

func (m *HeaderMatcherExact) Match(targetValue string) bool {
	return m.ExactMatch == targetValue
}

func (m *HeaderMatcherExact) isHeaderMatcher() {}

type HeaderMatcherPrefix struct {
	PrefixMatch string
}

func (m *HeaderMatcherPrefix) Match(targetValue string) bool {
	return strings.HasPrefix(targetValue, m.PrefixMatch)
}

func (m *HeaderMatcherPrefix) isHeaderMatcher() {}

type HeaderMatcherSuffix struct {
	SuffixMatch string
}

func (m *HeaderMatcherSuffix) isHeaderMatcher() {}

func (m *HeaderMatcherSuffix) Match(targetValue string) bool {
	return strings.HasSuffix(targetValue, m.SuffixMatch)
}

type HeaderMatcherContains struct {
	ContainsMatch string
}

func (m *HeaderMatcherContains) isHeaderMatcher() {}

func (m *HeaderMatcherContains) Match(targetValue string) bool {
	return strings.Contains(targetValue, m.ContainsMatch)
}

type HeaderMatcherRegex struct {
	RegexMatch *regexp.Regexp
}

func (m *HeaderMatcherRegex) isHeaderMatcher() {}

func (m *HeaderMatcherRegex) Match(targetValue string) bool {
	return m.RegexMatch.MatchString(targetValue)
}

func NewHeaderMatcher(header *envoyroutev3.HeaderMatcher) (*HeaderMatcher, error) {
	headerMatcher := &HeaderMatcher{
		Name:                      header.Name,
		InvertMatch:               header.InvertMatch,
		TreatMissingHeaderAsEmpty: header.TreatMissingHeaderAsEmpty,
	}
	switch header.HeaderMatchSpecifier.(type) {
	case *envoyroutev3.HeaderMatcher_ExactMatch:
		headerMatcher.HeaderMatchSpecifier = &HeaderMatcherExact{
			ExactMatch: header.HeaderMatchSpecifier.(*envoyroutev3.HeaderMatcher_ExactMatch).ExactMatch,
		}
	case *envoyroutev3.HeaderMatcher_PrefixMatch:
		headerMatcher.HeaderMatchSpecifier = &HeaderMatcherPrefix{
			PrefixMatch: header.HeaderMatchSpecifier.(*envoyroutev3.HeaderMatcher_PrefixMatch).PrefixMatch,
		}
	case *envoyroutev3.HeaderMatcher_SuffixMatch:
		headerMatcher.HeaderMatchSpecifier = &HeaderMatcherSuffix{
			SuffixMatch: header.HeaderMatchSpecifier.(*envoyroutev3.HeaderMatcher_SuffixMatch).SuffixMatch,
		}
	case *envoyroutev3.HeaderMatcher_SafeRegexMatch:
		re := header.HeaderMatchSpecifier.(*envoyroutev3.HeaderMatcher_SafeRegexMatch).SafeRegexMatch
		if _, ok := re.EngineType.(*envoymatcherv3.RegexMatcher_GoogleRe2); !ok {
			return nil, fmt.Errorf("[NewHeaderMatcher] failed to build regex, unsupported engine type: %v", reflect.TypeOf(re.EngineType))
		}
		if rePattern, err := regexp.Compile(re.Regex); err != nil {
			return nil, fmt.Errorf("[NewHeaderMatcher] failed to build regex, error: %v", err)
		} else {
			headerMatcher.HeaderMatchSpecifier = &HeaderMatcherRegex{
				RegexMatch: rePattern,
			}
		}
	case *envoyroutev3.HeaderMatcher_PresentMatch:
		headerMatcher.HeaderMatchSpecifier = &HeaderMatcherPresent{
			PresentMatch: header.HeaderMatchSpecifier.(*envoyroutev3.HeaderMatcher_PresentMatch).PresentMatch,
		}
	case *envoyroutev3.HeaderMatcher_RangeMatch:
		headerMatcher.HeaderMatchSpecifier = &HeaderMatcherRange{
			Start: header.HeaderMatchSpecifier.(*envoyroutev3.HeaderMatcher_RangeMatch).RangeMatch.Start,
			End:   header.HeaderMatchSpecifier.(*envoyroutev3.HeaderMatcher_RangeMatch).RangeMatch.End,
		}
	case *envoyroutev3.HeaderMatcher_StringMatch:
		stringMatcher, err := NewStringMatcher(header.HeaderMatchSpecifier.(*envoyroutev3.HeaderMatcher_StringMatch).StringMatch)
		if err != nil {
			return nil, err
		}
		headerMatcher.HeaderMatchSpecifier = &HeaderMatcherString{
			StringMatch: stringMatcher,
		}
	default:
		return nil, fmt.Errorf(
			"[NewHeaderMatcher] not support HeaderMatchSpecifier type found, error: %v",
			reflect.TypeOf(header.HeaderMatchSpecifier))
	}

	return headerMatcher, nil
}

type UrlPathMatcher interface {
	//	*PathMatcher_Path (supported)
	isUrlPathMatcher()
	Match(string) bool
}

type DefaultUrlPathMatcher struct {
	Matcher *StringMatcher
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

type ValueMatcher struct {
	// Specifies how to match a value.
	//
	// Types that are assignable to MatchPattern:
	//	*ValueMatcher_NullMatch_
	//	*ValueMatcher_DoubleMatch
	//	*ValueMatcher_StringMatch
	//	*ValueMatcher_BoolMatch
	//	*ValueMatcher_PresentMatch
	//	*ValueMatcher_ListMatch
	MatchPattern ValueMatcherMatchPattern
}

func (v *ValueMatcher) Match(targetValue string) bool {

	return v.MatchPattern.Match(targetValue)
}

type ValueMatcherMatchPattern interface {
	// Specifies how to match a value.
	//
	// Types that are assignable to MatchPattern:
	//	*ValueMatcher_NullMatch_
	//	*ValueMatcher_DoubleMatch
	//	*ValueMatcher_StringMatch
	//	*ValueMatcher_BoolMatch
	//	*ValueMatcher_PresentMatch
	//	*ValueMatcher_ListMatch
	isValueMatcher()
	Match(string) bool
}

type NullValueMatcher struct {
	// If specified, a match occurs if and only if the target value is a NullValue.
}

func (n *NullValueMatcher) isValueMatcher() {}
func (n *NullValueMatcher) Match(targetValue string) bool {
	if targetValue != "" {
		return false
	}
	return true
}

type DoubleRangeValueMatcher struct {
	// start of the range (inclusive)
	Start float64
	// end of the range (exclusive)
	End float64
}

func (d *DoubleRangeValueMatcher) isValueMatcher() {}
func (d *DoubleRangeValueMatcher) Match(targetValue string) bool {
	v, err := strconv.ParseFloat(targetValue, 32)
	if err != nil {
		return false
	}
	if v < d.Start || v >= d.End {
		return false
	}
	return true
}

type DoubleExactValueMatcher struct {
	// If specified, the input double value must be equal to the value specified here.
	Exact float64
}

func (d *DoubleExactValueMatcher) isValueMatcher() {}
func (d *DoubleExactValueMatcher) Match(targetValue string) bool {
	v, err := strconv.ParseFloat(targetValue, 32)
	if err != nil {
		return false
	}
	return v == d.Exact
}

type StringValueMatcher struct {
	StringMatch *StringMatcher
}

func (s *StringValueMatcher) isValueMatcher() {}
func (s *StringValueMatcher) Match(targetValue string) bool {
	return s.StringMatch.Match(targetValue)
}

type BoolValueMatcher struct {
	// If specified, a match occurs if and only if the target value is a bool value and is equal
	// to this field.
	BoolMatch bool
}

func (b *BoolValueMatcher) isValueMatcher() {}
func (b *BoolValueMatcher) Match(targetValue string) bool {
	v, err := strconv.ParseBool(targetValue)
	if err != nil {
		return false
	}
	return v == b.BoolMatch
}

type PresentValueMatcher struct {
	// If specified, value match will be performed based on whether the path is referring to a
	// valid primitive value in the metadata. If the path is referring to a non-primitive value,
	// the result is always not matched.
	PresentMatch bool
}

func (p *PresentValueMatcher) isValueMatcher() {}
func (p *PresentValueMatcher) Match(targetValue string) bool {
	return p.PresentMatch
}

type ListValueOneOfMatcher struct {
	// If specified, a match occurs if and only if the target value is a list and it contains at
	// least one element that matches the specified match pattern.
	OneOf *ValueMatcher
}

func (l *ListValueOneOfMatcher) isValueMatcher() {}
func (l *ListValueOneOfMatcher) Match(targetValue string) bool {
	values := strings.Split(targetValue, ",")
	for _, v := range values {
		if l.OneOf.Match(v) {
			return true
		}
	}
	return false
}

func NewValueMatcher(value *envoymatcherv3.ValueMatcher) (*ValueMatcher, error) {
	valueMatcher := &ValueMatcher{}
	switch value.MatchPattern.(type) {
	case *envoymatcherv3.ValueMatcher_NullMatch_:
		valueMatcher.MatchPattern = &NullValueMatcher{}
	case *envoymatcherv3.ValueMatcher_DoubleMatch:
		matcher := value.MatchPattern.(*envoymatcherv3.ValueMatcher_DoubleMatch)
		switch matcher.DoubleMatch.MatchPattern.(type) {
		case *envoymatcherv3.DoubleMatcher_Range:
			valueMatcher.MatchPattern = &DoubleRangeValueMatcher{
				Start: matcher.DoubleMatch.MatchPattern.(*envoymatcherv3.DoubleMatcher_Range).Range.Start,
				End:   matcher.DoubleMatch.MatchPattern.(*envoymatcherv3.DoubleMatcher_Range).Range.End,
			}
		case *envoymatcherv3.DoubleMatcher_Exact:
			valueMatcher.MatchPattern = &DoubleExactValueMatcher{
				Exact: matcher.DoubleMatch.MatchPattern.(*envoymatcherv3.DoubleMatcher_Exact).Exact,
			}
		default:
			return nil, fmt.Errorf(
				"[NewValueMatcher] not support DoubleMatcher type found, error: %v",
				reflect.TypeOf(matcher.DoubleMatch))
		}
	case *envoymatcherv3.ValueMatcher_StringMatch:
		stringValueMatcher, err := NewStringMatcher(value.MatchPattern.(*envoymatcherv3.ValueMatcher_StringMatch).StringMatch)
		if err != nil {
			return nil, err
		}
		valueMatcher.MatchPattern = &StringValueMatcher{StringMatch: stringValueMatcher}
	case *envoymatcherv3.ValueMatcher_BoolMatch:
		valueMatcher.MatchPattern = &BoolValueMatcher{}
	case *envoymatcherv3.ValueMatcher_PresentMatch:
		valueMatcher.MatchPattern = &PresentValueMatcher{}
	case *envoymatcherv3.ValueMatcher_ListMatch:
		matcher := value.MatchPattern.(*envoymatcherv3.ValueMatcher_ListMatch)
		switch matcher.ListMatch.MatchPattern.(type) {
		case *envoymatcherv3.ListMatcher_OneOf:
			oneOfValueMatcher, err := NewValueMatcher(matcher.ListMatch.MatchPattern.(*envoymatcherv3.ListMatcher_OneOf).OneOf)
			if err != nil {
				return nil, err
			}
			valueMatcher.MatchPattern = &ListValueOneOfMatcher{
				OneOf: oneOfValueMatcher,
			}
		default:
			return nil, fmt.Errorf(
				"[NewValueMatcher] not support ListMatcher type found, error: %v",
				reflect.TypeOf(matcher.ListMatch))
		}
	default:
		return nil, fmt.Errorf(
			"[NewValueMatcher] not support ValueMatcher type found, error: %v",
			reflect.TypeOf(value.MatchPattern))
	}

	return valueMatcher, nil
}
