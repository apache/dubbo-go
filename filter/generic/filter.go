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

// Package generic provides generic invoke filter.
package generic

import (
	"context"
	"reflect"
	"sync"
)

import (
	hessian "github.com/apache/dubbo-go-hessian2"

	"github.com/dubbogo/gost/log/logger"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/common/extension"
	"dubbo.apache.org/dubbo-go/v3/filter"
	"dubbo.apache.org/dubbo-go/v3/protocol/base"
	"dubbo.apache.org/dubbo-go/v3/protocol/invocation"
	"dubbo.apache.org/dubbo-go/v3/protocol/result"
)

var (
	genericOnce sync.Once
	instance    *genericFilter
)

func init() {
	extension.SetFilter(constant.GenericFilterKey, newGenericFilter)
}

// genericFilter ensures the structs are converted to maps, this filter is for consumer
type genericFilter struct{}

func newGenericFilter() filter.Filter {
	if instance == nil {
		genericOnce.Do(func() {
			instance = &genericFilter{}
		})
	}
	return instance
}

// Invoke turns the parameters to map for generic method
func (f *genericFilter) Invoke(ctx context.Context, invoker base.Invoker, inv base.Invocation) result.Result {
	if isCallingToGenericService(invoker, inv) {

		mtdName := inv.MethodName()
		oldArgs := inv.Arguments()

		types := make([]string, 0, len(oldArgs))
		args := make([]hessian.Object, 0, len(oldArgs))

		// get generic info from attachments of invocation, the default value is "true"
		generic := inv.GetAttachmentWithDefaultValue(constant.GenericKey, constant.GenericSerializationDefault)
		// get generalizer according to value in the `generic`
		g := getGeneralizer(generic)

		for _, arg := range oldArgs {
			// use the default generalizer(MapGeneralizer)
			typ, err := g.GetType(arg)
			if err != nil {
				logger.Errorf("failed to get type, %v", err)
			}
			obj, err := g.Generalize(arg)
			if err != nil {
				logger.Errorf("generalization failed, %v", err)
				return invoker.Invoke(ctx, inv)
			}
			types = append(types, typ)
			args = append(args, obj)
		}

		// construct a new invocation for generic call
		newArgs := []any{
			mtdName,
			types,
			args,
		}

		// For Triple protocol non-IDL mode, we need to set parameterRawValues
		// The format is [param1, param2, ..., paramN, reply] where the last element is the reply placeholder
		// Triple invoker slices as: request = inRaw[0:len-1], reply = inRaw[len-1]
		// So for generic call, we need [methodName, types, args, reply] to get request = [methodName, types, args]
		reply := inv.Reply()
		parameterRawValues := []any{mtdName, types, args, reply}

		newIvc := invocation.NewRPCInvocationWithOptions(
			invocation.WithMethodName(constant.Generic),
			invocation.WithArguments(newArgs),
			invocation.WithParameterRawValues(parameterRawValues),
			invocation.WithAttachments(inv.Attachments()),
			invocation.WithReply(reply),
		)
		newIvc.Attachments()[constant.GenericKey] = invoker.GetURL().GetParam(constant.GenericKey, "")

		// Copy CallType attribute from original invocation for Triple protocol support
		// If not present, set default to CallUnary for generic calls
		if callType, ok := inv.GetAttribute(constant.CallTypeKey); ok {
			newIvc.SetAttribute(constant.CallTypeKey, callType)
		} else {
			newIvc.SetAttribute(constant.CallTypeKey, constant.CallUnary)
		}

		return invoker.Invoke(ctx, newIvc)
	} else if isMakingAGenericCall(invoker, inv) {
		// Arguments format: [methodName string, types []string, args []hessian.Object]
		oldArgs := inv.Arguments()
		reply := inv.Reply()

		// For Triple protocol non-IDL mode, we need to set parameterRawValues
		// parameterRawValues format: [methodName, types, args, reply]
		// Triple invoker slices as: request = inRaw[0:len-1], reply = inRaw[len-1]
		parameterRawValues := []any{oldArgs[0], oldArgs[1], oldArgs[2], reply}

		newIvc := invocation.NewRPCInvocationWithOptions(
			invocation.WithMethodName(inv.MethodName()),
			invocation.WithArguments(oldArgs),
			invocation.WithParameterRawValues(parameterRawValues),
			invocation.WithAttachments(inv.Attachments()),
			invocation.WithReply(reply),
		)
		newIvc.Attachments()[constant.GenericKey] = invoker.GetURL().GetParam(constant.GenericKey, "")

		// Set CallType for Triple protocol support
		if callType, ok := inv.GetAttribute(constant.CallTypeKey); ok {
			newIvc.SetAttribute(constant.CallTypeKey, callType)
		} else {
			newIvc.SetAttribute(constant.CallTypeKey, constant.CallUnary)
		}

		return invoker.Invoke(ctx, newIvc)
	}
	return invoker.Invoke(ctx, inv)
}

// OnResponse deserializes the map result to the target struct if reply is provided.
// If inv.Reply() is a non-nil pointer to a struct, the map result will be automatically
// deserialized into it using the appropriate generalizer.
func (f *genericFilter) OnResponse(_ context.Context, res result.Result, invoker base.Invoker,
	inv base.Invocation) result.Result {
	// Only process if this is a generic call and there's no error
	if res.Error() != nil {
		return res
	}

	// Check if this is a generic invocation
	if !isGeneric(invoker.GetURL().GetParam(constant.GenericKey, "")) {
		return res
	}

	// Get the reply from invocation
	reply := inv.Reply()
	if reply == nil {
		return res
	}

	// Check if reply is a valid pointer
	replyValue := reflect.ValueOf(reply)
	if replyValue.Kind() != reflect.Ptr || replyValue.IsNil() {
		return res
	}

	// Get the result data
	data := res.Result()
	if data == nil {
		return res
	}

	// Check if data is a map type that needs to be deserialized
	dataValue := reflect.ValueOf(data)
	if dataValue.Kind() != reflect.Map && dataValue.Kind() != reflect.Slice {
		// If data is not a map or slice, it's already a primitive type, no need to deserialize
		return res
	}

	// Get the element type that the pointer points to
	replyElemType := replyValue.Elem().Type()

	// Get the generalizer based on the generic serialization type
	generic := invoker.GetURL().GetParam(constant.GenericKey, constant.GenericSerializationDefault)
	g := getGeneralizer(generic)

	// Realize the map/slice to the target struct using shared helper
	realized, err := realizeResult(data, replyElemType, g)
	if err != nil {
		logger.Warnf("failed to deserialize generic result: %v", err)
		return res
	}

	// Set the realized value to reply
	if realized != nil {
		replyValue.Elem().Set(reflect.ValueOf(realized))
	}

	// Update the result with the deserialized reply
	res.SetResult(reply)

	return res
}
