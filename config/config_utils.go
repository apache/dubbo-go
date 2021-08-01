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
	"fmt"
	"github.com/go-playground/validator/v10"
	"github.com/pkg/errors"
	"regexp"
	"strings"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common/constant"
)

var validate *validator.Validate

func init() {
	validate = validator.New()
}

func mergeValue(str1, str2, def string) string {
	if str1 == "" && str2 == "" {
		return def
	}
	s1 := strings.Split(str1, ",")
	s2 := strings.Split(str2, ",")
	str := "," + strings.Join(append(s1, s2...), ",")
	defKey := strings.Contains(str, ","+constant.DEFAULT_KEY)
	if !defKey {
		str = "," + constant.DEFAULT_KEY + str
	}
	str = strings.TrimPrefix(strings.Replace(str, ","+constant.DEFAULT_KEY, ","+def, -1), ",")
	return removeMinus(strings.Split(str, ","))
}

func removeMinus(strArr []string) string {
	if len(strArr) == 0 {
		return ""
	}
	var normalStr string
	var minusStrArr []string
	for _, v := range strArr {
		if strings.HasPrefix(v, "-") {
			minusStrArr = append(minusStrArr, v[1:])
		} else {
			normalStr += fmt.Sprintf(",%s", v)
		}
	}
	normalStr = strings.Trim(normalStr, ",")
	for _, v := range minusStrArr {
		normalStr = strings.Replace(normalStr, v, "", 1)
	}
	reg := regexp.MustCompile("[,]+")
	normalStr = reg.ReplaceAllString(strings.Trim(normalStr, ","), ",")
	return normalStr
}

// removeDuplicateElement remove duplicate element
func removeDuplicateElement(items []string) []string {
	result := make([]string, 0, len(items))
	temp := map[string]struct{}{}
	for _, item := range items {
		if _, ok := temp[item]; !ok && item != "" {
			temp[item] = struct{}{}
			result = append(result, item)
		}
	}
	return result
}

// translateRegistryIds string "nacos,zk" => ["nacos","zk"]
func translateRegistryIds(registryIds []string) []string {
	ids := make([]string, 0)
	for _, id := range registryIds {
		ids = append(ids, strings.Split(id, ",")...)
	}
	return removeDuplicateElement(ids)
}

func verify(s interface{}) error {
	if err := validate.Struct(s); err != nil {
		errs := err.(validator.ValidationErrors)
		var slice []string
		for _, msg := range errs {
			slice = append(slice, msg.Error())
		}
		return errors.New(strings.Join(slice, ","))
	}
	return nil
}
