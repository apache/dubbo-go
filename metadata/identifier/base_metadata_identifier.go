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
	"net/url"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common/constant"
)

// BaseMetadataIdentifier defined for describe the Metadata base identify
type IMetadataIdentifier interface {
	GetFilePathKey() string
	GetIdentifierKey() string
}

// BaseMetadataIdentifier is the base implement of BaseMetadataIdentifier interface
type BaseMetadataIdentifier struct {
	ServiceInterface string
	Version          string
	Group            string
	Side             string
}

// joinParams will join the specified char in slice, and build as string
func joinParams(joinChar string, params []string) string {
	var joinedStr string
	for _, param := range params {
		joinedStr += joinChar
		joinedStr += param
	}
	return joinedStr
}

// getIdentifierKey returns string that format is service:Version:Group:Side:param1:param2...
func (mdi *BaseMetadataIdentifier) getIdentifierKey(params ...string) string {
	return mdi.ServiceInterface +
		constant.KEY_SEPARATOR + mdi.Version +
		constant.KEY_SEPARATOR + mdi.Group +
		constant.KEY_SEPARATOR + mdi.Side +
		joinParams(constant.KEY_SEPARATOR, params)
}

// getFilePathKey returns string that format is metadata/path/Version/Group/Side/param1/param2...
func (mdi *BaseMetadataIdentifier) getFilePathKey(params ...string) string {
	path := serviceToPath(mdi.ServiceInterface)

	return constant.DEFAULT_PATH_TAG +
		withPathSeparator(path) +
		withPathSeparator(mdi.Version) +
		withPathSeparator(mdi.Group) +
		withPathSeparator(mdi.Side) +
		joinParams(constant.PATH_SEPARATOR, params)
}

// serviceToPath uss URL encode to decode the @serviceInterface
func serviceToPath(serviceInterface string) string {
	if serviceInterface == constant.ANY_VALUE {
		return ""
	}
	return url.PathEscape(serviceInterface)
}

// withPathSeparator return "/" + @path
func withPathSeparator(path string) string {
	if len(path) != 0 {
		path = constant.PATH_SEPARATOR + path
	}
	return path
}

// BaseApplicationMetadataIdentifier is the base implement of BaseApplicationMetadataIdentifier interface
type BaseApplicationMetadataIdentifier struct {
	Application string
	Group       string
}

// getIdentifierKey returns string that format is application/param
func (madi *BaseApplicationMetadataIdentifier) getIdentifierKey(params ...string) string {
	return madi.Application + joinParams(constant.KEY_SEPARATOR, params)
}

// getFilePathKey returns string that format is metadata/application/revision
func (madi *BaseApplicationMetadataIdentifier) getFilePathKey(params ...string) string {
	return constant.DEFAULT_PATH_TAG +
		withPathSeparator(madi.Application) +
		joinParams(constant.PATH_SEPARATOR, params)
}
