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

package triple

import (
	"strconv"
	"testing"
)

import (
	"github.com/stretchr/testify/require"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
)

func TestResolveHandlerCompressMinBytesValidation(t *testing.T) {
	for _, tc := range []struct {
		name       string
		configured int
		override   string
		wantError  bool
	}{
		{"zero rollback", 0, "", false},
		{"configured threshold", 1024, "", false},
		{"explicit service zero", 1024, "0", false},
		{"negative global", -1, "", true},
		{"negative service", 1024, "-1", true},
		{"malformed service", 1024, "1KiB", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			config := NewOptions(WithServerCompressMinBytes(tc.configured)).Triple
			url := common.NewURLWithOptions(common.WithAttribute(constant.TripleConfigKey, config))
			if tc.override != "" {
				url.SetParam(ServerCompressMinBytesParam, tc.override)
			}
			_, err := resolveHandlerOptions(url)
			if tc.wantError {
				require.ErrorContains(t, err, "expected a non-negative integer")
			} else {
				require.NoError(t, err)
			}
			clone := config.Clone()
			require.Equal(t, strconv.Itoa(tc.configured), strconv.Itoa(*clone.ServerCompressMinBytes))
			*clone.ServerCompressMinBytes = 256
			require.Equal(t, tc.configured, *config.ServerCompressMinBytes, "clone must not alias threshold pointer")
		})
	}
}
