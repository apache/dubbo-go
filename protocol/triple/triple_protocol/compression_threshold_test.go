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

package triple_protocol

import (
	"testing"
)

// TestCompressionThresholdDefaultsUnchanged verifies that the server default of
// 512 bytes applies only to server responses: the shared low-level options and
// client request compression keep their own defaults, and explicit thresholds
// are preserved.
func TestCompressionThresholdDefaultsUnchanged(t *testing.T) {
	for _, options := range [][]ClientOption{nil, {WithSendGzip()}} {
		config, err := newClientConfig("http://localhost/test.Threshold", options)
		if err != nil {
			t.Fatal(err)
		}
		if config.CompressMinBytes != 0 {
			t.Fatalf("client threshold changed: %d", config.CompressMinBytes)
		}
	}
	if config := newHandlerConfig("/test.Threshold/Echo", nil); config.CompressMinBytes != 0 {
		t.Fatalf("low-level handler threshold changed: %d", config.CompressMinBytes)
	}
	for _, minimum := range []int{0, 256, 512} {
		clientConfig, err := newClientConfig("http://localhost/test.Threshold", []ClientOption{WithCompressMinBytes(minimum)})
		if err != nil {
			t.Fatal(err)
		}
		handlerConfig := newHandlerConfig("/test.Threshold/Echo", []HandlerOption{WithCompressMinBytes(minimum)})
		if clientConfig.CompressMinBytes != minimum || handlerConfig.CompressMinBytes != minimum {
			t.Fatalf("explicit low-level threshold %d not preserved", minimum)
		}
	}
}
