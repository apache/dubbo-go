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

package definition

import (
	"encoding/json"
	"sort"
	"testing"
)

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestJSONShapeMatchesJavaContract pins the wire format.
//
// Compatibility with Dubbo Admin rests entirely on these key names: Admin
// deserializes both Java's FullServiceDefinition and this document into one
// ServiceProviderMetadata structure. Renaming any key here silently makes Go
// providers unreadable, and nothing else in the build would catch it.
func TestJSONShapeMatchesJavaContract(t *testing.T) {
	def := &ServiceDefinition{
		CanonicalName: "org.example.UserService",
		Methods: []MethodDefinition{{
			Name:           "getUser",
			ParameterTypes: []string{"string"},
			Parameters:     []ParameterDefinition{{Name: "arg0", Type: "string"}},
			ReturnType:     "*org.example.User",
		}},
		Parameters: map[string]string{
			"application": "demo",
			"interface":   "org.example.UserService",
			"release":     "dubbo-golang-3.3.0",
			"side":        "provider",
		},
		Types: []TypeDefinition{
			{Type: "*org.example.User", Items: []string{"org.example.User"}},
			{Type: "org.example.User", Properties: map[string]string{"name": "string"}},
		},
	}

	raw, err := json.Marshal(def)
	require.NoError(t, err)

	var decoded map[string]any
	require.NoError(t, json.Unmarshal(raw, &decoded))

	assert.Equal(t, []string{"canonicalName", "methods", "parameters", "types"}, sortedKeys(decoded))

	method := decoded["methods"].([]any)[0].(map[string]any)
	assert.Equal(t, []string{"name", "parameterTypes", "parameters", "returnType"}, sortedKeys(method))

	parameter := method["parameters"].([]any)[0].(map[string]any)
	assert.Equal(t, []string{"name", "type"}, sortedKeys(parameter),
		"Admin's Parameter message carries both a name and a type, unlike Java's deprecated shape")

	wrapper := decoded["types"].([]any)[0].(map[string]any)
	assert.Equal(t, []string{"items", "type"}, sortedKeys(wrapper))

	named := decoded["types"].([]any)[1].(map[string]any)
	assert.Equal(t, []string{"properties", "type"}, sortedKeys(named))
}

// TestJSONOmitsEmptyOptionalTypeFields keeps documents compact and avoids
// publishing empty enums/properties that would read as "this type has no
// fields" rather than "this field does not apply".
func TestJSONOmitsEmptyOptionalTypeFields(t *testing.T) {
	raw, err := json.Marshal(TypeDefinition{Type: "string"})
	require.NoError(t, err)
	assert.JSONEq(t, `{"type":"string"}`, string(raw))
}

// TestJSONAlwaysEmitsRequiredTopLevelFields guards the other direction: a
// service with no types must still present the key, so Admin's decoder does not
// have to distinguish absent from empty.
func TestJSONAlwaysEmitsRequiredTopLevelFields(t *testing.T) {
	raw, err := json.Marshal(&ServiceDefinition{CanonicalName: "org.example.Empty"})
	require.NoError(t, err)

	var decoded map[string]any
	require.NoError(t, json.Unmarshal(raw, &decoded))
	assert.Equal(t, []string{"canonicalName", "methods", "parameters", "types"}, sortedKeys(decoded))
}

func TestJSONRoundTrip(t *testing.T) {
	def, _ := build(t, &basicService{})

	raw, err := json.Marshal(def)
	require.NoError(t, err)

	var back ServiceDefinition
	require.NoError(t, json.Unmarshal(raw, &back))
	assert.Equal(t, *def, back)
}

func sortedKeys(m map[string]any) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
