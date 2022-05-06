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

import (
	"dubbo.apache.org/dubbo-go/v3/common/constant"
)

func TestApplicationConfig(t *testing.T) {

	err := Load(WithPath("./testdata/config/application/application.yaml"))
	assert.Nil(t, err)

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
	assert.Nil(t, err)
	err = application.Init()

	assert.Nil(t, err)
	assert.Equal(t, application.Name, "name")
	assert.Equal(t, application.Organization, "organization")
	assert.Equal(t, application.Module, "module")
	assert.Equal(t, application.Version, "version")
	assert.Equal(t, application.Owner, "owner")
	assert.Equal(t, application.Environment, "environment")
	assert.Equal(t, application.MetadataType, "metadataType")
	assert.Equal(t, application.Prefix(), constant.ApplicationConfigPrefix)
}
