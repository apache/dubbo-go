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

package param

import (
	"dubbo.apache.org/dubbo-go/v3/cluster/router/condition/matcher/base"
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/protocol"
)

type ParamConditionMatcher struct {
	base.BaseConditionMatcher
}

func NewParamConditionMatcher(key string) *ParamConditionMatcher {
	conditionMatcher := &ParamConditionMatcher{
		*base.NewBaseConditionMatcher(key),
	}
	conditionMatcher.Matcher = conditionMatcher
	return conditionMatcher
}

func (p *ParamConditionMatcher) GetValue(sample map[string]string, url *common.URL, invocation protocol.Invocation) string {
	return p.GetSampleValueFromURL(p.Key, sample, url, invocation)
}
