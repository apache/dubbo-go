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
func (s *GenericService) InvokeWithType(ctx context.Context, methodName string, types []string, args []hessian.Object, reply any) error {
	// Validate the reply pointer
	replyValue, err := validateReplyPointer(reply)
	if err != nil {
		return err
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
	realized, err := realizeResult(result, replyType, g)
	if err != nil {
		return err
	}

	// Set the realized value to reply
	replyValue.Elem().Set(reflect.ValueOf(realized))
	return nil
}
