package match_judger

import (
	"github.com/apache/dubbo-go/config"
	"regexp"
	"strings"
)

type StringMatchJudger struct {
	config.StringMatch
}

func (smj *StringMatchJudger) Judge(input string) bool {
	if smj.Exact != "" {
		return input == smj.Exact
	}
	if smj.Prefix != "" {
		return strings.HasPrefix(input, smj.Prefix)
	}
	if smj.Regex != "" {
		ok, err := regexp.MatchString(smj.Regex, input)
		return ok && err == nil
	}
	if smj.NoEmpty != "" {
		return input != ""
	}
	if smj.Empty != "" {
		return input == ""
	}
	return true
}

func NewStringMatchJudger(matchConf *config.StringMatch) *StringMatchJudger {
	return &StringMatchJudger{
		StringMatch: *matchConf,
	}
}
