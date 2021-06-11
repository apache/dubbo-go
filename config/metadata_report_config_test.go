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
	"testing"
)

import (
	"github.com/stretchr/testify/assert"
)

func TestMetadataReportConfig_ToUrl(t *testing.T) {
	GetBaseConfig().Remotes["mock"] = &RemoteConfig{
		Address:    "127.0.0.1:2181",
		Username:   "test",
		Password:   "test",
		TimeoutStr: "3s",
	}
	metadataReportConfig := MetadataReportConfig{
		Protocol:  "mock",
		RemoteRef: "mock",
		Params: map[string]string{
			"k": "v",
		},
	}
	url, err := metadataReportConfig.ToUrl()
	assert.NoError(t, err)
	assert.Equal(t, "mock", url.Protocol)
	assert.Equal(t, "127.0.0.1:2181", url.Location)
	assert.Equal(t, "127.0.0.1", url.Ip)
	assert.Equal(t, "2181", url.Port)
	assert.Equal(t, "test", url.Username)
	assert.Equal(t, "test", url.Password)
	assert.Equal(t, "v", url.GetParam("k", ""))
	assert.Equal(t, "mock", url.GetParam("metadata", ""))
}
