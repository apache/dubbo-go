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
	"context"
	"strconv"
	"testing"
	"time"
)

import (
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
)

import (
	"github.com/apache/dubbo-go/common"
	"github.com/apache/dubbo-go/common/constant"
	"github.com/apache/dubbo-go/protocol"
	"github.com/apache/dubbo-go/protocol/invocation"
	"github.com/apache/dubbo-go/protocol/mock"
)

func TestProviderAuthFilter_Invoke(t *testing.T) {
	secret := "dubbo-sk"
	access := "dubbo-ak"
	url, _ := common.NewURL("dubbo://127.0.0.1:20000/com.ikurento.user.UserProvider?interface=com.ikurento.user.UserProvider&group=gg&version=2.6.0")
	url.SetParam(constant.ACCESS_KEY_ID_KEY, access)
	url.SetParam(constant.SECRET_ACCESS_KEY_KEY, secret)
	parmas := []interface{}{
		"OK",
		struct {
			Name string
			Id   int64
		}{"YUYU", 1},
	}
	inv := invocation.NewRPCInvocation("test", parmas, nil)
	requestTime := strconv.Itoa(int(time.Now().Unix() * 1000))
	signature, _ := getSignature(url, inv, secret, requestTime)

	inv = invocation.NewRPCInvocation("test", []interface{}{"OK"}, map[string]interface{}{
		constant.REQUEST_SIGNATURE_KEY: signature,
		constant.CONSUMER:              "test",
		constant.REQUEST_TIMESTAMP_KEY: requestTime,
		constant.AK_KEY:                access,
	})
	ctrl := gomock.NewController(t)
	filter := &ProviderAuthFilter{}
	defer ctrl.Finish()
	invoker := mock.NewMockInvoker(ctrl)
	result := &protocol.RPCResult{}
	invoker.EXPECT().Invoke(inv).Return(result).Times(2)
	invoker.EXPECT().GetUrl().Return(url).Times(2)
	assert.Equal(t, result, filter.Invoke(context.Background(), invoker, inv))
	url.SetParam(constant.SERVICE_AUTH_KEY, "true")
	assert.Equal(t, result, filter.Invoke(context.Background(), invoker, inv))

}
