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

package filter_impl

import (
	"context"
	"errors"
	"strconv"
	"testing"
)

import (
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
)

import (
	"github.com/apache/dubbo-go/common"
	"github.com/apache/dubbo-go/protocol"
	"github.com/apache/dubbo-go/protocol/invocation"
	"github.com/apache/dubbo-go/protocol/mock"
)

func TestActiveFilterInvoke(t *testing.T) {
	invoc := invocation.NewRPCInvocation("test", []interface{}{"OK"}, make(map[string]interface{}, 0))
	url, _ := common.NewURL("dubbo://192.168.10.10:20000/com.ikurento.user.UserProvider")
	filter := ActiveFilter{}
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	invoker := mock.NewMockInvoker(ctrl)
	invoker.EXPECT().Invoke(gomock.Any()).Return(nil)
	invoker.EXPECT().GetUrl().Return(url).Times(1)
	filter.Invoke(context.Background(), invoker, invoc)
	assert.True(t, invoc.AttachmentsByKey(dubboInvokeStartTime, "") != "")

}

func TestActiveFilterOnResponse(t *testing.T) {
	c := protocol.CurrentTimeMillis()
	elapsed := 100
	invoc := invocation.NewRPCInvocation("test", []interface{}{"OK"}, map[string]interface{}{
		dubboInvokeStartTime: strconv.FormatInt(c-int64(elapsed), 10),
	})
	url, _ := common.NewURL("dubbo://192.168.10.10:20000/com.ikurento.user.UserProvider")
	filter := ActiveFilter{}
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	invoker := mock.NewMockInvoker(ctrl)
	invoker.EXPECT().GetUrl().Return(url).Times(1)
	result := &protocol.RPCResult{
		Err: errors.New("test"),
	}
	filter.OnResponse(nil, result, invoker, invoc)
	methodStatus := protocol.GetMethodStatus(url, "test")
	urlStatus := protocol.GetURLStatus(url)

	assert.Equal(t, int32(1), methodStatus.GetTotal())
	assert.Equal(t, int32(1), urlStatus.GetTotal())
	assert.Equal(t, int32(1), methodStatus.GetFailed())
	assert.Equal(t, int32(1), urlStatus.GetFailed())
	assert.Equal(t, int32(1), methodStatus.GetSuccessiveRequestFailureCount())
	assert.Equal(t, int32(1), urlStatus.GetSuccessiveRequestFailureCount())
	assert.True(t, methodStatus.GetFailedElapsed() >= int64(elapsed))
	assert.True(t, urlStatus.GetFailedElapsed() >= int64(elapsed))
	assert.True(t, urlStatus.GetLastRequestFailedTimestamp() != int64(0))
	assert.True(t, methodStatus.GetLastRequestFailedTimestamp() != int64(0))

}
