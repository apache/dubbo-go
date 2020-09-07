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
	"reflect"
)

// Invocation is a invocation for each remote method.
type Invocation interface {
	// MethodName gets invocation method name.
	MethodName() string
	// ParameterTypes gets invocation parameter types.
	ParameterTypes() []reflect.Type
	// ParameterValues gets invocation parameter values.
	ParameterValues() []reflect.Value
	// Arguments gets arguments.
	Arguments() []interface{}
	// Reply gets response of request
	Reply() interface{}
	// Attachments gets all attachments
	Attachments() map[string]interface{}
	// AttachmentsByKey gets attachment by key , if nil then return default value. （It will be deprecated in the future）
	AttachmentsByKey(string, string) string
	Attachment(string) interface{}
	// Attributes refers to dubbo 2.7.6.  It is different from attachment. It is used in internal process.
	Attributes() map[string]interface{}
	// AttributeByKey gets attribute by key , if nil then return default value
	AttributeByKey(string, interface{}) interface{}
	// SetAttachments sets attribute by @key and @value.
	SetAttachments(key string, value interface{})
	// Invoker gets the invoker in current context.
	Invoker() Invoker
}
