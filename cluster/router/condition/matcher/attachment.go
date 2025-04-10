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

package matcher

import (
	"regexp"
	"strings"
)

import (
	"github.com/dubbogo/gost/log/logger"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/protocol"
)

var (
	attachmentPattern       = regexp.MustCompile("attachments\\[([a-zA-Z0-9_]+)\\]")
	notFoundAttachmentValue = "dubbo internal not found attachment condition value"
)

// AttachmentConditionMatcher analysis the attachments in the rule.
// Examples would be like this:
// "attachments[version]=1.0.0&attachments[timeout]=1000", whenCondition is that the version is equal to '1.0.0' and the timeout is equal to '1000'.
type AttachmentConditionMatcher struct {
	BaseConditionMatcher
}

func NewAttachmentConditionMatcher(key string) *AttachmentConditionMatcher {
	return &AttachmentConditionMatcher{
		*NewBaseConditionMatcher(key),
	}
}

func (a *AttachmentConditionMatcher) GetValue(sample map[string]string, url *common.URL, invocation protocol.Invocation) string {
	// split the rule
	expressArray := strings.Split(a.key, "\\.")
	attachmentExpress := expressArray[0]
	matcher := attachmentPattern.FindStringSubmatch(attachmentExpress)
	if len(matcher) == 0 {
		logger.Warn(notFoundAttachmentValue)
		return ""
	}
	// extract the attachment key
	attachmentKey := matcher[1]
	if attachmentKey == "" {
		logger.Warn(notFoundAttachmentValue)
		return ""
	}
	// extract the attachment value
	attachment, _ := invocation.GetAttachment(attachmentKey)
	return attachment
}
