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
	"github.com/stretchr/testify/require"
	"go.yaml.in/yaml/v4"
	"testing"
)

func TestAcceptedSerializationsClonePresence(t *testing.T) {
	for _, input := range [][]string{nil, {}, {"hessian2"}} {
		original := &ReferenceConfig{AcceptedSerializations: input}
		clone := original.Clone()
		if (input == nil) != (clone.AcceptedSerializations == nil) {
			t.Fatal("clone changed list presence")
		}
		if len(input) > 0 {
			clone.AcceptedSerializations[0] = "different"
			if input[0] != "hessian2" {
				t.Fatal("clone shares the allowed set")
			}
		}
	}
}

func TestCodecReceivePolicyYAMLAndNestedClones(t *testing.T) {
	for _, test := range []struct {
		name, field string
		want        []string
	}{{"omitted", "", nil}, {"empty", "accepted-serializations: []", []string{}}, {"selected", "accepted-serializations: [hessian2]", []string{"hessian2"}}} {
		t.Run(test.name, func(t *testing.T) {
			consumer := &ConsumerConfig{}
			service := &ServiceConfig{}
			provider := &ProviderConfig{}
			require.NoError(t, yaml.Unmarshal([]byte(test.field), consumer))
			require.NoError(t, yaml.Unmarshal([]byte(test.field), service))
			providerYAML := "services:\n  svc:\n    interface: codec.Service\n"
			if test.field != "" {
				providerYAML += "    " + test.field + "\n"
			}
			require.NoError(t, yaml.Unmarshal([]byte(providerYAML), provider))
			provider.ServiceConfig = *service.Clone()
			require.Equal(t, test.want, consumer.AcceptedSerializations)
			require.Equal(t, test.want, service.AcceptedSerializations)
			require.Equal(t, test.want, provider.Services["svc"].AcceptedSerializations)
			consumer.References = map[string]*ReferenceConfig{"ref": {AcceptedSerializations: cloneAcceptedSerializations(test.want)}}

			cc, pc, sc := consumer.Clone(), provider.Clone(), service.Clone()
			require.Equal(t, test.want, cc.AcceptedSerializations)
			require.Equal(t, test.want, cc.References["ref"].AcceptedSerializations)
			require.Equal(t, test.want, pc.Services["svc"].AcceptedSerializations)
			require.Equal(t, test.want, sc.AcceptedSerializations)
			if len(test.want) > 0 {
				cc.AcceptedSerializations[0] = "changed"
				cc.References["ref"].AcceptedSerializations[0] = "changed"
				pc.AcceptedSerializations[0] = "changed"
				pc.Services["svc"].AcceptedSerializations[0] = "changed"
				sc.AcceptedSerializations[0] = "changed"
				require.Equal(t, test.want, consumer.AcceptedSerializations)
				require.Equal(t, test.want, consumer.References["ref"].AcceptedSerializations)
				require.Equal(t, test.want, provider.AcceptedSerializations)
				require.Equal(t, test.want, service.AcceptedSerializations)
			}
		})
	}
}
