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
)

import (
	"dubbo.apache.org/dubbo-go/v3/config"
)

func TestInitServer(t *testing.T) {
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
	initServer("dubbo")
	config.SetRootConfig(*originRootConf)
	assert.NotNil(t, srvConf)
}
