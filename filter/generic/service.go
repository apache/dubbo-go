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
	"errors"
	"reflect"
)

import (
	hessian "github.com/apache/dubbo-go-hessian2"
)

import (
	"dubbo.apache.org/dubbo-go/v3/filter/generic/generalizer"
	"dubbo.apache.org/dubbo-go/v3/protocol/base"
)

// GenericService uses for generic invoke for service call
type GenericService struct {
	// Invoke is the legacy generic call entry point. Use InvokeWithOptions when
	// per-call options such as response metadata targets are required.
	Invoke func(ctx context.Context, methodName string, types []string, args []hessian.Object) (any, error) `dubbo:"$invoke"`
	// InvokeWithOptions is the options-aware generic call entry point.
	InvokeWithOptions func(ctx context.Context, methodName string, types []string, args []hessian.Object, opts ...base.CallOption) (any, error) `dubbo:"$invoke"`
	referenceStr      string
}

// NewGenericService returns a GenericService instance
func NewGenericService(referenceStr string) *GenericService {
	return &GenericService{referenceStr: referenceStr}
}

// Reference gets referenceStr from GenericService
func (s *GenericService) Reference() string {
	return s.referenceStr
}

// InvokeWithType invokes the remote method and deserializes the result into the reply struct.
// The reply parameter must be a non-nil pointer to the target type. Optional call options
// are forwarded through the options-aware generic call path.
//
// Note: This method uses MapGeneralizer for deserialization, which means it only supports
// the default map-based generic serialization (generic=true). If you are using other
// serialization types like Gson or Protobuf-JSON, use the Invoke method directly and
// handle deserialization manually.
//
// Example usage:
//
//	var user User
//	err := genericService.InvokeWithType(ctx, "getUser", []string{"java.lang.String"}, []hessian.Object{"123"}, &user)
//	if err != nil {
//	    return err
//	}
//	fmt.Println(user.Name, user.Age)
func (s *GenericService) InvokeWithType(ctx context.Context, methodName string, types []string, args []hessian.Object, reply any, opts ...base.CallOption) error {
	if len(opts) > 0 {
		return s.invokeWithTypeOptions(ctx, methodName, types, args, reply, opts...)
	}
	return s.invokeWithType(ctx, methodName, types, args, reply, func(ctx context.Context, methodName string, types []string, args []hessian.Object) (any, error) {
		return s.Invoke(ctx, methodName, types, args)
	})
}

// InvokeWithTypeOptions is the explicit options-named form of InvokeWithType.
// The reply parameter must be a non-nil pointer. It returns an initialization error
// if the service was not implemented with the options-aware proxy path.
func (s *GenericService) InvokeWithTypeOptions(ctx context.Context, methodName string, types []string, args []hessian.Object, reply any, opts ...base.CallOption) error {
	return s.invokeWithTypeOptions(ctx, methodName, types, args, reply, opts...)
}

func (s *GenericService) invokeWithTypeOptions(ctx context.Context, methodName string, types []string, args []hessian.Object, reply any, opts ...base.CallOption) error {
	if s.InvokeWithOptions == nil {
		return errors.New("generic invoke with options is not initialized")
	}
	return s.invokeWithType(ctx, methodName, types, args, reply, func(ctx context.Context, methodName string, types []string, args []hessian.Object) (any, error) {
		return s.InvokeWithOptions(ctx, methodName, types, args, opts...)
	})
}

func (s *GenericService) invokeWithType(ctx context.Context, methodName string, types []string, args []hessian.Object, reply any, invoke func(context.Context, string, []string, []hessian.Object) (any, error)) error {
	replyValue, err := validateReplyPointer(reply)
	if err != nil {
		return err
	}

	result, err := invoke(ctx, methodName, types, args)
	if err != nil {
		return err
	}
	if result == nil {
		return nil
	}

	g := generalizer.GetMapGeneralizer()
	realized, err := realizeResult(result, replyValue.Elem().Type(), g)
	if err != nil {
		return err
	}
	if realized != nil {
		replyValue.Elem().Set(reflect.ValueOf(realized))
	}
	return nil
}
