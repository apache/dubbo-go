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

package config

import (
	"regexp"
	"strings"
)

import (
	"github.com/apache/dubbo-go/common/constant"
)

func mergeValue(str1, str2, def string) string {
	if str1 == "" && str2 == "" {
		return def
	}
	str := "," + strings.Trim(str1, ",")
	if str1 == "" {
		str = "," + strings.Trim(str2, ",")
	} else if str2 != "" {
		str = str + "," + strings.Trim(str2, ",")
	}
	defKey := strings.Contains(str, ","+constant.DEFAULT_KEY)
	if !defKey {
		str = "," + constant.DEFAULT_KEY + str
	}
	str = strings.TrimPrefix(strings.Replace(str, ","+constant.DEFAULT_KEY, ","+def, -1), ",")

	strArr := strings.Split(str, ",")
	strMap := make(map[string][]int)
	for k, v := range strArr {
		add := true
		if strings.HasPrefix(v, "-") {
			v = v[1:]
			add = false
		}
		if _, ok := strMap[v]; !ok {
			if add {
				strMap[v] = []int{1, k}
			}
		} else {
			if add {
				strMap[v][0] += 1
				strMap[v] = append(strMap[v], k)
			} else {
				strMap[v][0] -= 1
				strMap[v] = strMap[v][:len(strMap[v])-1]
			}
		}
	}
	strArr = make([]string, len(strArr))
	for key, value := range strMap {
		if value[0] == 0 {
			continue
		}
		for i := 1; i < len(value); i++ {
			strArr[value[i]] = key
		}
	}
	reg := regexp.MustCompile("[,]+")
	str = reg.ReplaceAllString(strings.Join(strArr, ","), ",")
	return strings.Trim(str, ",")
}
