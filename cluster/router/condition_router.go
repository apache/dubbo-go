package router

import (
	"github.com/apache/dubbo-go/cluster"
	"github.com/apache/dubbo-go/common/constant"
	"regexp"
	"strings"

	"github.com/apache/dubbo-go/common"
	"github.com/apache/dubbo-go/protocol"
	perrors "github.com/pkg/errors"
)

const (
	RoutePattern = `([&!=,]*)\\s*([^&!=,\\s]+)`
)

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
	if !c.matchWhen(url, invocation) {
		return invokers, nil
	}
	var result []protocol.Invoker
	if len(c.ThenCondition) == 0 {
		return result, nil
	}
	for _, invoker := range invokers {
		if c.matchThen(invoker.GetUrl(), url) {
			result = append(result, invoker)
		}
	}
	if len(result) > 0 {
		return result, nil
	} else if c.Force {
		//todo 日志
		return result, nil
	}
	return result, nil
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
	//rule := url.GetParam("rule", "")
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
		url.GetParamInt("Priority", 0),
		url.GetParamBool("Force", false),
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
	values := make(map[string]interface{})

	reg := regexp.MustCompile(`([&!=,]*)\s*([^&!=,\s]+)`)

	startIndex := reg.FindIndex([]byte(rule))
	matches := reg.FindAllSubmatch([]byte(rule), -1)
	for _, groups := range matches {
		separator := string(groups[1])
		content := string(groups[2])

		switch separator {
		case "":
			pair = MatchPair{}
			condition[content] = pair
		case "&":
			if r, ok := condition[content]; ok {
				pair = r
			} else {
				pair = MatchPair{}
				condition[content] = pair
			}
		case "=":
			if &pair == nil {
				return nil, perrors.Errorf("Illegal route rule \"%s\", The error char '%s' at index %d before \"%d\".", rule, separator, startIndex[0], startIndex[0])
			}
			values = pair.Matches
			values[content] = ""
		case "!=":
			if &pair == nil {
				return nil, perrors.Errorf("Illegal route rule \"%s\", The error char '%s' at index %d before \"%d\".", rule, separator, startIndex[0], startIndex[0])
			}
			values = pair.Matches
			values[content] = ""
		case ",":
			if len(values) == 0 {
				return nil, perrors.Errorf("Illegal route rule \"%s\", The error char '%s' at index %d before \"%d\".", rule, separator, startIndex[0], startIndex[0])
			}
			values[content] = ""
		default:
			return nil, perrors.Errorf("Illegal route rule \"%s\", The error char '%s' at index %d before \"%d\".", rule, separator, startIndex[0], startIndex[0])

		}
	}

	return condition, nil
	//var pair MatchPair

}

func (c *ConditionRouter) matchWhen(url common.URL, invocation protocol.Invocation) bool {

	return len(c.WhenCondition) == 0 || len(c.WhenCondition) == 0 || matchCondition(c.WhenCondition, &url, nil, invocation)
}
func (c *ConditionRouter) matchThen(url common.URL, param common.URL) bool {

	return !(len(c.ThenCondition) == 0) && matchCondition(c.ThenCondition, &url, &param, nil)
}

func matchCondition(pairs map[string]MatchPair, url *common.URL, param *common.URL, invocation protocol.Invocation) bool {
	sample := url.ToMap()
	result := false
	for key, matchPair := range pairs {
		var sampleValue string

		if invocation != nil && ((constant.METHOD_KEY == key) || (constant.METHOD_KEYS == key)) {
			sampleValue = invocation.MethodName()
		} else {
			sampleValue = sample[key]
			if &sampleValue == nil {
				sampleValue = sample[constant.DEFAULT_KEY_PREFIX+key]
			}
		}
		if &sampleValue != nil {
			if !matchPair.isMatch(sampleValue, param) {
				return false
			} else {
				result = true
			}
		} else {
			if !(len(matchPair.Matches) == 0) {
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
	Matches    map[string]interface{}
	Mismatches map[string]interface{}
}

func (pair MatchPair) isMatch(s string, param *common.URL) bool {

	return false
}
