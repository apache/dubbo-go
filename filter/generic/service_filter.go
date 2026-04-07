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
	"strings"
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
	dubboHessian "dubbo.apache.org/dubbo-go/v3/protocol/dubbo/hessian2"
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
	newArgs, err := realizeInvocationArgs(getGeneralizer(generic), argsType, args, method.IsVariadic(), types)
	if err != nil {
		return &result.RPCResult{Err: err}
	}

	newIvc := invocation.NewRPCInvocation(mtdName, newArgs, inv.Attachments())
	newIvc.SetReply(inv.Reply())
	if method.IsVariadic() {
		newIvc.SetAttribute(constant.GenericVariadicCallSliceKey, true)
	}

	return invoker.Invoke(ctx, newIvc)
}

// validateGenericArgs checks the generic arg count against the method signature.
// Variadic methods accept any count >= the fixed parameter count.
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

// realizeInvocationArgs realizes generic args and packs a variadic tail into the declared slice type.
func realizeInvocationArgs(g generalizer.Generalizer, argsType []reflect.Type, args []hessian.Object, isVariadic bool, types any) ([]any, error) {
	if !isVariadic {
		return realizeFixedArgs(g, args, argsType)
	}

	newArgs, err := realizeFixedArgs(g, args[:len(argsType)-1], argsType[:len(argsType)-1])
	if err != nil {
		return nil, err
	}

	variadicArg, err := realizeVariadicArg(g, args[len(argsType)-1:], argsType[len(argsType)-1], variadicTypeName(types))
	if err != nil {
		return nil, err
	}

	return append(newArgs, variadicArg), nil
}

// realizeFixedArgs realizes non-variadic parameters one by one.
func realizeFixedArgs(g generalizer.Generalizer, args []hessian.Object, argsType []reflect.Type) ([]any, error) {
	newArgs := make([]any, len(argsType))
	for i := range argsType {
		newArg, err := g.Realize(args[i], argsType[i])
		if err != nil {
			return nil, perrors.Errorf("realization of arg[%d] failed: %v", i, err)
		}
		newArgs[i] = newArg
	}

	return newArgs, nil
}

// realizeVariadicArg realizes the variadic tail into the declared slice type.
// It unwraps a single arg only when its declared generic type matches the variadic slice.
func realizeVariadicArg(g generalizer.Generalizer, args []hessian.Object, variadicSliceType reflect.Type, variadicType string) (any, error) {
	variadicArgs := normalizeVariadicArgs(args, variadicSliceType, variadicType)
	slice := reflect.MakeSlice(variadicSliceType, len(variadicArgs), len(variadicArgs))
	elemType := variadicSliceType.Elem()

	for i, arg := range variadicArgs {
		realized, err := g.Realize(arg, elemType)
		if err != nil {
			return nil, perrors.Errorf("realization of variadic arg[%d] failed: %v", i, err)
		}
		realizedValue, err := assignableValue(realized, elemType)
		if err != nil {
			return nil, perrors.Errorf("realization of variadic arg[%d] failed: %v", i, err)
		}
		slice.Index(i).Set(realizedValue)
	}

	return slice.Interface(), nil
}

// assignableValue fits a realized value into the target type without panicking on Set.
func assignableValue(value any, targetType reflect.Type) (reflect.Value, error) {
	if value == nil {
		if canBeNil(targetType) {
			return reflect.Zero(targetType), nil
		}
		return reflect.Value{}, perrors.Errorf("nil is not assignable to %s", targetType)
	}

	realizedValue := reflect.ValueOf(value)
	if realizedValue.Type().AssignableTo(targetType) {
		return realizedValue, nil
	}
	if realizedValue.Type().ConvertibleTo(targetType) {
		return realizedValue.Convert(targetType), nil
	}

	return reflect.Value{}, perrors.Errorf("type %s is not assignable to %s", realizedValue.Type(), targetType)
}

func canBeNil(typ reflect.Type) bool {
	switch typ.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Ptr, reflect.Slice:
		return true
	default:
		return false
	}
}

// variadicTypeName returns the last generic type name when the caller provides $invoke type metadata.
func variadicTypeName(types any) string {
	switch typeNames := types.(type) {
	case []string:
		if len(typeNames) == 0 {
			return ""
		}
		return typeNames[len(typeNames)-1]
	case []any:
		if len(typeNames) == 0 {
			return ""
		}
		if typeName, ok := typeNames[len(typeNames)-1].(string); ok {
			return typeName
		}
	}

	return ""
}

// normalizeVariadicArgs unwraps one packed array only when the generic type says it
// is the variadic slice itself; otherwise the single arg stays as one variadic value.
func normalizeVariadicArgs(args []hessian.Object, variadicSliceType reflect.Type, variadicType string) []hessian.Object {
	if len(args) != 1 {
		return args
	}
	if !shouldUnwrapPackedVariadicArg(variadicType, variadicSliceType) {
		if variadicType != "" {
			return args
		}
		return normalizeVariadicArgsWithoutType(args[0], variadicSliceType)
	}
	if args[0] == nil {
		return nil
	}

	return unwrapToSlice(args[0])
}

// normalizeVariadicArgsWithoutType keeps dubbo-go compatibility when generic callers omit `types`.
// A real single variadic value stays packed as one element, while a packed tail slice is unwrapped once.
func normalizeVariadicArgsWithoutType(arg hessian.Object, variadicSliceType reflect.Type) []hessian.Object {
	if arg == nil {
		return nil
	}

	v := reflect.ValueOf(arg)
	if v.Kind() != reflect.Slice && v.Kind() != reflect.Array {
		return []hessian.Object{arg}
	}

	elemType := variadicSliceType.Elem()
	argType := v.Type()
	if elemType.Kind() != reflect.Interface && (argType.AssignableTo(elemType) || argType.ConvertibleTo(elemType)) {
		return []hessian.Object{arg}
	}

	return unwrapToSlice(arg)
}

// shouldUnwrapPackedVariadicArg matches the declared variadic slice against the
// generic tail type, including Java names and JVM array descriptors.
func shouldUnwrapPackedVariadicArg(variadicType string, variadicSliceType reflect.Type) bool {
	if variadicType == "" {
		return false
	}

	for _, typeName := range javaTypeNamesForType(variadicSliceType) {
		if variadicType == typeName {
			return true
		}
	}

	elemType := variadicSliceType.Elem()
	if elemType.Kind() == reflect.Interface && (variadicType == "[Ljava.lang.Object;" || variadicType == "java.lang.Object[]") {
		return true
	}

	return false
}

// javaTypeNamesForType returns the generic type spellings we accept for the variadic slice.
func javaTypeNamesForType(typ reflect.Type) []string {
	zero := reflect.Zero(typ)
	if !zero.IsValid() {
		return nil
	}

	names := make([]string, 0, 2)
	if name, err := dubboHessian.GetJavaName(zero.Interface()); err == nil && name != "" {
		names = append(names, name)
	}
	if desc := dubboHessian.GetClassDesc(zero.Interface()); desc != "" && desc != "V" {
		names = appendUniqueString(names, desc)
	}
	if desc := jvmArrayDescriptorForType(typ); desc != "" {
		names = appendUniqueString(names, desc)
	}

	return names
}

// jvmArrayDescriptorForType builds descriptors like [B, [[B or [[Ljava.lang.String;.
func jvmArrayDescriptorForType(typ reflect.Type) string {
	if typ.Kind() != reflect.Slice && typ.Kind() != reflect.Array {
		return ""
	}

	depth := 0
	for typ.Kind() == reflect.Slice || typ.Kind() == reflect.Array {
		depth++
		typ = typ.Elem()
	}

	leaf := jvmLeafDescriptorForType(typ)
	if leaf == "" {
		return ""
	}

	return strings.Repeat("[", depth) + leaf
}

func jvmLeafDescriptorForType(typ reflect.Type) string {
	switch typ.Kind() {
	case reflect.Bool:
		return "Z"
	case reflect.Int8, reflect.Uint8:
		return "B"
	case reflect.Int16:
		return "S"
	case reflect.Uint16:
		return "C"
	case reflect.Int, reflect.Int64:
		return "J"
	case reflect.Int32:
		return "I"
	case reflect.Float32:
		return "F"
	case reflect.Float64:
		return "D"
	case reflect.String:
		return "Ljava.lang.String;"
	case reflect.Interface:
		return "Ljava.lang.Object;"
	case reflect.Map:
		return "Ljava.util.Map;"
	case reflect.Struct:
		if typ.PkgPath() == "time" && typ.Name() == "Time" {
			return "Ljava.util.Date;"
		}
		return "Ljava.lang.Object;"
	default:
		zero := reflect.New(typ).Elem().Interface()
		desc := dubboHessian.GetClassDesc(zero)
		switch desc {
		case "", "V", "java.util.List":
			return ""
		case "java.lang.String":
			return "Ljava.lang.String;"
		case "java.lang.Object":
			return "Ljava.lang.Object;"
		case "java.util.Date":
			return "Ljava.util.Date;"
		case "java.util.Map":
			return "Ljava.util.Map;"
		}
		if len(desc) == 1 {
			return desc
		}
		if strings.HasPrefix(desc, "L") && strings.HasSuffix(desc, ";") {
			return desc
		}
		if strings.Contains(desc, ".") {
			return "L" + strings.ReplaceAll(desc, ".", "/") + ";"
		}
		return ""
	}
}

func appendUniqueString(values []string, value string) []string {
	for _, existing := range values {
		if existing == value {
			return values
		}
	}
	return append(values, value)
}

// unwrapToSlice returns slice/array elements, or keeps obj as one variadic element.
func unwrapToSlice(obj hessian.Object) []hessian.Object {
	if obj == nil {
		return []hessian.Object{nil}
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
