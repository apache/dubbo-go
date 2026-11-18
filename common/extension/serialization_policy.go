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

package extension

import (
	"strings"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
)

const serializationPolicyAttribute = "serialization.receive-policy"

type serializationPolicy struct {
	names    []string
	explicit bool
}

// SnapshotSerializationPolicy captures local receive settings before discovery
// merges. A provider's advertised parameters must not broaden a client policy.
func SnapshotSerializationPolicy(url *common.URL) {
	names, explicit := serializationPolicyFromParams(url)
	url.SetAttribute(serializationPolicyAttribute, &serializationPolicy{names: names, explicit: explicit})
}
func AcceptedSerializations(url *common.URL) ([]string, bool) {
	if raw, ok := url.GetAttribute(serializationPolicyAttribute); ok {
		if policy, ok := raw.(*serializationPolicy); ok {
			return append([]string(nil), policy.names...), policy.explicit
		}
	}
	return serializationPolicyFromParams(url)
}
func serializationPolicyFromParams(url *common.URL) ([]string, bool) {
	values, explicit := url.CopyParams()["accepted-serializations"]
	if !explicit {
		return nil, false
	}
	raw := strings.Join(values, ",")
	if raw == "" {
		return []string{}, true
	}
	return strings.Split(raw, ","), true
}
