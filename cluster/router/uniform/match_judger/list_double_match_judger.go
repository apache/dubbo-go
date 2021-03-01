package match_judger

import (
	"github.com/apache/dubbo-go/config"
)

type ListDoubleMatchJudger struct {
	config.ListDoubleMatch
}

func (lsmj *ListDoubleMatchJudger) Judge(input float64) bool {
	for _, v := range lsmj.Oneof {
		if newDoubleMatchJudger(v).Judge(input) {
			return true
		}
	}
	return false
}

func newListDoubleMatchJudger(matchConf *config.ListDoubleMatch) *ListDoubleMatchJudger {
	return &ListDoubleMatchJudger{
		ListDoubleMatch: *matchConf,
	}
}
