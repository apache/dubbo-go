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
	"reflect"
	"strings"
)

import (
	perrors "github.com/pkg/errors"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/filter/generic/generalizer"
	"dubbo.apache.org/dubbo-go/v3/protocol/base"
)

// isCallingToGenericService check if it calls to a generic service
func isCallingToGenericService(invoker base.Invoker, invocation base.Invocation) bool {
	return isGeneric(invoker.GetURL().GetParam(constant.GenericKey, "")) &&
		invocation.MethodName() != constant.Generic &&
		invocation.MethodName() != constant.GenericAsync
}

// isMakingAGenericCall check if it is making a generic call to a generic service
func isMakingAGenericCall(invoker base.Invoker, invocation base.Invocation) bool {
	return isGeneric(invoker.GetURL().GetParam(constant.GenericKey, "")) &&
		(invocation.MethodName() == constant.Generic ||
			invocation.MethodName() == constant.GenericAsync) &&
		invocation.Arguments() != nil &&
		len(invocation.Arguments()) == 3
}

// isGeneric receives a generic field from url of invoker to determine whether the service is generic or not
func isGeneric(generic string) bool {
	_, err := getGeneralizer(generic)
	return err == nil
}

func isGenericDisabled(generic string) bool {
	return generic == "" || strings.EqualFold(generic, "false")
}

// getGeneralizer resolves a generic mode to its generalizer.
// Recommended modes are true, gson, bean, and protobuf-json. protobuf keeps
// the legacy Map/Hessian generic semantics used by Triple generic invocations.
func getGeneralizer(generic string) (generalizer.Generalizer, error) {
	switch {
	case strings.EqualFold(generic, constant.GenericSerializationDefault):
		return generalizer.GetMapGeneralizer(), nil
	case strings.EqualFold(generic, constant.GenericSerializationGson):
		return generalizer.GetGsonGeneralizer(), nil
	case strings.EqualFold(generic, constant.GenericSerializationProtobufJson):
		return generalizer.GetProtobufJsonGeneralizer(), nil
	case strings.EqualFold(generic, constant.GenericSerializationProtobuf):
		return generalizer.GetMapGeneralizer(), nil
	case strings.EqualFold(generic, constant.GenericSerializationBean):
		return generalizer.GetBeanGeneralizer(), nil
	default:
		return nil, perrors.Errorf("unsupported generic mode %q", generic)
	}
}

// resolveGeneralizer validates configured generic modes and selects a non-empty invocation mode when present.
// Invocation generic mode takes precedence over the URL mode; empty or false values do not override URL mode.
func resolveGeneralizer(configuredGeneric string, invocation base.Invocation) (string, generalizer.Generalizer, error) {
	if isGenericDisabled(configuredGeneric) {
		return "", nil, nil
	}

	var g generalizer.Generalizer
	var err error
	g, err = getGeneralizer(configuredGeneric)
	if err != nil {
		return "", nil, err
	}

	if invocationGeneric, ok := invocation.GetAttachment(constant.GenericKey); ok {
		if isGenericDisabled(invocationGeneric) {
			return configuredGeneric, g, nil
		}
		invocationGeneralizer, err := getGeneralizer(invocationGeneric)
		if err != nil {
			return "", nil, err
		}
		return invocationGeneric, invocationGeneralizer, nil
	}

	return configuredGeneric, g, nil
}

func resolveGenericInvocationGeneralizer(invocation base.Invocation) (string, generalizer.Generalizer, error) {
	generic := invocation.GetAttachmentWithDefaultValue(constant.GenericKey, constant.GenericSerializationDefault)
	if generic == "" {
		generic = constant.GenericSerializationDefault
	}
	g, err := getGeneralizer(generic)
	if err != nil {
		return "", nil, err
	}
	return generic, g, nil
}

// realizeResult deserializes the data into the target type using the provided generalizer.
// It returns the realized value and any error that occurred during deserialization.
//
// Parameters:
//   - data: the source data to deserialize (typically map[string]any or []any)
//   - targetType: the reflect.Type of the target struct
//   - g: the generalizer to use for deserialization
//
// Returns:
//   - the realized value matching targetType
//   - error if deserialization fails
func realizeResult(data any, targetType reflect.Type, g generalizer.Generalizer) (any, error) {
	if data == nil {
		return nil, nil
	}

	realized, err := g.Realize(data, targetType)
	if err != nil {
		return nil, perrors.Errorf("failed to deserialize result to %s: %v", targetType.String(), err)
	}

	return realized, nil
}

// validateReplyPointer checks if the reply is a valid non-nil pointer.
//
// Parameters:
//   - reply: the value to validate
//
// Returns:
//   - the reflect.Value of the reply
//   - error if validation fails
func validateReplyPointer(reply any) (reflect.Value, error) {
	if reply == nil {
		return reflect.Value{}, perrors.New("reply cannot be nil")
	}

	replyValue := reflect.ValueOf(reply)
	if replyValue.Kind() != reflect.Pointer {
		return reflect.Value{}, perrors.New("reply must be a pointer")
	}

	if replyValue.IsNil() {
		return reflect.Value{}, perrors.New("reply cannot be a nil pointer")
	}

	return replyValue, nil
}
