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

package parser

import (
	"encoding/json"
	"testing"
)

import (
	"github.com/stretchr/testify/assert"
)

var (
	testDataStore = `
	{
		"book":[
			{
				"category":"reference",
				"author":"Nigel Rees",
				"title":"Sayings of the Century",
				"price":8.95
			},
			{
				"category":"fiction",
				"author":"Evelyn Waugh",
				"title":"Sword of Honor",
				"price":12.99
			},
			{
				"category":"fiction",
				"author":"Herman Melville",
				"title":"Moby Dick",
				"isbn":"0-553-21311-3",
				"price":8.99
			},
			{
				"category":"fiction",
				"author":"J. R. R. Tolkien",
				"title":"The Lord of the Rings",
				"isbn":"0-395-19395-8",
				"price":22.99
			}
		],
		"bicycle":{
			"color":"red",
			"price":19.95
		}
	}
	`

	testDataBicyle = `
	{
		"color":"red",
		"price":19.95
	}
	`
)

func TestParseArgumentsByExpression(t *testing.T) {

	var (
		argStore, argBicyle interface{}
	)

	json.Unmarshal([]byte(testDataStore), &argStore)
	json.Unmarshal([]byte(testDataBicyle), &argBicyle)

	t.Run("test-case-1", func(t *testing.T) {
		ret := ParseArgumentsByExpression("param.$.book[0].category", []interface{}{argStore})
		assert.Equal(t, "reference", ret)
	})

	t.Run("test-case-2", func(t *testing.T) {
		ret := ParseArgumentsByExpression("param[0].$.book[0].category", []interface{}{argStore, argBicyle})
		assert.Equal(t, "reference", ret)
	})

	t.Run("test-case-2", func(t *testing.T) {
		ret := ParseArgumentsByExpression("param[1].$.color", []interface{}{argStore, argBicyle})
		assert.Equal(t, "red", ret)
	})

	t.Run("test-case-3", func(t *testing.T) {
		ret := ParseArgumentsByExpression("param.$.color", []interface{}{argBicyle})
		assert.Equal(t, "red", ret)
	})

}

func Test_resolveIndex(t *testing.T) {
	type args struct {
		key string
	}
	tests := []struct {
		name  string
		args  args
		want  int
		want1 string
	}{
		{
			name: "case-1",
			args: args{
				key: "param.$.key",
			},
			want:  0,
			want1: "$.key",
		},
		{
			name: "case-2",
			args: args{
				key: "param[1].$.key",
			},
			want:  1,
			want1: "$.key",
		},
		{
			name: "case-3",
			args: args{
				key: "param[10].$.key",
			},
			want:  10,
			want1: "$.key",
		},
		{
			name: "case-4",
			args: args{
				key: "param[11]$.key",
			},
			want:  11,
			want1: "$.key",
		},
		{
			name: "case-5",
			args: args{
				key: "param[11]key",
			},
			want:  11,
			want1: "key",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, got1 := resolveIndex(tt.args.key)
			if got != tt.want {
				t.Errorf("resolveIndex() got = %v, want %v", got, tt.want)
			}
			if got1 != tt.want1 {
				t.Errorf("resolveIndex() got1 = %v, want %v", got1, tt.want1)
			}
		})
	}
}
