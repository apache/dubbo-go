package match_judger

import (
	"github.com/apache/dubbo-go/config"
)

type DoubleRangeMatchJudger struct {
	config.DoubleRangeMatch
}

func (drmj *DoubleRangeMatchJudger) Judge(input float64) bool {
	return input >= drmj.Start && input < drmj.End
}

func newDoubleRangeMatchJudger(matchConf *config.DoubleRangeMatch) *DoubleRangeMatchJudger {
	return &DoubleRangeMatchJudger{
		DoubleRangeMatch: *matchConf,
	}
}
