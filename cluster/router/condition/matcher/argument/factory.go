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

package argument

import (
	"strings"
)

import (
	"dubbo.apache.org/dubbo-go/v3/cluster/router/condition/matcher"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/common/extension"
)

func init() {
	extension.SetMatcherFactory("argument", NewArgumentMatcherFactory)
}

// MatcherFactory matcher factory
type MatcherFactory struct {
}

// NewArgumentMatcherFactory constructs a new argument.MatcherFactory
func NewArgumentMatcherFactory() matcher.ConditionMatcherFactory {
	return &MatcherFactory{}
}

func (m *MatcherFactory) ShouldMatch(key string) bool {
	return strings.HasPrefix(key, constant.ARGUMENTS)
}

// NewMatcher construct a new matcher
func (m *MatcherFactory) NewMatcher(key string) matcher.ConditionMatcher {
	return NewConditionMatcher(key)
}

func (c *MatcherFactory) Priority() int64 {
	return 300
}
