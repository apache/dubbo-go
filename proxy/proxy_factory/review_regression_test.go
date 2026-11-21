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

package proxy_factory

import (
	"context"
	"fmt"
	"testing"
)

import (
	"google.golang.org/protobuf/types/known/wrapperspb"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/rpcerror"
	"dubbo.apache.org/dubbo-go/v3/protocol/invocation"
	tri "dubbo.apache.org/dubbo-go/v3/protocol/triple/triple_protocol"
)

func TestInfoProxyPreservesTripleStatusAndMetadata(t *testing.T) {
	for _, code := range []tri.Code{tri.CodeUnavailable, tri.CodeDeadlineExceeded, tri.CodeBizError} {
		for _, wrapped := range []bool{false, true} {
			t.Run(fmt.Sprintf("code-%d/wrapped-%v", code, wrapped), func(t *testing.T) {
				source := tri.NewError(code, context.DeadlineExceeded)
				source.Meta().Set("X-Attribution", "preserve")
				detail, err := tri.NewErrorDetail(wrapperspb.String("detail"))
				if err != nil {
					t.Fatal(err)
				}
				source.AddDetail(detail)
				var returned error = source
				if wrapped {
					returned = fmt.Errorf("provider wrapper: %w", source)
				}
				info := &common.ServiceInfo{Methods: []common.MethodInfo{{Name: "Call", MethodFunc: func(context.Context, []any, any) (any, error) { return nil, returned }}}}
				invoker := newInfoInvoker(newURL("tri", "ReviewTripleErrors"), info, nil)
				res := invoker.Invoke(context.Background(), invocation.NewRPCInvocation("Call", nil, nil))
				if res.Error() != returned {
					t.Fatalf("existing Triple error replaced: %T", res.Error())
				}
				got := rpcerror.Classify(res.Error())
				if got.ProtocolCode != uint32(code) || got.Category == rpcerror.CategoryUnknown {
					t.Fatalf("double status/conflict: %+v", got)
				}
				if source.Meta().Get("X-Attribution") != "preserve" || len(source.Details()) != 1 || source.Details()[0] != detail {
					t.Fatal("Triple metadata/details lost")
				}
			})
		}
	}
}
