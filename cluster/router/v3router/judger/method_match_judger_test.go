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
	"reflect"
	"testing"
)

import (
	"github.com/stretchr/testify/assert"
)

import (
	"dubbo.apache.org/dubbo-go/v3/config"
	"dubbo.apache.org/dubbo-go/v3/protocol/invocation"
)

func TestMethodMatchJudger(t *testing.T) {
	methodArgs := make([]*config.DubboMethodArg, 0)
	methodArgs = append(methodArgs, &config.DubboMethodArg{
		Index:     1,
		Type:      "string",
		StrValue:  &config.ListStringMatch{Oneof: []*config.StringMatch{{Exact: "hello world"}}},
		NumValue:  nil,
		BoolValue: nil,
	})
	methodArgs = append(methodArgs, &config.DubboMethodArg{
		Index:     2,
		Type:      "bool",
		StrValue:  nil,
		NumValue:  nil,
		BoolValue: &config.BoolMatch{Exact: true},
	})
	methodArgs = append(methodArgs, &config.DubboMethodArg{
		Index:     3,
		Type:      "float64",
		StrValue:  nil,
		NumValue:  &config.ListDoubleMatch{Oneof: []*config.DoubleMatch{{Exact: 10}}},
		BoolValue: nil,
	})

	methodMatch := &config.DubboMethodMatch{
		NameMatch: &config.StringMatch{Exact: "Greet"},
		Argc:      3,
		Args:      methodArgs,
		Argp:      nil,
		Headers:   nil,
	}

	stringValue := reflect.ValueOf("hello world")
	boolValue := reflect.ValueOf(true)
	numValue := reflect.ValueOf(10.0)
	ivc := invocation.NewRPCInvocationWithOptions(
		invocation.WithMethodName("Greet"),
		invocation.WithParameterValues([]reflect.Value{stringValue, boolValue, numValue}),
	)

	assert.False(t, NewMethodMatchJudger(&config.DubboMethodMatch{NameMatch: &config.StringMatch{Exact: "Great"}}).Judge(ivc))
	assert.False(t, NewMethodMatchJudger(&config.DubboMethodMatch{NameMatch: &config.StringMatch{Exact: "Greet"}, Argc: 1}).Judge(ivc))
	assert.True(t, NewMethodMatchJudger(methodMatch).Judge(ivc))
}
