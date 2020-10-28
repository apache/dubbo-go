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

// Result is a RPC result
type Result interface {
	// SetError sets error.
	SetError(error)
	// Error gets error.
	Error() error
	// SetResult sets invoker result.
	SetResult(interface{})
	// Result gets invoker result.
	Result() interface{}
	// SetAttachments replaces the existing attachments with the specified param.
	SetAttachments(map[string]interface{})
	// Attachments gets all attachments
	Attachments() map[string]interface{}

	// AddAttachment adds the specified map to existing attachments in this instance.
	AddAttachment(string, interface{})
	// Attachment gets attachment by key with default value.
	Attachment(string, interface{}) interface{}
}

/////////////////////////////
// Result Implement of RPC
/////////////////////////////

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
	return r.Attrs
}

// AddAttachment adds the specified map to existing attachments in this instance.
func (r *RPCResult) AddAttachment(key string, value interface{}) {
	r.Attrs[key] = value
}

// Attachment gets attachment by key with default value.
func (r *RPCResult) Attachment(key string, defaultValue interface{}) interface{} {
	v, ok := r.Attrs[key]
	if !ok {
		v = defaultValue
	}
	return v
}
