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

package extension

import "dubbo.apache.org/dubbo-go/v3/common"

// PrepareCodecURL binds protocol codecs before reference/export side effects.
// Protocol implementations may opt into this error-returning initialization.
func PrepareCodecURL(url *common.URL, side string) error {
	p := GetProtocol(url.Protocol)
	if initializer, ok := p.(interface {
		PrepareCodec(*common.URL, string) error
	}); ok {
		return initializer.PrepareCodec(url, side)
	}
	return nil
}

// ReleaseCodecURL rolls back codec ownership when reference/export fails.
func ReleaseCodecURL(url *common.URL, side string) error {
	if initializer, ok := GetProtocol(url.Protocol).(interface {
		ReleaseCodec(*common.URL, string) error
	}); ok {
		return initializer.ReleaseCodec(url, side)
	}
	return nil
}
