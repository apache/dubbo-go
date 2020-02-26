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

package auth

import (
	"net/url"
	"testing"
)

import (
	"github.com/stretchr/testify/assert"
)

import (
	"github.com/apache/dubbo-go/common"
	"github.com/apache/dubbo-go/common/constant"
	invocation2 "github.com/apache/dubbo-go/protocol/invocation"
)

func TestDefaultAccesskeyStorage_GetAccesskeyPair(t *testing.T) {
	url := common.NewURLWithOptions(
		common.WithParams(url.Values{}),
		common.WithParamsValue(constant.SECRET_ACCESS_KEY_KEY, "skey"),
		common.WithParamsValue(constant.ACCESS_KEY_ID_KEY, "akey"))
	invocation := &invocation2.RPCInvocation{}
	storage := GetDefaultAccesskeyStorage()
	accesskeyPair := storage.GetAccessKeyPair(invocation, url)
	assert.Equal(t, "skey", accesskeyPair.SecretKey)
	assert.Equal(t, "akey", accesskeyPair.AccessKey)
}
