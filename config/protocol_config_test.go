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

func TestGetProtocolsConfig(t *testing.T) {

	t.Run("empty use default", func(t *testing.T) {
		err := Load(WithPath("./testdata/config/protocol/empty_application.yaml"))
		assert.Nil(t, err)
		protocols := rootConfig.Protocols
		assert.NotNil(t, protocols)
		// default
		assert.Equal(t, "dubbo", protocols["dubbo"].Name)
		assert.Equal(t, string("20000"), protocols["dubbo"].Port)
	})

	t.Run("use config", func(t *testing.T) {
		err := Load(WithPath("./testdata/config/protocol/application.yaml"))
		assert.Nil(t, err)
		protocols := rootConfig.Protocols
		assert.NotNil(t, protocols)
		// default
		assert.Equal(t, "dubbo", protocols["dubbo"].Name)
		assert.Equal(t, string("20000"), protocols["dubbo"].Port)
	})
}
