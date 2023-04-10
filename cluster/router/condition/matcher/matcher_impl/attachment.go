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
	"regexp"
	"strings"
)

import (
	"github.com/pkg/errors"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/protocol"
)

var (
	AttachmentsPattern = regexp.MustCompile("attachments\\[(.+)\\]")
)

/*
 * analysis the attachments in the rule.
 * Examples would be like this:
 * "attachments[foo]=bar", whenCondition is that the attachment value of 'foo' is equal to 'bar'.
 */

type AttachmentConditionMatcher struct {
	BaseConditionMatcher
}

func NewAttachmentConditionMatcher(key string) *AttachmentConditionMatcher {
	conditionMatcher := &AttachmentConditionMatcher{
		*NewBaseConditionMatcher(key),
	}
	conditionMatcher.Matcher = conditionMatcher
	return conditionMatcher
}

func (a *AttachmentConditionMatcher) GetValue(sample map[string]string, url *common.URL, invocation protocol.Invocation) (string, error) {
	// split the rule
	expressArray := strings.Split(a.Key, "\\.")
	argumentExpress := expressArray[0]
	matcher := AttachmentsPattern.FindStringSubmatch(argumentExpress)
	if len(matcher) == 0 {
		return "", errors.Errorf("dubbo internal not found argument condition value")
	}

	// extract the attachment index
	attachmentKey := matcher[1]
	if attachmentKey == "" {
		return "", errors.Errorf("dubbo internal not found argument condition value")
	}

	//extract the attachment value
	attachment, _ := invocation.GetAttachment(attachmentKey)
	return attachment, nil
}
