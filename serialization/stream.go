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

package serialization

// StreamCodec creates exclusive state for each complete Dubbo message body.
// Independent Marshal calls must never be concatenated to emulate a stream.
type StreamCodec interface {
	Codec
	NewEncoder() (Encoder, error)
	NewDecoder([]byte) (Decoder, error)
}
type Encoder interface {
	Encode(any) error
	Finish() ([]byte, error)
	Close() error
}
type Decoder interface {
	// Decode fills a non-nil pointer with the next object. Protocol fields use
	// concrete targets (string, int32, map[string]any); business arguments use
	// *any during routing and concrete method argument pointers when needed;
	// response values use the caller's concrete reply pointer.
	Decode(any) error
	Close() error
}
