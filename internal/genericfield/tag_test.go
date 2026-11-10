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

package genericfield_test

import (
	"reflect"
	"testing"
)

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

import (
	"dubbo.apache.org/dubbo-go/v3/internal/genericfield"
)

type taggedFields struct {
	Ünicode   string
	Named     string            `m:"display_name"`
	Optional  string            `m:"optional,omitempty"`
	Ignored   string            `m:"-"`
	Embedded  struct{}          `m:",squash"`
	Remaining map[string]string `m:",remain"`
}

func TestParseMTagDefinesTheSharedGenericFieldContract(t *testing.T) {
	typ := reflect.TypeFor[taggedFields]()
	tests := []struct {
		field string
		want  genericfield.Tag
	}{
		{"Ünicode", genericfield.Tag{Name: "ünicode"}},
		{"Named", genericfield.Tag{Name: "display_name"}},
		{"Optional", genericfield.Tag{Name: "optional", OmitEmpty: true}},
		{"Ignored", genericfield.Tag{Name: "ignored", Ignore: true}},
		{"Embedded", genericfield.Tag{Name: "embedded", Squash: true}},
		{"Remaining", genericfield.Tag{Name: "remaining", UnknownOptions: []string{"remain"}}},
	}

	for _, tt := range tests {
		t.Run(tt.field, func(t *testing.T) {
			field, ok := typ.FieldByName(tt.field)
			require.True(t, ok)
			assert.Equal(t, tt.want, genericfield.ParseMTag(field))
		})
	}
}
