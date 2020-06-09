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

package identifier

import (
	"encoding/base64"
)

import (
	"github.com/apache/dubbo-go/common/constant"
)

// BaseMetadataIdentifier defined for description the Metadata base identify
type BaseMetadataIdentifier interface {
	getFilePathKey(params ...string) string
	getIdentifierKey(params ...string) string
}

// BaseMetadataIdentifier is the base implement of BaseMetadataIdentifier interface
type BaseServiceMetadataIdentifier struct {
	serviceInterface string
	version          string
	group            string
	side             string
}

// joinParams will join the specified char in slice, and return a string
func joinParams(joinChar string, params []string) string {
	var joinedStr string
	for _, param := range params {
		joinedStr += joinChar
		joinedStr += param
	}
	return joinedStr
}

// getIdentifierKey will return string format is service:Version:Group:Side:param1:param2...
func (mdi *BaseServiceMetadataIdentifier) getIdentifierKey(params ...string) string {
	return mdi.serviceInterface +
		constant.KEY_SEPARATOR + mdi.version +
		constant.KEY_SEPARATOR + mdi.group +
		constant.KEY_SEPARATOR + mdi.side +
		joinParams(constant.KEY_SEPARATOR, params)
}

// getFilePathKey will return string format is metadata/path/Version/Group/Side/param1/param2...
func (mdi *BaseServiceMetadataIdentifier) getFilePathKey(params ...string) string {
	path := serviceToPath(mdi.serviceInterface)

	return constant.DEFAULT_PATH_TAG +
		withPathSeparator(path) +
		withPathSeparator(mdi.version) +
		withPathSeparator(mdi.group) +
		withPathSeparator(mdi.side) +
		joinParams(constant.PATH_SEPARATOR, params)

}

func serviceToPath(serviceInterface string) string {
	if serviceInterface == constant.ANY_VALUE {
		return ""
	} else {
		decoded, err := base64.URLEncoding.DecodeString(serviceInterface)
		if err != nil {
			return ""
		}
		return string(decoded)
	}

}

func withPathSeparator(path string) string {
	if len(path) != 0 {
		path = constant.PATH_SEPARATOR + path
	}
	return path
}
