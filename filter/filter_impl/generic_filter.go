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
)

import (
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/common/extension"
	"dubbo.apache.org/dubbo-go/v3/common/logger"
	"dubbo.apache.org/dubbo-go/v3/filter"
	"dubbo.apache.org/dubbo-go/v3/protocol"
	invocation2 "dubbo.apache.org/dubbo-go/v3/protocol/invocation"
)

const (
	// GENERIC
	// generic module name
	GENERIC = "generic"
)

func init() {
	extension.SetFilter(GENERIC, GetGenericFilter)
}

//  when do a generic invoke, struct need to be map

// nolint
type GenericFilter struct{}

// Invoke turns the parameters to map for generic method
func (ef *GenericFilter) Invoke(ctx context.Context, invoker protocol.Invoker, invocation protocol.Invocation) protocol.Result {
	if invocation.MethodName() != constant.GENERIC || len(invocation.Arguments()) != 3 {
		return invoker.Invoke(ctx, invocation)
	}
	logger.Debugf("[generic filter] Attachments %+v", invocation.Attachments())

	genericKey := invocation.AttachmentsByKey(constant.GENERIC_KEY, constant.GENERIC_SERIALIZATION_DEFAULT)
	processor := extension.GetGenericProcessor(genericKey)
	if processor == nil {
		logger.Errorf("[Generic Filter] Don't support this generic: %s", genericKey)
		return &protocol.RPCResult{}
	}
	newArgs, err := processor.Serialize(invocation.Arguments())
	if err != nil {
		logger.Errorf("[Generic Filter] Serialization error", genericKey)
		return &protocol.RPCResult{}
	}
	newInvocation := invocation2.NewRPCInvocation(invocation.MethodName(), newArgs, invocation.Attachments())
	newInvocation.SetReply(invocation.Reply())
	return invoker.Invoke(ctx, newInvocation)
}

// OnResponse dummy process, returns the result directly
func (ef *GenericFilter) OnResponse(_ context.Context, result protocol.Result, _ protocol.Invoker,
	_ protocol.Invocation) protocol.Result {
	return result
}

// GetGenericFilter returns GenericFilter instance
func GetGenericFilter() filter.Filter {
	return &GenericFilter{}
}
