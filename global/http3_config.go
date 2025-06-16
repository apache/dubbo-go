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

package global

type Http3Config struct {
	// Whether to enable HTTP/3 support.
	// The default value is false.
	Enable bool `yaml:"enable" json:"enable,omitempty"`
	// TODO: add more params about http3

	// TODO: negotiation implementation
	// ref: https://quic-go.net/docs/http3/server/#advertising-http3-via-alt-svc
	//
	// Whether to enable HTTP/3 negotiation.
	// If set to false, HTTP/2 alt-svc negotiation will be skipped,
	// enabling HTTP/3 but disabling HTTP/2 on the consumer side.
	// negotiation bool
}

func DefaultHttp3Config() *Http3Config {
	return &Http3Config{}
}

// Clone a new TripleConfig
func (t *Http3Config) Clone() *Http3Config {
	if t == nil {
		return nil
	}

	return &Http3Config{
		Enable: t.Enable,
	}
}
