package router

import (
	"encoding/base64"
	"github.com/apache/dubbo-go/cluster"
	"github.com/apache/dubbo-go/common/constant"
	"github.com/apache/dubbo-go/common/utils"
	"regexp"
	"strings"

	"github.com/apache/dubbo-go/common"
	"github.com/apache/dubbo-go/protocol"
	perrors "github.com/pkg/errors"
)

const (
	RoutePattern = `([&!=,]*)\\s*([^&!=,\\s]+)`
)

var itemExists = struct{}{}

type ConditionRouter struct {
	Pattern       string
	Url           common.URL
	Priority      int64
	Force         bool
	WhenCondition map[string]MatchPair
	ThenCondition map[string]MatchPair
}

func (c *ConditionRouter) Route(invokers []protocol.Invoker, url common.URL, invocation protocol.Invocation) ([]protocol.Invoker, error) {
	if len(invokers) == 0 {
		return invokers, nil
	}
	if !c.MatchWhen(url, invocation) {
		return invokers, nil
	}
	var result []protocol.Invoker
	if len(c.ThenCondition) == 0 {
		return result, nil
	}
	for _, invoker := range invokers {

		if c.MatchThen(invoker.GetUrl(), url) {
			result = append(result, invoker)
		}
	}
	if len(result) > 0 {
		return result, nil
	} else if c.Force {
		//todo 日志
		return result, nil
	}
	return invokers, nil
}

func (c ConditionRouter) CompareTo(r cluster.Router) int {
	var result int
	router, ok := r.(*ConditionRouter)
	if r == nil || !ok {
		return 1
	}
	if c.Priority == router.Priority {
		result = strings.Compare(c.Url.String(), router.Url.String())
	} else {
		if c.Priority > router.Priority {
			result = 1
		} else {
			result = -1
		}
	}
	return result
}

func newConditionRouter(url common.URL) (*ConditionRouter, error) {
	var whenRule string
	var thenRule string

	ruleDec, err := base64.URLEncoding.DecodeString(url.GetParam("rule", ""))
	rule := string(ruleDec)
	if err != nil || rule == "" {
		return nil, perrors.Errorf("Illegal route rule!")
	}
	rule = strings.Replace(rule, "consumer.", "", -1)
	rule = strings.Replace(rule, "provider.", "", -1)
	i := strings.Index(rule, "=>")
	whenRule = strings.Trim(If(i < 0, "", rule[0:i]).(string), " ")
	thenRule = strings.Trim(If(i < 0, rule, rule[i+2:]).(string), " ")

	w, err := parseRule(whenRule)
	if err != nil {
		return nil, perrors.Errorf("%s", "")
	}

	t, err := parseRule(thenRule)
	if err != nil {
		return nil, perrors.Errorf("%s", "")
	}
	when := If(whenRule == "" || "true" == whenRule, make(map[string]MatchPair), w).(map[string]MatchPair)
	then := If(thenRule == "" || "false" == thenRule, make(map[string]MatchPair), t).(map[string]MatchPair)

	return &ConditionRouter{
		RoutePattern,
		url,
		url.GetParamInt("priority", 0),
		url.GetParamBool("force", false),
		when,
		then,
	}, nil
}

func parseRule(rule string) (map[string]MatchPair, error) {

	condition := make(map[string]MatchPair)
	if rule == "" {
		return condition, nil
	}
	var pair MatchPair
	values := utils.NewSet()

	reg := regexp.MustCompile(`([&!=,]*)\s*([^&!=,\s]+)`)
	var startIndex = 0
	if indexTuple := reg.FindIndex([]byte(rule)); len(indexTuple) > 0 {
		startIndex = indexTuple[0]
	}
	matches := reg.FindAllSubmatch([]byte(rule), -1)
	for _, groups := range matches {
		separator := string(groups[1])
		content := string(groups[2])

		switch separator {
		case "":
			pair = MatchPair{
				Matches:    utils.NewSet(),
				Mismatches: utils.NewSet(),
			}
			condition[content] = pair
		case "&":
			if r, ok := condition[content]; ok {
				pair = r
			} else {
				pair = MatchPair{
					Matches:    utils.NewSet(),
					Mismatches: utils.NewSet(),
				}
				condition[content] = pair
			}
		case "=":
			if &pair == nil {
				return nil, perrors.Errorf("Illegal route rule \"%s\", The error char '%s' at index %d before \"%d\".", rule, separator, startIndex, startIndex)
			}
			values = pair.Matches
			values.Add(content)
		case "!=":
			if &pair == nil {
				return nil, perrors.Errorf("Illegal route rule \"%s\", The error char '%s' at index %d before \"%d\".", rule, separator, startIndex, startIndex)
			}
			values = pair.Mismatches
			values.Add(content)
		case ",":
			if values.Empty() {
				return nil, perrors.Errorf("Illegal route rule \"%s\", The error char '%s' at index %d before \"%d\".", rule, separator, startIndex, startIndex)
			}
			values.Add(content)
		default:
			return nil, perrors.Errorf("Illegal route rule \"%s\", The error char '%s' at index %d before \"%d\".", rule, separator, startIndex, startIndex)

		}
	}

	return condition, nil
	//var pair MatchPair

}

func (c *ConditionRouter) MatchWhen(url common.URL, invocation protocol.Invocation) bool {

	return len(c.WhenCondition) == 0 || MatchCondition(c.WhenCondition, &url, nil, invocation)
}
func (c *ConditionRouter) MatchThen(url common.URL, param common.URL) bool {

	return len(c.ThenCondition) > 0 && MatchCondition(c.ThenCondition, &url, &param, nil)
}

func MatchCondition(pairs map[string]MatchPair, url *common.URL, param *common.URL, invocation protocol.Invocation) bool {
	sample := url.ToMap()
	if len(sample) == 0 {
		return true
	}
	result := false
	for key, matchPair := range pairs {
		var sampleValue string

		if invocation != nil && ((constant.METHOD_KEY == key) || (constant.METHOD_KEYS == key)) {
			sampleValue = invocation.MethodName()
		} else {
			sampleValue = sample[key]
			if sampleValue == "" {
				sampleValue = sample[constant.PREFIX_DEFAULT_KEY+key]
			}
		}
		if sampleValue != "" {
			if !matchPair.isMatch(sampleValue, param) {
				return false
			} else {
				result = true
			}
		} else {
			if !(matchPair.Matches.Empty()) {
				return false
			} else {
				result = true
			}
		}

	}
	return result
}

func If(b bool, t, f interface{}) interface{} {
	if b {
		return t
	}
	return f
}

type MatchPair struct {
	Matches    *utils.HashSet
	Mismatches *utils.HashSet
}

func (pair MatchPair) isMatch(value string, param *common.URL) bool {

	if !pair.Matches.Empty() && pair.Mismatches.Empty() {

		for match := range pair.Matches.Items {
			if isMatchGlobPattern(match.(string), value, param) {
				return true
			}
		}
		return false
	}
	if !pair.Mismatches.Empty() && pair.Matches.Empty() {

		for mismatch := range pair.Mismatches.Items {
			if isMatchGlobPattern(mismatch.(string), value, param) {
				return false
			}
		}
		return true
	}
	if !pair.Mismatches.Empty() && !pair.Matches.Empty() {
		for mismatch := range pair.Mismatches.Items {
			if isMatchGlobPattern(mismatch.(string), value, param) {
				return false
			}
		}
		for match := range pair.Matches.Items {
			if isMatchGlobPattern(match.(string), value, param) {
				return true
			}
		}
		return false
	}

	return false
}

func isMatchGlobPattern(pattern string, value string, param *common.URL) bool {
	if param != nil && strings.HasPrefix(pattern, "$") {
		pattern = param.GetRawParameter(pattern[1:])
	}
	if "*" == pattern {
		return true
	}
	if pattern == "" && value == "" {
		return true
	}
	if pattern == "" || value == "" {
		return false
	}
	i := strings.LastIndex(pattern, "*")
	switch i {
	case -1:
		return value == pattern
	case len(pattern) - 1:
		return strings.HasPrefix(value, pattern[0:i])
	case 0:
		return strings.HasSuffix(value, pattern[:i+1])
	default:
		prefix := pattern[0:1]
		suffix := pattern[i+1:]
		return strings.HasPrefix(value, prefix) && strings.HasSuffix(value, suffix)

	}
}
