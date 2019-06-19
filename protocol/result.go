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
	SetError(error)
	Error() error
	SetResult(interface{})
	Result() interface{}
	SetAttachments(map[string]string)
	Attachments() map[string]string
	AddAttachment(string, string)
	Attachment(string, string) string
}

/////////////////////////////
// Result Impletment of RPC
/////////////////////////////

type RPCResult struct {
	Attrs map[string]string
	Err   error
	Rest  interface{}
}

func (r *RPCResult) SetError(err error) {
	r.Err = err
}

func (r *RPCResult) Error() error {
	return r.Err
}

func (r *RPCResult) SetResult(rest interface{}) {
	r.Rest = rest
}

func (r *RPCResult) Result() interface{} {
	return r.Rest
}

func (r *RPCResult) SetAttachments(attr map[string]string) {
	r.Attrs = attr
}

func (r *RPCResult) Attachments() map[string]string {
	return r.Attrs
}

func (r *RPCResult) AddAttachment(key, value string) {
	r.Attrs[key] = value
}

func (r *RPCResult) Attachment(key, defaultValue string) string {
	v, ok := r.Attrs[key]
	if !ok {
		v = defaultValue
	}
	return v
}
