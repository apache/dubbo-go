package match_judger

import "github.com/apache/dubbo-go/config"

type BoolMatchJudger struct {
	config.BoolMatch
}

func (lsmj *BoolMatchJudger) Judge(input bool) bool {
	return input == lsmj.Exact
}

func newBoolMatchJudger(matchConf *config.BoolMatch) *BoolMatchJudger {
	return &BoolMatchJudger{
		BoolMatch: *matchConf,
	}
}
