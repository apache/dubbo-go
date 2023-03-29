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
	"fmt"
)

// Result is a interface that represents RPC result
//
// SetError method sets error.
//
// Error method gets error.
//
// SetResult method sets invoker result.
//
// Result method gets invoker result.
//
// SetAttachments method replaces the existing attachments with the specified param.
//
// # Attachments method gets all attachments
//
// AddAttachment method adds the specified map to existing attachments in this instance.
//
// Attachment method gets attachment by key with default value.
type Result interface {
	SetError(error)
	Error() error
	SetResult(interface{})
	Result() interface{}
	SetAttachments(map[string]interface{})
	Attachments() map[string]interface{}
	AddAttachment(string, interface{})
	Attachment(string, interface{}) interface{}
}

var _ Result = (*RPCResult)(nil)

// RPCResult is default RPC result.
type RPCResult struct {
	Attrs map[string]interface{}
	Err   error
	Rest  interface{}
}

// SetError sets error.
func (r *RPCResult) SetError(err error) {
	r.Err = err
}

// Error gets error.
func (r *RPCResult) Error() error {
	return r.Err
}

// SetResult sets invoker result.
func (r *RPCResult) SetResult(rest interface{}) {
	r.Rest = rest
}

// Result gets invoker result.
func (r *RPCResult) Result() interface{} {
	return r.Rest
}

// SetAttachments replaces the existing attachments with the specified param.
func (r *RPCResult) SetAttachments(attr map[string]interface{}) {
	r.Attrs = attr
}

// Attachments gets all attachments
func (r *RPCResult) Attachments() map[string]interface{} {
	if r.Attrs == nil {
		r.Attrs = make(map[string]interface{})
	}
	return r.Attrs
}

// AddAttachment adds the specified map to existing attachments in this instance.
func (r *RPCResult) AddAttachment(key string, value interface{}) {
	if r.Attrs == nil {
		r.Attrs = make(map[string]interface{})
	}
	r.Attrs[key] = value
}

// Attachment gets attachment by key with default value.
func (r *RPCResult) Attachment(key string, defaultValue interface{}) interface{} {
	if r.Attrs == nil {
		r.Attrs = make(map[string]interface{})
		return nil
	}
	v, ok := r.Attrs[key]
	if !ok {
		v = defaultValue
	}
	return v
}

func (r *RPCResult) String() string {
	return fmt.Sprintf("&RPCResult{Rest: %v, Attrs: %v, Err: %v}", r.Rest, r.Attrs, r.Err)
}
