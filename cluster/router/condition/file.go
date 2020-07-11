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

package condition

import (
	"encoding/base64"
	"net/url"
	"strconv"
	"strings"
	"sync"
)

import (
	perrors "github.com/pkg/errors"
)

import (
	"github.com/apache/dubbo-go/common"
	"github.com/apache/dubbo-go/common/constant"
)

// FileConditionRouter Use for parse config file of condition router
type FileConditionRouter struct {
	listenableRouter
	parseOnce sync.Once
	url       common.URL
}

// NewFileConditionRouter Create file condition router instance with content ( from config file)
func NewFileConditionRouter(content []byte) (*FileConditionRouter, error) {
	fileRouter := &FileConditionRouter{}
	rule, err := getRule(string(content))
	if err != nil {
		return nil, perrors.Errorf("yaml.Unmarshal() failed , error:%v", perrors.WithStack(err))
	}

	if !rule.Valid {
		return nil, perrors.Errorf("rule content is not verify for condition router , error:%v", perrors.WithStack(err))
	}

	fileRouter.generateConditions(rule)

	return fileRouter, nil
}

// URL Return URL in file condition router n
func (f *FileConditionRouter) URL() common.URL {
	f.parseOnce.Do(func() {
		routerRule := f.routerRule
		rule := parseCondition(routerRule.Conditions)
		f.url = *common.NewURLWithOptions(
			common.WithProtocol(constant.CONDITION_ROUTE_PROTOCOL),
			common.WithIp(constant.ANYHOST_VALUE),
			common.WithParams(url.Values{}),
			common.WithParamsValue(constant.RouterForce, strconv.FormatBool(routerRule.Force)),
			common.WithParamsValue(constant.RouterPriority, strconv.Itoa(routerRule.Priority)),
			common.WithParamsValue(constant.RULE_KEY, base64.URLEncoding.EncodeToString([]byte(rule))),
			common.WithParamsValue(constant.ROUTER_KEY, constant.CONDITION_ROUTE_PROTOCOL),
			common.WithParamsValue(constant.CATEGORY_KEY, constant.ROUTERS_CATEGORY),
			common.WithParamsValue(constant.RouterRuleKey, routerRule.Key),
			common.WithParamsValue(constant.RouterScope, routerRule.Scope),
		)
	})
	return f.url
}

func parseCondition(conditions []string) string {
	var when, then string
	for _, condition := range conditions {
		condition = strings.Trim(condition, " ")
		if strings.Contains(condition, "=>") {
			array := strings.SplitN(condition, "=>", 2)
			consumer := strings.Trim(array[0], " ")
			provider := strings.Trim(array[1], " ")
			if len(consumer) != 0 {
				if len(when) != 0 {
					when = strings.Join([]string{when, consumer}, " & ")
				} else {
					when = consumer
				}
			}
			if len(provider) != 0 {
				if len(then) != 0 {
					then = strings.Join([]string{then, provider}, " & ")
				} else {
					then = provider
				}
			}
		}
	}
	return strings.Join([]string{when, then}, " => ")
}
