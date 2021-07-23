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

type AttachmentMatchJudger struct {
	config.DubboAttachmentMatch
}

// nolint
func (amj *AttachmentMatchJudger) Judge(invocation protocol.Invocation) bool {
	invAttaMap := invocation.Attachments()
	if amj.EagleeyeContext != nil && !judge(amj.EagleeyeContext, invAttaMap) {
		return false
	}

	return amj.DubboContext == nil || judge(amj.DubboContext, invAttaMap)
}

func judge(condition map[string]*config.StringMatch, invAttaMap map[string]interface{}) bool {
	for k, v := range condition {
		invAttaValue, ok := invAttaMap[k]
		if !ok {
			if v.Empty == "" {
				return false
			}
			continue
		}
		// exist this key
		str, ok := invAttaValue.(string)
		if !ok {
			return false
		}
		strJudger := NewStringMatchJudger(v)
		if !strJudger.Judge(str) {
			return false
		}
	}

	return true
}

// nolint
func NewAttachmentMatchJudger(matchConf *config.DubboAttachmentMatch) *AttachmentMatchJudger {
	return &AttachmentMatchJudger{
		DubboAttachmentMatch: *matchConf,
	}
}
