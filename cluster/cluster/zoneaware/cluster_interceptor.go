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

package zoneaware

import (
	"context"
)

import (
	clusterpkg "dubbo.apache.org/dubbo-go/v3/cluster/cluster"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/protocol"
)

type interceptor struct{}

func (z *interceptor) Invoke(ctx context.Context, invoker protocol.Invoker, invocation protocol.Invocation) protocol.Result {
	key := constant.RegistryKey + "." + constant.RegistryZoneForceKey
	force := ctx.Value(key)

	if force != nil {
		switch value := force.(type) {
		case bool:
			if value {
				invocation.SetAttachment(key, "true")
			}
		case string:
			if "true" == value {
				invocation.SetAttachment(key, "true")
			}
		default:
			// ignore
		}
	}

	return invoker.Invoke(ctx, invocation)
}

func newInterceptor() clusterpkg.Interceptor {
	return &interceptor{}
}
