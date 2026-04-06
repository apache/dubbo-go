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

package generic

import (
	"context"
	"reflect"
	"sync"
)

import (
	hessian "github.com/apache/dubbo-go-hessian2"

	"github.com/dubbogo/gost/log/logger"

	perrors "github.com/pkg/errors"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/common/extension"
	"dubbo.apache.org/dubbo-go/v3/filter"
	"dubbo.apache.org/dubbo-go/v3/filter/generic/generalizer"
	"dubbo.apache.org/dubbo-go/v3/protocol/base"
	"dubbo.apache.org/dubbo-go/v3/protocol/invocation"
	"dubbo.apache.org/dubbo-go/v3/protocol/result"
)

var (
	serviceGenericOnce sync.Once
	serviceGeneric     *genericServiceFilter
)

func init() {
	extension.SetFilter(constant.GenericServiceFilterKey, newGenericServiceFilter)
}

// genericServiceFilter is for Server
type genericServiceFilter struct{}

func newGenericServiceFilter() filter.Filter {
	if serviceGeneric == nil {
		serviceGenericOnce.Do(func() {
			serviceGeneric = &genericServiceFilter{}
		})
	}
	return serviceGeneric
}
func (f *genericServiceFilter) Invoke(ctx context.Context, invoker base.Invoker, inv base.Invocation) result.Result {
	if !inv.IsGenericInvocation() {
		return invoker.Invoke(ctx, inv)
	}

	// get real invocation info from the generic invocation
	mtdName := inv.Arguments()[0].(string)
	// types are not required in dubbo-go, for dubbo-go client to dubbo-go server, types could be nil
	types := inv.Arguments()[1]
	args := inv.Arguments()[2].([]hessian.Object)

	logger.Debugf(`received a generic invocation:
		MethodName: %s,
		Types: %s,
		Args: %s
	`, mtdName, types, args)

	// get the type of the argument
	ivkURL := invoker.GetURL()
	svc := common.ServiceMap.GetServiceByServiceKey(ivkURL.Protocol, ivkURL.ServiceKey())
	method := svc.Method()[mtdName]
	if method == nil {
		return &result.RPCResult{
			Err: perrors.Errorf("\"%s\" method is not found, service key: %s", mtdName, ivkURL.ServiceKey()),
		}
	}

	argsType := method.ArgsType()
	if err := validateGenericArgs(method.IsVariadic(), len(argsType), len(args), mtdName); err != nil {
		return &result.RPCResult{Err: err}
	}

	// get generic info from attachments of invocation, the default value is "true"
	generic := inv.GetAttachmentWithDefaultValue(constant.GenericKey, constant.GenericSerializationDefault)
	// get generalizer according to value in the `generic`
	// realize
	newArgs, err := realizeInvocationArgs(getGeneralizer(generic), argsType, args, method.IsVariadic())
	if err != nil {
		return &result.RPCResult{Err: err}
	}

	newIvc := invocation.NewRPCInvocation(mtdName, newArgs, inv.Attachments())
	newIvc.SetReply(inv.Reply())

	return invoker.Invoke(ctx, newIvc)
}

// validateGenericArgs checks whether the number of generic invocation arguments
// matches the target method signature. Variadic methods accept argCount >= fixedParams.
func validateGenericArgs(isVariadic bool, argsTypeCount, argCount int, methodName string) error {
	if isVariadic {
		if argCount >= argsTypeCount-1 {
			return nil
		}
	} else if argCount == argsTypeCount {
		return nil
	}

	return perrors.Errorf("the number of args(=%d) is not matched with \"%s\" method", argCount, methodName)
}

// realizeInvocationArgs converts generic invocation arguments to concrete types.
// For variadic methods, fixed params are realized normally and trailing args are
// reshaped into a typed slice via realizeVariadicArg.
func realizeInvocationArgs(g generalizer.Generalizer, argsType []reflect.Type, args []hessian.Object, isVariadic bool) ([]any, error) {
	if !isVariadic {
		return realizeFixedArgs(g, args, argsType)
	}

	newArgs, err := realizeFixedArgs(g, args[:len(argsType)-1], argsType[:len(argsType)-1])
	if err != nil {
		return nil, err
	}

	variadicArg, err := realizeVariadicArg(g, args[len(argsType)-1:], argsType[len(argsType)-1])
	if err != nil {
		return nil, err
	}

	return append(newArgs, variadicArg), nil
}

// realizeFixedArgs converts each arg to its corresponding concrete type via Generalizer.Realize.
func realizeFixedArgs(g generalizer.Generalizer, args []hessian.Object, argsType []reflect.Type) ([]any, error) {
	newArgs := make([]any, len(argsType))
	for i := range argsType {
		newArg, err := g.Realize(args[i], argsType[i])
		if err != nil {
			return nil, perrors.Errorf("realization failed, %v", err)
		}
		newArgs[i] = newArg
	}

	return newArgs, nil
}

// realizeVariadicArg reshapes trailing generic args into a typed slice (e.g. []string)
// for the variadic parameter. Handles both discrete args and single packed array from Java.
func realizeVariadicArg(g generalizer.Generalizer, args []hessian.Object, variadicSliceType reflect.Type) (any, error) {
	variadicArgs := normalizeVariadicArgs(args)
	slice := reflect.MakeSlice(variadicSliceType, len(variadicArgs), len(variadicArgs))
	elemType := variadicSliceType.Elem()

	for i, arg := range variadicArgs {
		realized, err := g.Realize(arg, elemType)
		if err != nil {
			return nil, perrors.Errorf("realization of variadic arg[%d] failed: %v", i, err)
		}
		slice.Index(i).Set(reflect.ValueOf(realized))
	}

	return slice.Interface(), nil
}

func normalizeVariadicArgs(args []hessian.Object) []hessian.Object {
	if len(args) != 1 {
		return args
	}

	return unwrapToSlice(args[0])
}

// unwrapToSlice checks if obj is a slice/array type and returns its elements
// as []hessian.Object. If it's not a collection, returns it as a single-element slice.
func unwrapToSlice(obj hessian.Object) []hessian.Object {
	if obj == nil {
		return nil
	}
	v := reflect.ValueOf(obj)
	if v.Kind() == reflect.Slice || v.Kind() == reflect.Array {
		out := make([]hessian.Object, v.Len())
		for i := 0; i < v.Len(); i++ {
			out[i] = v.Index(i).Interface()
		}
		return out
	}
	// not a collection — treat as single variadic element
	return []hessian.Object{obj}
}

func (f *genericServiceFilter) OnResponse(_ context.Context, result result.Result, _ base.Invoker, inv base.Invocation) result.Result {
	if inv.IsGenericInvocation() && result.Result() != nil {
		// get generic info from attachments of invocation, the default value is "true"
		generic := inv.GetAttachmentWithDefaultValue(constant.GenericKey, constant.GenericSerializationDefault)
		// get generalizer according to value in the `generic`
		g := getGeneralizer(generic)

		obj, err := g.Generalize(result.Result())
		if err != nil {
			err = perrors.Errorf("generalizaion failed, %v", err)
			result.SetError(err)
			result.SetResult(nil)
			return result
		}
		result.SetResult(obj)
	}
	return result
}
