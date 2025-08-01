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

package seata

import (
	"context"
	"testing"
)

import (
	"github.com/stretchr/testify/assert"
)

import (
	"dubbo.apache.org/dubbo-go/v3/protocol/base"
	"dubbo.apache.org/dubbo-go/v3/protocol/invocation"
	"dubbo.apache.org/dubbo-go/v3/protocol/result"
)

type testMockSeataInvoker struct {
	base.BaseInvoker
}

func (iv *testMockSeataInvoker) Invoke(ctx context.Context, _ base.Invocation) result.Result {
	val := ctx.Value(SEATA_XID)
	if val != nil {
		xid, ok := val.(string)
		if ok {
			return &result.RPCResult{Rest: xid}
		}
	}
	return &result.RPCResult{}
}

func TestSeataFilter_Invoke(t *testing.T) {
	filter := &seataFilter{}
	result := filter.Invoke(context.Background(), &testMockSeataInvoker{}, invocation.NewRPCInvocation("$echo",
		[]any{"OK"}, map[string]any{
			string(SEATA_XID): "10.30.21.227:8091:2000047792",
		}))
	assert.Equal(t, "10.30.21.227:8091:2000047792", result.Result())
}
