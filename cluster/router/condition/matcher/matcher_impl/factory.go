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

package matcher_impl

import (
	"math"
	"strings"
)

import (
	"dubbo.apache.org/dubbo-go/v3/cluster/router/condition/matcher"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/common/extension"
)

func init() {
	extension.SetMatcherFactory("argument", NewArgumentMatcherFactory)
	extension.SetMatcherFactory("attachments", NewAttachmentMatcherFactory)
	extension.SetMatcherFactory("param", NewParamMatcherFactory)
}

// ArgumentMatcherFactory matcher factory
type ArgumentMatcherFactory struct {
}

// NewArgumentMatcherFactory constructs a new argument.ArgumentMatcherFactory
func NewArgumentMatcherFactory() matcher.ConditionMatcherFactory {
	return &ArgumentMatcherFactory{}
}

func (a *ArgumentMatcherFactory) ShouldMatch(key string) bool {
	return strings.HasPrefix(key, constant.Arguments)
}

// NewMatcher construct a new matcher
func (a *ArgumentMatcherFactory) NewMatcher(key string) matcher.ConditionMatcher {
	return NewArgumentConditionMatcher(key)
}

func (a *ArgumentMatcherFactory) Priority() int64 {
	return 300
}

// AttachmentMatcherFactory matcher factory
type AttachmentMatcherFactory struct {
}

// NewAttachmentMatcherFactory constructs a new attachment.AttachmentMatcherFactory
func NewAttachmentMatcherFactory() matcher.ConditionMatcherFactory {
	return &AttachmentMatcherFactory{}
}

func (a *AttachmentMatcherFactory) ShouldMatch(key string) bool {
	return strings.HasPrefix(key, constant.Attachments)
}

// NewMatcher construct a new matcher
func (a *AttachmentMatcherFactory) NewMatcher(key string) matcher.ConditionMatcher {
	return NewAttachmentConditionMatcher(key)
}

func (a *AttachmentMatcherFactory) Priority() int64 {
	return 200
}

// ParamMatcherFactory matcher factory
type ParamMatcherFactory struct {
}

// NewParamMatcherFactory constructs a new paramMatcherFactory
func NewParamMatcherFactory() matcher.ConditionMatcherFactory {
	return &ParamMatcherFactory{}
}

func (p *ParamMatcherFactory) ShouldMatch(key string) bool {
	return true
}

// NewMatcher construct a new matcher
func (p *ParamMatcherFactory) NewMatcher(key string) matcher.ConditionMatcher {
	return NewParamConditionMatcher(key)
}

// Priority make sure this is the last matcher being executed.
// This instance will be loaded separately to ensure it always gets executed as the last matcher.
func (p *ParamMatcherFactory) Priority() int64 {
	return math.MaxInt64
}
