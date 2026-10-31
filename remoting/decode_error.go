/*
 * Licensed to the Apache Software Foundation (ASF) under one or more
 * contributor license agreements. See the NOTICE file distributed with
 * this work for additional information regarding copyright ownership.
 * The ASF licenses this file to You under the Apache License, Version 2.0
 * (the "License"); you may not use this file except in compliance with
 * the License. You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package remoting

// ResponseDecodeError retains a validated response ID after a complete frame's
// body failed to deserialize. It is local metadata, never a new wire status.
type ResponseDecodeError struct {
	RequestID int64
	Cause     error
}

func (e *ResponseDecodeError) Error() string { return e.Cause.Error() }
func (e *ResponseDecodeError) Unwrap() error { return e.Cause }

// ObserveResponseDecodeFailure records only for the original sending session.
// It deliberately does not remove the pending request or wake its waiter.
func ObserveResponseDecodeFailure(id int64, session any, err error) bool {
	p := GetPendingResponse(SequenceType(id))
	return p != nil && p.Observation.Record(session, err)
}
