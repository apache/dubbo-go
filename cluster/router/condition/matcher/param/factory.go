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
	"math"
)

import (
	"dubbo.apache.org/dubbo-go/v3/cluster/router/condition/matcher"
	"dubbo.apache.org/dubbo-go/v3/common/extension"
)

func init() {
	extension.SetMatcherFactory("param", NewParamMatcherFactory)
}

// MatcherFactory matcher factory
type MatcherFactory struct {
}

// NewParamMatcherFactory constructs a new paramMatcherFactory
func NewParamMatcherFactory() matcher.ConditionMatcherFactory {
	return &MatcherFactory{}
}

func (m *MatcherFactory) ShouldMatch(key string) bool {
	return true
}

// NewMatcher construct a new matcher
func (m *MatcherFactory) NewMatcher(key string) matcher.ConditionMatcher {
	return NewParamConditionMatcher(key)
}

// Priority make sure this is the last matcher being executed.
// This instance will be loaded separately to ensure it always gets executed as the last matcher.
// So we don't put Active annotation here.
func (c *MatcherFactory) Priority() int64 {
	return math.MaxInt64
}
