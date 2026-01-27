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
	"github.com/dubbogo/gost/log/logger"

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
	return strings.EqualFold(generic, constant.GenericSerializationDefault) ||
		strings.EqualFold(generic, constant.GenericSerializationGson) ||
		strings.EqualFold(generic, constant.GenericSerializationProtobufJson)
}

func getGeneralizer(generic string) (g generalizer.Generalizer) {
	switch {
	case strings.EqualFold(generic, constant.GenericSerializationDefault):
		g = generalizer.GetMapGeneralizer()
	case strings.EqualFold(generic, constant.GenericSerializationGson):
		g = generalizer.GetGsonGeneralizer()
	case strings.EqualFold(generic, constant.GenericSerializationProtobufJson):
		g = generalizer.GetProtobufJsonGeneralizer()
	default:
		logger.Debugf("\"%s\" is not supported, use the default generalizer(MapGeneralizer)", generic)
		g = generalizer.GetMapGeneralizer()
	}
	return
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
	if replyValue.Kind() != reflect.Ptr {
		return reflect.Value{}, perrors.New("reply must be a pointer")
	}

	if replyValue.IsNil() {
		return reflect.Value{}, perrors.New("reply cannot be a nil pointer")
	}

	return replyValue, nil
}
