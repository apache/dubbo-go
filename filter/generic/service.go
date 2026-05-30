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
)

import (
	"dubbo.apache.org/dubbo-go/v3/filter/generic/generalizer"
)

// GenericService uses for generic invoke for service call
type GenericService struct {
	Invoke       func(ctx context.Context, methodName string, types []string, args []hessian.Object) (any, error) `dubbo:"$invoke"`
	referenceStr string
	attachments  sync.Map
}

// GenericService2 keeps the legacy generic invoke surface for attachment access compatibility.
type GenericService2 struct {
	Invoke       func(ctx context.Context, methodName string, types []string, args []hessian.Object) (any, error) `dubbo:"$invoke"`
	referenceStr string
	attachments  sync.Map
}

// NewGenericService returns a GenericService instance
func NewGenericService(referenceStr string) *GenericService {
	return &GenericService{referenceStr: referenceStr}
}

// NewGenericService2 returns a GenericService2 instance.
func NewGenericService2(referenceStr string) *GenericService2 {
	return &GenericService2{referenceStr: referenceStr}
}

// Reference gets referenceStr from GenericService
func (s *GenericService) Reference() string {
	return s.referenceStr
}

// Reference gets referenceStr from GenericService2
func (s *GenericService2) Reference() string {
	return s.referenceStr
}

// SetResponseAttachments stores response attachments for the provided context.
func (s *GenericService) SetResponseAttachments(ctx context.Context, attachments map[string]any) {
	setResponseAttachments(&s.attachments, ctx, attachments)
}

// GetResponseAttachments returns response attachments of the provided context.
func (s *GenericService) GetResponseAttachments(ctx context.Context) map[string]any {
	return getResponseAttachments(&s.attachments, ctx)
}

// SetResponseAttachments stores response attachments for the provided context.
func (s *GenericService2) SetResponseAttachments(ctx context.Context, attachments map[string]any) {
	setResponseAttachments(&s.attachments, ctx, attachments)
}

// GetResponseAttachments returns response attachments of the provided context.
func (s *GenericService2) GetResponseAttachments(ctx context.Context) map[string]any {
	return getResponseAttachments(&s.attachments, ctx)
}

func setResponseAttachments(store *sync.Map, ctx context.Context, attachments map[string]any) {
	if ctx == nil {
		return
	}
	key := contextKey(ctx)
	if key == nil {
		return
	}
	store.Store(key, cloneAttachments(attachments))
}

func getResponseAttachments(store *sync.Map, ctx context.Context) map[string]any {
	if ctx == nil {
		return nil
	}
	key := contextKey(ctx)
	if key == nil {
		return nil
	}
	value, ok := store.Load(key)
	if !ok {
		return nil
	}
	attachments, _ := value.(map[string]any)
	return cloneAttachments(attachments)
}

func contextKey(ctx context.Context) any {
	value := reflect.ValueOf(ctx)
	if !value.IsValid() || value.Kind() != reflect.Ptr || value.IsNil() {
		return nil
	}
	return value.Pointer()
}

func cloneAttachments(attachments map[string]any) map[string]any {
	if attachments == nil {
		return nil
	}
	cloned := make(map[string]any, len(attachments))
	for k, v := range attachments {
		cloned[k] = v
	}
	return cloned
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
	if realized != nil {
		replyValue.Elem().Set(reflect.ValueOf(realized))
	}
	return nil
}
