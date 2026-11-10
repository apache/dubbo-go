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

import (
	"encoding/json"
	"testing"
)

import (
	"github.com/stretchr/testify/require"

	"go.yaml.in/yaml/v4"
)

func TestTripleCompressionConfigRoundTrip(t *testing.T) {
	for _, format := range []struct {
		name      string
		marshal   func(any) ([]byte, error)
		unmarshal func([]byte, any) error
	}{
		{"json", json.Marshal, json.Unmarshal},
		{"yaml", yaml.Marshal, func(b []byte, v any) error { return yaml.Unmarshal(b, v) }},
	} {
		for _, value := range []string{"absent", "null", "0", "512"} {
			t.Run(format.name+"/"+value, func(t *testing.T) {
				input := "{}"
				if value != "absent" {
					input = `{"server-compress-min-bytes":` + value + `}`
					if format.name == "yaml" {
						input = "server-compress-min-bytes: " + value + "\n"
					}
				}
				var config TripleConfig
				require.NoError(t, format.unmarshal([]byte(input), &config))
				if value == "absent" || value == "null" {
					require.Nil(t, config.ServerCompressMinBytes)
				} else {
					require.NotNil(t, config.ServerCompressMinBytes)
					want := 0
					if value == "512" {
						want = 512
					}
					require.Equal(t, want, *config.ServerCompressMinBytes)
				}
				clone := config.Clone()
				require.Equal(t, config.ServerCompressMinBytes, clone.ServerCompressMinBytes)
				encoded, err := format.marshal(&config)
				require.NoError(t, err)
				var roundTrip TripleConfig
				require.NoError(t, format.unmarshal(encoded, &roundTrip))
				require.Equal(t, config.ServerCompressMinBytes, roundTrip.ServerCompressMinBytes)
				if clone.ServerCompressMinBytes != nil {
					*clone.ServerCompressMinBytes = 123
					require.NotEqual(t, *clone.ServerCompressMinBytes, *config.ServerCompressMinBytes)
				}
			})
		}
	}
	require.Nil(t, DefaultTripleConfig().ServerCompressMinBytes)
	require.Nil(t, DefaultTripleConfig().Clone().ServerCompressMinBytes)
}
