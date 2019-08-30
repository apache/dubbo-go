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

type Result interface {
	SetError(error) Result
	Error() error
	SetResult(interface{}) Result
	Result() interface{}
	SetAttachments(map[string]string) Result
	Attachments() map[string]string
	AddAttachment(string, string) Result
	Attachment(string, string) string
}

/////////////////////////////
// Result Impletment of RPC
/////////////////////////////

type RPCResult struct {
	attrs map[string]string
	err   error
	rest  interface{}
}

func NewErrorRpcResult(err error) *RPCResult {
	result := &RPCResult{}
	result.err = err
	return result
}

func NewRpcResult(rest interface{}) *RPCResult {
	result := &RPCResult{}
	result.rest = rest
	return result
}

func (r *RPCResult) SetError(err error) Result {
	r.err = err
	return r
}

func (r *RPCResult) Error() error {
	return r.err
}

func (r *RPCResult) SetResult(rest interface{}) Result {
	r.rest = rest
	return r
}

func (r *RPCResult) Result() interface{} {
	return r.rest
}

func (r *RPCResult) SetAttachments(attr map[string]string) Result {
	r.attrs = attr
	return r
}

func (r *RPCResult) Attachments() map[string]string {
	return r.attrs
}

func (r *RPCResult) AddAttachment(key, value string) Result {
	r.attrs[key] = value
	return r
}

func (r *RPCResult) Attachment(key, defaultValue string) string {
	v, ok := r.attrs[key]
	if !ok {
		v = defaultValue
	}
	return v
}
