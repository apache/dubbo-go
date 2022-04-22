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

package protocol

import (
	"context"
	"reflect"
)

// Invocation is a interface which is invocation for each remote method.
type Invocation interface {
	// MethodName gets invocation method name.
	MethodName() string
	// ActualMethodName gets actual invocation method name. It returns the method name been called if it's a generic call
	ActualMethodName() string
	// ParameterTypeNames gets invocation parameter type names.
	ParameterTypeNames() []string
	// ParameterTypes gets invocation parameter types.
	ParameterTypes() []reflect.Type
	// ParameterValues gets invocation parameter values.
	ParameterValues() []reflect.Value
	// Arguments gets arguments.
	Arguments() []interface{}
	// Reply gets response of request
	Reply() interface{}
	// Attachments gets all attachments

	// Invoker gets the invoker in current context.
	Invoker() Invoker
	// IsGenericInvocation gets if this is a generic invocation
	IsGenericInvocation() bool

	Attachments() map[string]interface{}
	SetAttachment(key string, value interface{})
	GetAttachment(key string) (string, bool)
	GetAttachmentInterface(string) interface{}
	GetAttachmentWithDefaultValue(key string, defaultValue string) string
	GetAttachmentAsContext() context.Context

	// Attributes firstly introduced on dubbo-java 2.7.6. It is
	// used in internal invocation, that is, it's not passed between
	// server and client.
	Attributes() map[string]interface{}
	SetAttribute(key string, value interface{})
	GetAttribute(key string) (interface{}, bool)
	GetAttributeWithDefaultValue(key string, defaultValue interface{}) interface{}
}
