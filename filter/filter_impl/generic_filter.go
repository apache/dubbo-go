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
	"strings"
)

import (
	hessian "github.com/apache/dubbo-go-hessian2"
)

import (
	"github.com/apache/dubbo-go/common/constant"
	"github.com/apache/dubbo-go/common/extension"
	"github.com/apache/dubbo-go/common/logger"
	"github.com/apache/dubbo-go/filter"
	"github.com/apache/dubbo-go/protocol"
	invocation2 "github.com/apache/dubbo-go/protocol/invocation"
)

const (
	// GENERIC
	//generic module name
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
	if isCallingToGenericService(invoker, invocation) {

		mtdname := invocation.MethodName()
		oldargs := invocation.Arguments()

		types := make([]string, 0, len(oldargs))
		args := make([]hessian.Object, 0, len(oldargs))

		// get generic info from attachments of invocation, the default value is "true"
		//generic := invocation.AttachmentsByKey(constant.GENERIC_KEY, GENERIC_SERIALIZATION_DEFAULT)
		// get generalizer according to value in the `generic`
		g := GetMapGeneralizer()

		for _, arg := range oldargs {
			// use the default generalizer(MapGeneralizer)
			typ, err := g.GetType(arg)
			if err != nil {
				logger.Errorf("failed to get type, %v", err)
			}
			obj, err := g.Generalize(arg)
			if err != nil {
				logger.Errorf("generalization failed, %v", err)
				return invoker.Invoke(ctx, invocation)
			}
			types = append(types, typ)
			args = append(args, obj)
		}

		// construct a new invocation for generic call
		newargs := []interface{}{
			mtdname,
			types,
			args,
		}
		newivc := invocation2.NewRPCInvocation(constant.GENERIC, newargs, invocation.Attachments())
		newivc.SetReply(invocation.Reply())
		newivc.Attachments()[constant.GENERIC_KEY] = invoker.GetURL().GetParam(constant.GENERIC_KEY, "")

		return invoker.Invoke(ctx, newivc)
	} else if isMakingAGenericCall(invoker, invocation) {
		invocation.Attachments()[constant.GENERIC_KEY] = invoker.GetURL().GetParam(constant.GENERIC_KEY, "")
	}
	return invoker.Invoke(ctx, invocation)
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

// isCallingToGenericService check if it calls to a generic service
func isCallingToGenericService(invoker protocol.Invoker, invocation protocol.Invocation) bool {
	return isGeneric(invoker.GetURL().GetParam(constant.GENERIC_KEY, "")) &&
		invocation.MethodName() != constant.GENERIC &&
		invocation.MethodName() != constant.GENERIC_ASYNC
}

// isMakingAGenericCall check if it is making a generic call to a generic service
func isMakingAGenericCall(invoker protocol.Invoker, invocation protocol.Invocation) bool {
	return isGeneric(invoker.GetURL().GetParam(constant.GENERIC_KEY, "")) &&
		(invocation.MethodName() == constant.GENERIC || invocation.MethodName() == constant.GENERIC_ASYNC) &&
		invocation.Arguments() != nil &&
		len(invocation.Arguments()) == 3
}

// isGeneric receives a generic field from url of invoker to determine whether the service is generic or not
func isGeneric(generic string) bool {
	lowerGeneric := strings.ToLower(generic)
	return lowerGeneric == GENERIC_SERIALIZATION_DEFAULT
}
