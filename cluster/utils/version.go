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

package utils

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

type Version []int

var (
	parseVersionRe = regexp.MustCompile(`^[Vv](\d+(\.\d+)*)$`)
	V3_1, _        = ParseVersion("v3.1")
)

func ParseVersion(versionStr string) (Version, error) {
	if versionContainsIllegalCharacters(versionStr) {
		return nil, fmt.Errorf("illegal version string: %s , parse fail ", versionStr)
	}
	matches := parseVersionRe.FindStringSubmatch(versionStr)
	if matches == nil {
		return nil, fmt.Errorf("invalid version string format")
	}

	versionParts := strings.Split(matches[1], ".")
	version := make([]int, len(versionParts))

	for i, part := range versionParts {
		number, err := strconv.Atoi(part)
		if err != nil {
			return nil, fmt.Errorf("invalid version number part: %s", part)
		}
		version[i] = number
	}

	return version, nil
}

const (
	versionEqual   = 0
	versionLess    = -1
	versionGreater = 1
)

func (v Version) Equal(target Version) bool {
	return v.compareVersions(target) == versionEqual
}
func (v Version) Less(target Version) bool {
	return v.compareVersions(target) == versionLess
}
func (v Version) Greater(target Version) bool {
	return v.compareVersions(target) == versionGreater
}

func (v Version) compareVersions(target Version) int {
	for i := 0; i < len(v) || i < len(target); i++ {
		v1 := 0
		if i < len(v) {
			v1 = v[i]
		}
		v2 := 0
		if i < len(target) {
			v2 = target[i]
		}
		if v1 > v2 {
			return versionGreater
		} else if v1 < v2 {
			return versionLess
		}
	}

	if len(v) > len(target) {
		return versionGreater
	} else if len(v) < len(target) {
		return versionLess
	} else {
		return versionEqual
	}
}

var versionContainsIllegalCharactersRe = regexp.MustCompile(`^[Vv0-9.]+$`)

func versionContainsIllegalCharacters(s string) bool {
	return !versionContainsIllegalCharactersRe.MatchString(s)
}
