package match_judger

import (
	"github.com/apache/dubbo-go/config"
)

type ListStringMatchJudger struct {
	config.ListStringMatch
}

func (lsmj *ListStringMatchJudger) Judge(input string) bool {
	for _, v := range lsmj.Oneof {
		if NewStringMatchJudger(v).Judge(input) {
			return true
		}
	}
	return false
}

func newListStringMatchJudger(matchConf *config.ListStringMatch) *ListStringMatchJudger {
	return &ListStringMatchJudger{
		ListStringMatch: *matchConf,
	}
}
