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
)

import (
	hessian "github.com/apache/dubbo-go-hessian2"

	perrors "github.com/pkg/errors"
)

import (
	"dubbo.apache.org/dubbo-go/v3/filter/generic/generalizer"
)

// GenericService uses for generic invoke for service call
type GenericService struct {
	Invoke       func(ctx context.Context, methodName string, types []string, args []hessian.Object) (any, error) `dubbo:"$invoke"`
	referenceStr string
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
// The reply parameter must be a non-nil pointer to the target type.
//
// Example usage:
//
//	var user User
//	err := genericService.InvokeWithType(ctx, "getUser", []string{"java.lang.String"}, []hessian.Object{"123"}, &user)
//	if err != nil {
//	    return err
//	}
//	fmt.Println(user.Name, user.Age)
func (s *GenericService) InvokeWithType(ctx context.Context, methodName string, types []string, args []hessian.Object, reply any) error {
	if reply == nil {
		return perrors.New("reply cannot be nil")
	}

	replyValue := reflect.ValueOf(reply)
	if replyValue.Kind() != reflect.Ptr {
		return perrors.New("reply must be a pointer")
	}

	if replyValue.IsNil() {
		return perrors.New("reply cannot be a nil pointer")
	}

	// Call the underlying Invoke method
	result, err := s.Invoke(ctx, methodName, types, args)
	if err != nil {
		return err
	}

	if result == nil {
		return nil
	}

	// Get the element type that the pointer points to
	replyType := replyValue.Elem().Type()

	// Use MapGeneralizer to realize the map result to the target struct
	g := generalizer.GetMapGeneralizer()
	realized, err := g.Realize(result, replyType)
	if err != nil {
		return perrors.Errorf("failed to deserialize result to %s: %v", replyType.String(), err)
	}

	// Set the realized value to reply
	replyValue.Elem().Set(reflect.ValueOf(realized))
	return nil
}
