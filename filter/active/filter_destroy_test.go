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

package active

import (
	"context"
	"fmt"
	"strconv"
	"testing"
)

import (
	"github.com/stretchr/testify/require"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/protocol/base"
	"dubbo.apache.org/dubbo-go/v3/protocol/invocation"
	"dubbo.apache.org/dubbo-go/v3/protocol/result"
)

type destroyOnInvokeInvoker struct {
	*base.BaseInvoker
}

func (d *destroyOnInvokeInvoker) Invoke(_ context.Context, _ base.Invocation) result.Result {
	// Simulate an invoker being destroyed while a request is in-flight.
	d.Destroy()
	return &result.RPCResult{}
}

func TestActiveFilterOnResponseWithDestroyedInvoker(t *testing.T) {
	base.CleanAllStatus()
	defer base.CleanAllStatus()

	invoc := invocation.NewRPCInvocation("test", []any{"OK"}, map[string]any{
		dubboInvokeStartTime: strconv.FormatInt(base.CurrentTimeMillis(), 10),
	})
	url, _ := common.NewURL(fmt.Sprintf("dubbo://%s:%d/com.ikurento.user.UserProvider", constant.LocalHostValue, constant.DefaultPort))
	invoker := &destroyOnInvokeInvoker{BaseInvoker: base.NewBaseInvoker(url)}

	filter := activeFilter{}
	require.NotPanics(t, func() {
		res := filter.Invoke(context.Background(), invoker, invoc)
		filter.OnResponse(context.Background(), res, invoker, invoc)
	})
}
