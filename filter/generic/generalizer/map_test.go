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

package generalizer

import (
	"reflect"
	"testing"
	"time"
)

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type testPlainObj struct {
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
	DaDa time.Time
	EeEe int
}

func TestObjToMap(t *testing.T) {
	obj := &testPlainObj{}
	obj.AaAa = "1"
	obj.BaBa = "1"
	obj.CaCa.BaBa = "2"
	obj.CaCa.AaAa = "2"
	obj.CaCa.XxYy.xxXx = "3"
	obj.CaCa.XxYy.Xx = "3"
	obj.DaDa = time.Date(2020, 10, 29, 2, 34, 0, 0, time.Local)
	obj.EeEe = 100
	m := objToMap(obj).(map[string]any)
	assert.Equal(t, "1", m["aaAa"].(string))
	assert.Equal(t, "1", m["baBa"].(string))
	assert.Equal(t, "2", m["caCa"].(map[string]any)["aaAa"].(string))
	assert.Equal(t, "3", m["caCa"].(map[string]any)["xxYy"].(map[string]any)["xx"].(string))

	assert.Equal(t, reflect.Map, reflect.TypeOf(m["caCa"]).Kind())
	assert.Equal(t, reflect.Map, reflect.TypeOf(m["caCa"].(map[string]any)["xxYy"]).Kind())
	assert.Equal(t, "2020-10-29 02:34:00", m["daDa"].(time.Time).Format("2006-01-02 15:04:05"))
	assert.Equal(t, 100, m["eeEe"].(int))
}

type testStruct struct {
	AaAa string
	BaBa string `m:"baBa"`
	XxYy struct {
		xxXx string `m:"xxXx"`
		Xx   string `m:"xx"`
	} `m:"xxYy"`
}

func TestObjToMap_Slice(t *testing.T) {
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
	m := objToMap(testData).(map[string]any)

	assert.Equal(t, "1", m["aaAa"].(string))
	assert.Equal(t, "1", m["baBa"].(string))
	assert.Equal(t, "2", m["caCa"].([]any)[0].(map[string]any)["aaAa"].(string))
	assert.Equal(t, "3", m["caCa"].([]any)[0].(map[string]any)["xxYy"].(map[string]any)["xx"].(string))

	assert.Equal(t, reflect.Slice, reflect.TypeOf(m["caCa"]).Kind())
	assert.Equal(t, reflect.Map, reflect.TypeOf(m["caCa"].([]any)[0].(map[string]any)["xxYy"]).Kind())
}

func TestObjToMap_Map(t *testing.T) {
	var testData struct {
		AaAa   string
		Baba   map[string]any
		CaCa   map[string]string
		DdDd   map[string]any
		IntMap map[int]any
	}
	testData.AaAa = "aaaa"
	testData.Baba = make(map[string]any)
	testData.CaCa = make(map[string]string)
	testData.DdDd = nil
	testData.IntMap = make(map[int]any)

	testData.Baba["kk"] = 1
	var structData struct {
		Str string
	}
	structData.Str = "str"
	testData.Baba["struct"] = structData
	testData.Baba["nil"] = nil
	testData.CaCa["k1"] = "v1"
	testData.CaCa["kv2"] = "v2"
	testData.IntMap[1] = 1
	m := objToMap(testData)

	assert.Equal(t, reflect.Map, reflect.TypeOf(m).Kind())
	mappedStruct := m.(map[string]any)
	assert.Equal(t, reflect.String, reflect.TypeOf(mappedStruct["aaAa"]).Kind())
	assert.Equal(t, reflect.Map, reflect.TypeOf(mappedStruct["baba"]).Kind())
	assert.Equal(t, reflect.Map, reflect.TypeOf(mappedStruct["baba"].(map[any]any)["struct"]).Kind())
	assert.Equal(t, "str", mappedStruct["baba"].(map[any]any)["struct"].(map[string]any)["str"])
	assert.Nil(t, mappedStruct["baba"].(map[any]any)["nil"])
	assert.Equal(t, reflect.Map, reflect.TypeOf(mappedStruct["caCa"]).Kind())
	assert.Equal(t, reflect.Map, reflect.TypeOf(mappedStruct["ddDd"]).Kind())
	intMap := mappedStruct["intMap"]
	assert.Equal(t, reflect.Map, reflect.TypeOf(intMap).Kind())
	assert.Equal(t, 1, intMap.(map[any]any)[1])
}

var mockMapGeneralizer = GetMapGeneralizer()

type mockParent struct {
	Gender, Email, Name string
	Age                 int
	Child               *mockChild
}

func (p mockParent) JavaClassName() string {
	return "org.apache.dubbo.mockParent"
}

type mockChild struct {
	Gender, Email, Name string
	Age                 int
}

func (c *mockChild) JavaClassName() string {
	return "org.apache.dubbo.mockChild"
}

func TestPOJOClassName(t *testing.T) {
	c := &mockChild{
		Age:    20,
		Gender: "male",
		Email:  "lmc@example.com",
		Name:   "lmc",
	}
	p := mockParent{
		Age:    30,
		Gender: "male",
		Email:  "xavierniu@example.com",
		Name:   "xavierniu",
		Child:  c,
	}

	m, err := mockMapGeneralizer.Generalize(p)
	require.NoError(t, err)
	// parent
	assert.Equal(t, "xavierniu", m.(map[string]any)["name"].(string))
	assert.Equal(t, 30, m.(map[string]any)["age"].(int))
	assert.Equal(t, "org.apache.dubbo.mockParent", m.(map[string]any)["class"].(string))
	// child
	assert.Equal(t, 20, m.(map[string]any)["child"].(map[string]any)["age"].(int))
	assert.Equal(t, "lmc", m.(map[string]any)["child"].(map[string]any)["name"].(string))
	assert.Equal(t, "org.apache.dubbo.mockChild", m.(map[string]any)["child"].(map[string]any)["class"].(string))

	r, err := mockMapGeneralizer.Realize(m, reflect.TypeOf(p))
	require.NoError(t, err)
	rMockParent, ok := r.(mockParent)
	assert.True(t, ok)
	// parent
	assert.Equal(t, "xavierniu", rMockParent.Name)
	assert.Equal(t, 30, rMockParent.Age)
	// child
	assert.Equal(t, "lmc", rMockParent.Child.Name)
	assert.Equal(t, 20, rMockParent.Child.Age)
}

func TestPOJOArray(t *testing.T) {
	c1 := &mockChild{
		Age:    20,
		Gender: "male",
		Email:  "lmc@example.com",
		Name:   "lmc",
	}
	c2 := &mockChild{
		Age:    21,
		Gender: "male",
		Email:  "lmc1@example.com",
		Name:   "lmc1",
	}

	pojoArr := []*mockChild{c1, c2}

	m, err := mockMapGeneralizer.Generalize(pojoArr)
	require.NoError(t, err)
	assert.Equal(t, "lmc", m.([]any)[0].(map[string]any)["name"].(string))
	assert.Equal(t, 20, m.([]any)[0].(map[string]any)["age"].(int))
	assert.Equal(t, "lmc1", m.([]any)[1].(map[string]any)["name"].(string))
	assert.Equal(t, 21, m.([]any)[1].(map[string]any)["age"].(int))

	r, err := mockMapGeneralizer.Realize(m, reflect.TypeOf(pojoArr))
	require.NoError(t, err)
	rPojoArr, ok := r.([]*mockChild)
	assert.True(t, ok)
	assert.Equal(t, "lmc", rPojoArr[0].Name)
	assert.Equal(t, 20, rPojoArr[0].Age)
	assert.Equal(t, "lmc1", rPojoArr[1].Name)
	assert.Equal(t, 21, rPojoArr[1].Age)
}

func TestNullField(t *testing.T) {
	p := mockParent{
		Age:    30,
		Gender: "male",
		Email:  "xavierniu@example.com",
		Name:   "xavierniu",
	}

	m, _ := mockMapGeneralizer.Generalize(p)
	assert.Nil(t, m.(map[string]any)["child"])

	r, err := mockMapGeneralizer.Realize(m, reflect.TypeOf(p))
	require.NoError(t, err)
	rMockParent, ok := r.(mockParent)
	assert.True(t, ok)
	assert.Nil(t, rMockParent.Child)
}
