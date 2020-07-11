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
	"strings"
)

import (
	gxstrings "github.com/dubbogo/gost/strings"
)

import (
	"github.com/apache/dubbo-go/common/constant"
	"github.com/apache/dubbo-go/cluster/router"
	"github.com/apache/dubbo-go/common"
	"github.com/apache/dubbo-go/common/yaml"
)

// RouterRule RouterRule config read from config file or config center
type RouterRule struct {
	router.BaseRouterRule `yaml:",inline"`
	Conditions            []string
}

/* Parse Router raw rule parser
 * example :
 * scope: application
 * runtime: true
 * force: false
 * conditions:
 *   - >
 *     method!=sayHello =>
 *   - >
 *     ip=127.0.0.1
 *     =>
 *     1.1.1.1
 */
func getRule(rawRule string) (*RouterRule, error) {
	r := &RouterRule{}
	err := yaml.UnmarshalYML([]byte(rawRule), r)
	if err != nil {
		return r, err
	}
	r.RawRule = rawRule
	if len(r.Conditions) != 0 && r.Key != "" && (r.Scope == constant.RouterApplicationScope || r.Scope == constant.RouterServiceScope){
		r.Valid = true
	}

	return r, nil
}

// isMatchGlobalPattern Match value to param content by pattern
func isMatchGlobalPattern(pattern string, value string, param *common.URL) bool {
	if param != nil && strings.HasPrefix(pattern, "$") {
		pattern = param.GetRawParam(pattern[1:])
	}
	return gxstrings.IsMatchPattern(pattern, value)
}
