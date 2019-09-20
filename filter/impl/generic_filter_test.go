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

package impl

import (
	"reflect"
	"testing"
)
import (
	"github.com/stretchr/testify/assert"
)

func Test_struct2MapAll(t *testing.T) {
	var testData struct {
		AaAa string `m:"aaAa"`
		BaBa string
		CaCa struct {
			AaAa string
			BaBa string `m:"baBa"`
			XxYy struct {
				xxXx string `m:"xxXx"`
				Xx   string `m:"xx"`
			} `m:"xxYy"`
		} `m:"caCa"`
	}
	testData.AaAa = "1"
	testData.BaBa = "1"
	testData.CaCa.BaBa = "2"
	testData.CaCa.AaAa = "2"
	testData.CaCa.XxYy.xxXx = "3"
	testData.CaCa.XxYy.Xx = "3"
	m := struct2MapAll(testData).(map[string]interface{})
	assert.Equal(t, "1", m["aaAa"].(string))
	assert.Equal(t, "1", m["baBa"].(string))
	assert.Equal(t, "2", m["caCa"].(map[string]interface{})["aaAa"].(string))
	assert.Equal(t, "3", m["caCa"].(map[string]interface{})["xxYy"].(map[string]interface{})["xx"].(string))

	assert.Equal(t, reflect.Map, reflect.TypeOf(m["caCa"]).Kind())
	assert.Equal(t, reflect.Map, reflect.TypeOf(m["caCa"].(map[string]interface{})["xxYy"]).Kind())
}

type testStruct struct {
	AaAa string
	BaBa string `m:"baBa"`
	XxYy struct {
		xxXx string `m:"xxXx"`
		Xx   string `m:"xx"`
	} `m:"xxYy"`
}

func Test_struct2MapAll_Slice(t *testing.T) {
	var testData struct {
		AaAa string `m:"aaAa"`
		BaBa string
		CaCa []testStruct `m:"caCa"`
	}
	testData.AaAa = "1"
	testData.BaBa = "1"
	var tmp testStruct
	tmp.BaBa = "2"
	tmp.AaAa = "2"
	tmp.XxYy.xxXx = "3"
	tmp.XxYy.Xx = "3"
	testData.CaCa = append(testData.CaCa, tmp)
	m := struct2MapAll(testData).(map[string]interface{})

	assert.Equal(t, "1", m["aaAa"].(string))
	assert.Equal(t, "1", m["baBa"].(string))
	assert.Equal(t, "2", m["caCa"].([]interface{})[0].(map[string]interface{})["aaAa"].(string))
	assert.Equal(t, "3", m["caCa"].([]interface{})[0].(map[string]interface{})["xxYy"].(map[string]interface{})["xx"].(string))

	assert.Equal(t, reflect.Slice, reflect.TypeOf(m["caCa"]).Kind())
	assert.Equal(t, reflect.Map, reflect.TypeOf(m["caCa"].([]interface{})[0].(map[string]interface{})["xxYy"]).Kind())
}
func Test_struct2MapAll_Map(t *testing.T) {
	var testData struct {
		AaAa string
		Baba map[string]interface{}
		CaCa map[string]string
		DdDd map[string]interface{}
	}
	testData.AaAa = "aaaa"
	testData.Baba = make(map[string]interface{})
	testData.CaCa = make(map[string]string)
	testData.DdDd = nil

	testData.Baba["kk"] = 1
	var structData struct {
		Str string
	}
	structData.Str = "str"
	testData.Baba["struct"] = structData
	testData.Baba["nil"] = nil
	testData.CaCa["k1"] = "v1"
	testData.CaCa["kv2"] = "v2"
	m := struct2MapAll(testData)

	assert.Equal(t, reflect.Map, reflect.TypeOf(m).Kind())
	assert.Equal(t, reflect.String, reflect.TypeOf(m.(map[string]interface{})["aaAa"]).Kind())
	assert.Equal(t, reflect.Map, reflect.TypeOf(m.(map[string]interface{})["baba"]).Kind())
	assert.Equal(t, reflect.Map, reflect.TypeOf(m.(map[string]interface{})["baba"].(map[string]interface{})["struct"]).Kind())
	assert.Equal(t, "str", m.(map[string]interface{})["baba"].(map[string]interface{})["struct"].(map[string]interface{})["str"])
	assert.Equal(t, nil, m.(map[string]interface{})["baba"].(map[string]interface{})["nil"])
	assert.Equal(t, reflect.Map, reflect.TypeOf(m.(map[string]interface{})["caCa"]).Kind())
	assert.Equal(t, reflect.Map, reflect.TypeOf(m.(map[string]interface{})["ddDd"]).Kind())
}
