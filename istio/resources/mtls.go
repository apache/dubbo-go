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

package resources

import "strings"

// MutualTLSMode is the mutual TLS mode specified by authentication policy.
type MutualTLSMode int

const (
	// MTLSUnknown is used to indicate the variable hasn't been initialized correctly (with the authentication policy).
	MTLSUnknown MutualTLSMode = iota

	// MTLSDisable if authentication policy disable mTLS.
	MTLSDisable

	// MTLSPermissive if authentication policy enable mTLS in permissive mode.
	MTLSPermissive

	// MTLSStrict if authentication policy enable mTLS in strict mode.
	MTLSStrict
)

var MutualTLSModeToStringMap = map[MutualTLSMode]string{
	MTLSUnknown:    "UNKNOWN",
	MTLSDisable:    "DISABLE",
	MTLSPermissive: "PERMISSIVE",
	MTLSStrict:     "STRICT",
}

var MutualTLSModeFromStringMap = map[string]MutualTLSMode{
	"UNKNOWN":    MTLSUnknown,
	"DISABLE":    MTLSDisable,
	"PERMISSIVE": MTLSPermissive,
	"STRICT":     MTLSStrict,
}

func MutualTLSModeToString(mode MutualTLSMode) string {
	str, ok := MutualTLSModeToStringMap[mode]
	if !ok {
		return "UNKNOWN"
	}
	return str
}

func StringToMutualTLSMode(str string) MutualTLSMode {
	mode, ok := MutualTLSModeFromStringMap[strings.ToUpper(str)]
	if !ok {
		return MTLSUnknown
	}
	return mode
}

type XdsTLSMode struct {
	IsTls       bool
	IsRawBuffer bool
}

func (t XdsTLSMode) GetMutualTLSMode() MutualTLSMode {
	if t.IsTls && t.IsRawBuffer {
		return MTLSPermissive
	}

	if t.IsTls {
		return MTLSStrict
	}

	return MTLSDisable
}
