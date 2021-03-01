package match_judger

import (
	"github.com/apache/dubbo-go/config"
)

type DoubleMatchJudger struct {
	config.DoubleMatch
}

func (dmj *DoubleMatchJudger) Judge(input float64) bool {
	if dmj.Exact != 0 {
		return input == dmj.Exact
	}
	if dmj.Range != nil {
		return newDoubleRangeMatchJudger(dmj.Range).Judge(input)
	}
	if dmj.Mode != 0 {
		// todo  mod  match ??
	}
	return true
}

func newDoubleMatchJudger(matchConf *config.DoubleMatch) *DoubleMatchJudger {
	return &DoubleMatchJudger{
		DoubleMatch: *matchConf,
	}
}
