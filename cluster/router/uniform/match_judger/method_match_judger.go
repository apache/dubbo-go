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
	}

	// todo now argc Must not be zero, else it will cause unexpected result
	if mmj.Argc != 0 && len(invocation.ParameterValues()) != mmj.Argc {
		return false
	}

	if mmj.Args != nil {
		params := invocation.ParameterValues()
		for _, v := range mmj.Args {
			index := int(v.Index)
			if index > len(params) || index < 1 {
				return false
			}
			value := params[index-1]
			if value.Type().String() != v.Type {
				return false
			}
			switch v.Type {
			case "string":
				if !newListStringMatchJudger(v.StrValue).Judge(value.String()) {
					return false
				}
			case "float", "int":
				// todo now numbers Must not be zero, else it will ignore this match
				if !newListDoubleMatchJudger(v.NumValue).Judge(value.Float()) {
					return false
				}
			case "bool":
				if !newBoolMatchJudger(v.BoolValue).Judge(value.Bool()) {
					return false
				}
			default:
			}
		}
	}
	if mmj.Argp != nil {
		// todo Argp match judge ??? conflict to args?
	}
	if mmj.Headers != nil {
		// todo Headers match judge: reserve for triple
	}
	return true
}

func NewMethodMatchJudger(matchConf *config.DubboMethodMatch) *MethodMatchJudger {
	return &MethodMatchJudger{
		DubboMethodMatch: *matchConf,
	}
}
