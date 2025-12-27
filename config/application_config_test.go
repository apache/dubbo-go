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
	"github.com/stretchr/testify/require"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	_ "dubbo.apache.org/dubbo-go/v3/metadata/report/nacos"
)

func TestApplicationConfig(t *testing.T) {

	err := Load(WithPath("./testdata/config/application/application.yaml"))
	require.NoError(t, err)

	center := rootConfig.Registries
	assert.NotNil(t, center)
}

func TestApplicationConfigBuilder(t *testing.T) {

	application := NewApplicationConfigBuilder().
		SetOrganization("organization").
		SetName("name").
		SetModule("module").
		SetVersion("version").
		SetOwner("owner").
		SetEnvironment("environment").
		SetMetadataType("metadataType").
		Build()

	err := application.check()
	require.NoError(t, err)
	err = application.Init()

	require.NoError(t, err)
	assert.Equal(t, "name", application.Name)
	assert.Equal(t, "organization", application.Organization)
	assert.Equal(t, "module", application.Module)
	assert.Equal(t, "version", application.Version)
	assert.Equal(t, "owner", application.Owner)
	assert.Equal(t, "environment", application.Environment)
	assert.Equal(t, "metadataType", application.MetadataType)
	assert.Equal(t, constant.ApplicationConfigPrefix, application.Prefix())
}
