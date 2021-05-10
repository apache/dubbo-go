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
	"testing"
)

import (
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/protocol"
	"dubbo.apache.org/dubbo-go/v3/protocol/invocation"
	"dubbo.apache.org/dubbo-go/v3/protocol/mock"
)

func TestConsumerSignFilter_Invoke(t *testing.T) {
	url, _ := common.NewURL("dubbo://127.0.0.1:20000/com.ikurento.user.UserProvider?interface=com.ikurento.user.UserProvider&group=gg&version=2.6.0")
	url.SetParam(constant.SECRET_ACCESS_KEY_KEY, "sk")
	url.SetParam(constant.ACCESS_KEY_ID_KEY, "ak")
	inv := invocation.NewRPCInvocation("test", []interface{}{"OK"}, nil)
	filter := &ConsumerSignFilter{}
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	invoker := mock.NewMockInvoker(ctrl)
	result := &protocol.RPCResult{}
	invoker.EXPECT().Invoke(inv).Return(result).Times(2)
	invoker.EXPECT().GetUrl().Return(url).Times(2)
	assert.Equal(t, result, filter.Invoke(context.Background(), invoker, inv))

	url.SetParam(constant.SERVICE_AUTH_KEY, "true")
	assert.Equal(t, result, filter.Invoke(context.Background(), invoker, inv))
}
