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

package getty

import (
	"testing"
)

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/config"
	"dubbo.apache.org/dubbo-go/v3/global"
)

// TODO: Temporary compatibility with old APIs, can be removed later
func TestInitServerOldApi(t *testing.T) {
	originRootConf := config.GetRootConfig()
	rootConf := config.RootConfig{
		Protocols: map[string]*config.ProtocolConfig{
			"dubbo": {
				Name: "dubbo",
				Ip:   "127.0.0.1",
				Port: "20003",
			},
		},
	}
	config.SetRootConfig(rootConf)
	url, err := common.NewURL("dubbo://127.0.0.1:20003/test")
	require.NoError(t, err)
	initServer(url)
	config.SetRootConfig(*originRootConf)
	assert.NotNil(t, srvConf)
}

func TestInitServer(t *testing.T) {
	url, err := common.NewURL("dubbo://127.0.0.1:20003/test")
	require.NoError(t, err)
	url.SetAttribute(constant.ProtocolConfigKey, map[string]*global.ProtocolConfig{
		"dubbo": {
			Name: "dubbo",
			Ip:   "127.0.0.1",
			Port: "20003",
		},
	})
	url.SetAttribute(constant.ApplicationKey, global.ApplicationConfig{})
	initServer(url)
}
