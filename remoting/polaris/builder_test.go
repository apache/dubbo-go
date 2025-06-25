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

package polaris

import (
	"net/url"
	"testing"
)

import (
	"github.com/stretchr/testify/assert"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
)

func TestGetPolarisConfigByUrl(t *testing.T) {
	regurl := getRegUrl()
	err := InitSDKContext(regurl)

	assert.Nil(t, err)
	assert.ElementsMatch(t, []string{"127.0.0.1:8091"}, sdkCtx.GetConfig().GetGlobal().GetServerConnector().GetAddresses(), "server address")
}

func getRegUrl() *common.URL {
	regurlMap := url.Values{}
	regurl, _ := common.NewURL("registry://127.0.0.1:8091", common.WithParams(regurlMap))
	return regurl
}
