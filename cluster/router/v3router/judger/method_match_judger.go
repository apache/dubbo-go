/*
 * Licensed to the Apache Software Foundation (ASF) under one or more
 * contributor license agreements.  See the NOTICE file distributed with
 * this work for additional information regarding copyright ownership.
 * The ASF licenses this file to You under the Apache License, Version 2.0
 * (the "License"); you may not use this file except in compliance with
 * the License.  You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package judger

import (
	"dubbo.apache.org/dubbo-go/v3/config"
	"dubbo.apache.org/dubbo-go/v3/protocol"
)

// nolint
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
				// FIXME int invoke Float may cause panic
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
	// todo Argp match judge ??? conflict to args?
	//if mmj.Argp != nil {
	//
	//}
	// todo Headers match judge: reserve for triple
	//if mmj.Headers != nil {
	//
	//}
	return true
}

// nolint
func NewMethodMatchJudger(matchConf *config.DubboMethodMatch) *MethodMatchJudger {
	return &MethodMatchJudger{
		DubboMethodMatch: *matchConf,
	}
}
