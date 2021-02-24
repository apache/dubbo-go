package match_judger

import (
	"github.com/apache/dubbo-go/config"
	"github.com/apache/dubbo-go/protocol"
)

type MethodMatchJudger struct {
	config.DubboMethodMatch
}

// Judge Method Match Judger only judge on
func (mmj *MethodMatchJudger) Judge(invocation protocol.Invocation) bool {
	if mmj.NameMatch != nil {
		strJudger := NewStringMatchJudger(mmj.NameMatch)
		if !strJudger.Judge(invocation.MethodName()) {
			return false
		}
		return true
	}

	// todo now argc Must not be zero, else it will cause unexpected result
	if mmj.Argc != 0 && len(invocation.ParameterValues()) != mmj.Argc {
		return false
	}

	if mmj.Args != nil {
		// todo Args match judge
	}
	if mmj.Argp != nil {
		// todo Argp match judge
	}
	if mmj.Headers != nil {
		// todo Headers match judge
	}

	return true
}

func NewMethodMatchJudger(matchConf *config.DubboMethodMatch) *MethodMatchJudger {
	return &MethodMatchJudger{
		DubboMethodMatch: *matchConf,
	}
}
