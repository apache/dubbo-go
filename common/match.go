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

package common

import (
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"fmt"
	"net"
	"regexp"
	"strings"
)

type ParamMatch struct {
	Key   string      `yaml:"key" json:"key,omitempty" property:"key"`
	Value StringMatch `yaml:"value" json:"value,omitempty" property:"value"`
}

func (p ParamMatch) IsMatch(url *URL) bool {
	return p.Value.IsMatch(url.GetParam(p.Key, ""))
}

type StringMatch struct {
	Exact    string `yaml:"exact" json:"exact,omitempty" property:"exact"`
	Prefix   string `yaml:"prefix" json:"prefix,omitempty" property:"prefix"`
	Regex    string `yaml:"regex" json:"regex,omitempty" property:"regex"`
	Noempty  string `yaml:"noempty" json:"noempty,omitempty" property:"noempty"`
	Empty    string `yaml:"empty" json:"empty,omitempty" property:"empty"`
	Wildcard string `yaml:"wildcard" json:"wildcard,omitempty" property:"wildcard"`
}

func (p StringMatch) IsMatch(value string) bool {
	if p.Exact != "" {
		return p.Exact == value
	} else if p.Prefix != "" {
		return strings.HasPrefix(value, p.Prefix)
	} else if p.Regex != "" {
		match, _ := regexp.MatchString(p.Regex, value)
		return match
	} else if p.Wildcard != "" {
		return value == p.Wildcard || constant.AnyValue == p.Wildcard
	} else if p.Empty != "" {
		return value == ""
	} else if p.Noempty != "" {
		return value != ""
	}
	return false
}

type AddressMatch struct {
	Wildcard string `yaml:"wildcard" json:"wildcard,omitempty" property:"wildcard"`
	Cird     string `yaml:"cird" json:"cird,omitempty" property:"cird"`
	Exact    string `yaml:"exact" json:"exact,omitempty" property:"exact"`
}

func (p AddressMatch) IsMatch(value string) bool {
	if p.Cird != "" && value != "" {
		_, ipnet, err := net.ParseCIDR(p.Cird)
		if err != nil {
			fmt.Println("Error", p.Cird, err)
			return false
		}
		return ipnet.Contains(net.ParseIP(value))
	}
	if p.Wildcard != "" && value != "" {
		if constant.AnyValue == value || constant.AnyHostValue == value {
			return true
		}
		return IsMatchGlobPattern(p.Wildcard, value)
	}
	if p.Exact != "" && value != "" {
		return p.Exact == value
	}
	return false
}

type ListStringMatch struct {
	Oneof []StringMatch `yaml:"oneof" json:"oneof,omitempty" property:"oneof"`
}

func (p ListStringMatch) IsMatch(value string) bool {
	for _, match := range p.Oneof {
		if match.IsMatch(value) {
			return true
		}
	}
	return false
}
