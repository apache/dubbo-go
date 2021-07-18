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
	m := objToMap(obj).(map[string]interface{})
	assert.Equal(t, "1", m["aaAa"].(string))
	assert.Equal(t, "1", m["baBa"].(string))
	assert.Equal(t, "2", m["caCa"].(map[string]interface{})["aaAa"].(string))
	assert.Equal(t, "3", m["caCa"].(map[string]interface{})["xxYy"].(map[string]interface{})["xx"].(string))

	assert.Equal(t, reflect.Map, reflect.TypeOf(m["caCa"]).Kind())
	assert.Equal(t, reflect.Map, reflect.TypeOf(m["caCa"].(map[string]interface{})["xxYy"]).Kind())
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
	m := objToMap(testData).(map[string]interface{})

	assert.Equal(t, "1", m["aaAa"].(string))
	assert.Equal(t, "1", m["baBa"].(string))
	assert.Equal(t, "2", m["caCa"].([]interface{})[0].(map[string]interface{})["aaAa"].(string))
	assert.Equal(t, "3", m["caCa"].([]interface{})[0].(map[string]interface{})["xxYy"].(map[string]interface{})["xx"].(string))

	assert.Equal(t, reflect.Slice, reflect.TypeOf(m["caCa"]).Kind())
	assert.Equal(t, reflect.Map, reflect.TypeOf(m["caCa"].([]interface{})[0].(map[string]interface{})["xxYy"]).Kind())
}

func TestObjToMap_Map(t *testing.T) {
	var testData struct {
		AaAa   string
		Baba   map[string]interface{}
		CaCa   map[string]string
		DdDd   map[string]interface{}
		IntMap map[int]interface{}
	}
	testData.AaAa = "aaaa"
	testData.Baba = make(map[string]interface{})
	testData.CaCa = make(map[string]string)
	testData.DdDd = nil
	testData.IntMap = make(map[int]interface{})

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
	mappedStruct := m.(map[string]interface{})
	assert.Equal(t, reflect.String, reflect.TypeOf(mappedStruct["aaAa"]).Kind())
	assert.Equal(t, reflect.Map, reflect.TypeOf(mappedStruct["baba"]).Kind())
	assert.Equal(t, reflect.Map, reflect.TypeOf(mappedStruct["baba"].(map[interface{}]interface{})["struct"]).Kind())
	assert.Equal(t, "str", mappedStruct["baba"].(map[interface{}]interface{})["struct"].(map[string]interface{})["str"])
	assert.Equal(t, nil, mappedStruct["baba"].(map[interface{}]interface{})["nil"])
	assert.Equal(t, reflect.Map, reflect.TypeOf(mappedStruct["caCa"]).Kind())
	assert.Equal(t, reflect.Map, reflect.TypeOf(mappedStruct["ddDd"]).Kind())
	intMap := mappedStruct["intMap"]
	assert.Equal(t, reflect.Map, reflect.TypeOf(intMap).Kind())
	assert.Equal(t, 1, intMap.(map[interface{}]interface{})[1])
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
	assert.Nil(t, err)
	// parent
	assert.Equal(t, "xavierniu", m.(map[string]interface{})["name"].(string))
	assert.Equal(t, 30, m.(map[string]interface{})["age"].(int))
	assert.Equal(t, "org.apache.dubbo.mockParent", m.(map[string]interface{})["class"].(string))
	// child
	assert.Equal(t, 20, m.(map[string]interface{})["child"].(map[string]interface{})["age"].(int))
	assert.Equal(t, "lmc", m.(map[string]interface{})["child"].(map[string]interface{})["name"].(string))
	assert.Equal(t, "org.apache.dubbo.mockChild", m.(map[string]interface{})["child"].(map[string]interface{})["class"].(string))

	r, err := mockMapGeneralizer.Realize(m, reflect.TypeOf(p))
	assert.Nil(t, err)
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
	assert.Nil(t, err)
	assert.Equal(t, "lmc", m.([]interface{})[0].(map[string]interface{})["name"].(string))
	assert.Equal(t, 20, m.([]interface{})[0].(map[string]interface{})["age"].(int))
	assert.Equal(t, "lmc1", m.([]interface{})[1].(map[string]interface{})["name"].(string))
	assert.Equal(t, 21, m.([]interface{})[1].(map[string]interface{})["age"].(int))

	r, err := mockMapGeneralizer.Realize(m, reflect.TypeOf(pojoArr))
	assert.Nil(t, err)
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
	assert.Nil(t, m.(map[string]interface{})["child"])

	r, err := mockMapGeneralizer.Realize(m, reflect.TypeOf(p))
	assert.Nil(t, err)
	rMockParent, ok := r.(mockParent)
	assert.True(t, ok)
	assert.Nil(t, rMockParent.Child)
}
