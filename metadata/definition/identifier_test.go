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
	"testing"
)

import (
	"github.com/stretchr/testify/assert"
)

func TestDataIDMatchesJavaUniqueKey(t *testing.T) {
	cases := []struct {
		name string
		args [4]string
		want string
	}{
		{
			name: "fully populated",
			args: [4]string{"org.example.UserService", "1.0.0", "g1", "app"},
			want: "org.example.UserService:1.0.0:g1:provider:app",
		},
		{
			// Java's KeyTypeEnum.build appends a separator for every argument
			// unconditionally, so empty version and group leave empty segments
			// rather than collapsing.
			name: "no version or group",
			args: [4]string{"org.example.UserService", "", "", "app"},
			want: "org.example.UserService:::provider:app",
		},
		{
			name: "version only",
			args: [4]string{"org.example.UserService", "1.0.0", "", "app"},
			want: "org.example.UserService:1.0.0::provider:app",
		},
		{
			name: "group only",
			args: [4]string{"org.example.UserService", "", "g1", "app"},
			want: "org.example.UserService::g1:provider:app",
		},
		{
			// StringUtils.isBlank treats whitespace as absent on the Java side.
			name: "whitespace is blank",
			args: [4]string{"org.example.UserService", "  ", "\t", "app"},
			want: "org.example.UserService:::provider:app",
		},
		{
			// A Go import path is a legal Nacos dataId; slashes need no escaping.
			name: "go style interface name",
			args: [4]string{"github.com/example/api.UserService", "", "", "app"},
			want: "github.com/example/api.UserService:::provider:app",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := DataID(tc.args[0], tc.args[1], tc.args[2], tc.args[3])
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestDataIDIsMatchedByAdminWatcher(t *testing.T) {
	// Admin's Nacos watcher fuzzy-searches "*:provider:*" under the dubbo group.
	// A dataId that does not contain that literal is invisible no matter how
	// well-formed the JSON body is.
	for _, version := range []string{"", "1.0.0"} {
		for _, group := range []string{"", "g1"} {
			id := DataID("org.example.UserService", version, group, "app")
			assert.Contains(t, id, ":provider:",
				"dataId must contain the substring Admin searches for")
		}
	}
}

func TestDataIDAlwaysHasFiveSegments(t *testing.T) {
	// Admin does not parse the dataId, but Java's identifier is fixed-arity and
	// divergence here would put a Go provider on a different key than a Java one
	// exporting the same service.
	id := DataID("org.example.UserService", "", "", "app")
	assert.Len(t, strings.Split(id, ":"), 5)
}

func TestMetadataGroupIsFixed(t *testing.T) {
	// Not derived from the report's own group config: Admin hardcodes this, and
	// the report's group serves application-level metadata instead.
	assert.Equal(t, "dubbo", MetadataGroup)
}
