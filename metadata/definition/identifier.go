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

package definition

import (
	"strings"
)

const (
	// keySeparator matches Java's MetadataConstants.KEY_SEPARATOR.
	keySeparator = ":"

	// providerSide is the fixed side segment of the identifier. Consumer-side
	// definitions are not published by dubbo-go.
	providerSide = "provider"

	// MetadataGroup is the metadata-center group interface-level definitions are
	// written to.
	//
	// This is fixed rather than derived from the metadata report's own group
	// config, and the two must not be conflated. dubbo-go's Nacos report
	// defaults its group to DEFAULT_GROUP, and PublishAppMetadata's
	// Java-compatible format even uses the revision as the Nacos group — neither
	// has anything to do with "dubbo".
	//
	// The value is dictated by the consumer: Dubbo Admin's Nacos watcher
	// hardcodes a fuzzy search for "*:provider:*" under the "dubbo" group. On
	// the Java side this is a metadata-report config option that also defaults
	// to "dubbo"; a Java provider configured with a different group is likewise
	// invisible to Admin. That is a pre-existing constraint of the discovery
	// chain, not something this package introduces.
	MetadataGroup = "dubbo"
)

// DataID builds the metadata-center key for one interface-level definition.
//
// The layout matches Java's MetadataIdentifier.getUniqueKey(UNIQUE_KEY):
//
//	{serviceInterface}:{version}:{group}:provider:{application}
//
// Empty segments are preserved, never collapsed, so a service with no version
// and no group yields consecutive separators:
//
//	org.example.UserService:::provider:app
//
// Admin matches on the "*:provider:*" substring and reconstructs every field
// from the JSON body, so the dataId only has to be findable — but it still has
// to be byte-identical to Java's, or the same service published by a Go and a
// Java provider would occupy two different keys.
func DataID(serviceInterface, version, group, application string) string {
	return joinKey(serviceInterface, version, group, providerSide, application)
}

// joinKey concatenates parts with the Java separator, mirroring
// KeyTypeEnum.build: every part contributes a segment unconditionally, and a
// blank part contributes an empty one.
//
// Blank (not merely empty) is the Java test — StringUtils.isBlank treats a
// whitespace-only value as absent — so a group of " " must produce the same key
// as a group of "".
func joinKey(parts ...string) string {
	segments := make([]string, len(parts))
	for i, part := range parts {
		if strings.TrimSpace(part) == "" {
			segments[i] = ""
			continue
		}
		segments[i] = part
	}
	return strings.Join(segments, keySeparator)
}
