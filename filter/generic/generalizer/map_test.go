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
	"encoding/json"
	"math"
	"reflect"
	"strconv"
	"testing"
	"time"
	"unicode/utf8"
)

import (
	"github.com/mitchellh/mapstructure"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common/config"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
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

type testMTagObj struct {
	UserID string `m:"user_id"`
	Name   string
}

type testMTagEmbedded struct {
	City string `m:"city_name"`
}

type testMTagOptionsObj struct {
	ID       string           `m:"id,omitempty"`
	Empty    string           `m:"empty,omitempty"`
	Ignored  string           `m:"-"`
	Embedded testMTagEmbedded `m:",squash"`
}

type testMTagSquashedPOJO struct {
	City string `m:"city"`
}

func (testMTagSquashedPOJO) JavaClassName() string {
	return "org.apache.dubbo.testMTagSquashedPOJO"
}

type testMTagSquashParentPOJO struct {
	Name     string               `m:"name"`
	Embedded testMTagSquashedPOJO `m:",squash"`
}

func (testMTagSquashParentPOJO) JavaClassName() string {
	return "org.apache.dubbo.testMTagSquashParentPOJO"
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
	generalized, err := objToMap(obj)
	require.NoError(t, err)
	m := generalized.(map[string]any)
	assert.Equal(t, "1", m["aaAa"].(string))
	assert.Equal(t, "1", m["baBa"].(string))
	assert.Equal(t, "2", m["caCa"].(map[string]any)["aaAa"].(string))
	assert.Equal(t, "3", m["caCa"].(map[string]any)["xxYy"].(map[string]any)["xx"].(string))

	assert.Equal(t, reflect.Map, reflect.TypeOf(m["caCa"]).Kind())
	assert.Equal(t, reflect.Map, reflect.TypeOf(m["caCa"].(map[string]any)["xxYy"]).Kind())
	assert.Equal(t, "2020-10-29 02:34:00", m["daDa"].(time.Time).Format("2006-01-02 15:04:05"))
	assert.Equal(t, 100, m["eeEe"].(int))
}

func TestMapGeneralizeNilPointers(t *testing.T) {
	var plain *testPlainObj
	var pojo *testMTagSquashedPOJO
	var number *int
	var indirect **testPlainObj
	tests := []struct {
		name  string
		input any
		want  any
	}{
		{"plain pointer", plain, nil},
		{"POJO pointer", pojo, nil},
		{"scalar pointer", number, nil},
		{"indirect pointer", &indirect, nil},
		{"slice element", []any{plain, pojo, number}, []any{nil, nil, nil}},
		{"map value", map[string]any{"plain": plain, "pojo": pojo}, map[any]any{"plain": nil, "pojo": nil}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var got any
			var err error
			require.NotPanics(t, func() {
				got, err = GetMapGeneralizer().Generalize(tt.input)
			})
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestMTagRoundTrip(t *testing.T) {
	original := testMTagObj{
		UserID: "42",
		Name:   "alice",
	}

	generalized, err := mockMapGeneralizer.Generalize(original)
	require.NoError(t, err)
	generalizedMap, ok := generalized.(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "42", generalizedMap["user_id"])

	realized, err := mockMapGeneralizer.Realize(generalized, reflect.TypeFor[testMTagObj]())
	require.NoError(t, err)
	assert.Equal(t, original, realized)
}

func TestMTagOptionsRoundTrip(t *testing.T) {
	original := testMTagOptionsObj{
		ID:      "42",
		Ignored: "secret",
		Embedded: testMTagEmbedded{
			City: "Hangzhou",
		},
	}

	generalized, err := mockMapGeneralizer.Generalize(original)
	require.NoError(t, err)
	generalizedMap, ok := generalized.(map[string]any)
	require.True(t, ok)
	expected := map[string]any{}
	decoder, err := mapstructure.NewDecoder(&mapstructure.DecoderConfig{
		Result:  &expected,
		TagName: "m",
	})
	require.NoError(t, err)
	require.NoError(t, decoder.Decode(original))
	assert.Equal(t, expected, generalizedMap)

	realized, err := mockMapGeneralizer.Realize(generalized, reflect.TypeFor[testMTagOptionsObj]())
	require.NoError(t, err)
	realizedObj, ok := realized.(testMTagOptionsObj)
	require.True(t, ok)
	assert.Equal(t, original.ID, realizedObj.ID)
	assert.Empty(t, realizedObj.Empty)
	assert.Empty(t, realizedObj.Ignored)
	assert.Equal(t, original.Embedded.City, realizedObj.Embedded.City)
}

func TestMTagSquashPreservesParentPOJOClass(t *testing.T) {
	original := testMTagSquashParentPOJO{
		Name: "alice",
		Embedded: testMTagSquashedPOJO{
			City: "Hangzhou",
		},
	}

	generalized, err := GetMapGeneralizer().Generalize(original)
	require.NoError(t, err)
	generalizedMap, ok := generalized.(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "org.apache.dubbo.testMTagSquashParentPOJO", generalizedMap["class"])
	assert.Equal(t, "alice", generalizedMap["name"])
	assert.Equal(t, "Hangzhou", generalizedMap["city"])
}

func TestMTagSquashRejectsNonStruct(t *testing.T) {
	obj := struct {
		Value string `m:",squash"`
	}{Value: "invalid"}

	_, err := mockMapGeneralizer.Generalize(obj)
	require.ErrorContains(t, err, "cannot squash non-struct type")
}

func TestUntaggedNameLowercasesByRune(t *testing.T) {
	// Lowercasing the first byte instead of the first rune split multi-byte
	// leading runes, producing keys that were not valid UTF-8.
	obj := struct {
		Ünicode string
		Ascii   string
	}{Ünicode: "value", Ascii: "value"}

	out, err := mockMapGeneralizer.Generalize(obj)
	require.NoError(t, err)
	m := out.(map[string]any)

	assert.Contains(t, m, "ünicode")
	assert.Contains(t, m, "ascii", "ASCII names must keep behaving exactly as before")
	for key := range m {
		assert.True(t, utf8.ValidString(key), "key %q is not valid UTF-8", key)
	}
}

func TestUntaggedNameRoundTripsForUnicode(t *testing.T) {
	type unicodeNamed struct {
		Ünicode string
	}
	original := unicodeNamed{Ünicode: "value"}

	generalized, err := mockMapGeneralizer.Generalize(original)
	require.NoError(t, err)

	realized, err := mockMapGeneralizer.Realize(generalized, reflect.TypeFor[unicodeNamed]())
	require.NoError(t, err)
	assert.Equal(t, original, realized)
}

func TestRealizeRejectsUnsignedOverflow(t *testing.T) {
	tests := []struct {
		name string
		data any
		typ  reflect.Type
		want any
	}{
		{name: "uint8 max", data: int16(math.MaxUint8), typ: reflect.TypeFor[uint8](), want: uint8(math.MaxUint8)},
		{name: "uint16 max", data: int32(math.MaxUint16), typ: reflect.TypeFor[uint16](), want: uint16(math.MaxUint16)},
		{name: "uint32 max", data: int64(math.MaxUint32), typ: reflect.TypeFor[uint32](), want: uint32(math.MaxUint32)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := mockMapGeneralizer.Realize(tt.data, tt.typ)
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}

	overflows := []struct {
		name string
		data any
		typ  reflect.Type
	}{
		{name: "negative uint8", data: int16(-1), typ: reflect.TypeFor[uint8]()},
		{name: "uint8 overflow", data: int16(math.MaxUint8 + 1), typ: reflect.TypeFor[uint8]()},
		{name: "uint16 overflow", data: int32(math.MaxUint16 + 1), typ: reflect.TypeFor[uint16]()},
		{name: "uint32 overflow", data: int64(math.MaxUint32 + 1), typ: reflect.TypeFor[uint32]()},
	}
	for _, tt := range overflows {
		t.Run(tt.name, func(t *testing.T) {
			_, err := mockMapGeneralizer.Realize(tt.data, tt.typ)
			require.Error(t, err)
			assert.Contains(t, err.Error(), "overflows")
		})
	}
}

type timedPayload struct {
	At    time.Time
	Maybe *time.Time
}

// TestRealizeJavaDate covers the forms a java.util.Date parameter can arrive
// in: a Date from a hessian2 caller, or epoch millis / RFC 3339 from a JSON
// caller such as Dubbo Admin.
func TestRealizeJavaDate(t *testing.T) {
	want := time.UnixMilli(1762826400123)
	for _, tt := range []struct {
		name string
		data any
	}{
		{name: "time.Time", data: want},
		{name: "int64 millis", data: int64(1762826400123)},
		{name: "float64 millis", data: float64(1762826400123)},
		{name: "json.Number millis", data: json.Number("1762826400123")},
		{name: "RFC 3339", data: want.UTC().Format(time.RFC3339Nano)},
	} {
		t.Run(tt.name, func(t *testing.T) {
			got, err := mockMapGeneralizer.Realize(
				map[string]any{"at": tt.data, "maybe": tt.data},
				reflect.TypeFor[timedPayload](),
			)
			require.NoError(t, err)
			payload := got.(timedPayload)
			assert.True(t, want.Equal(payload.At), "got %v", payload.At)
			require.NotNil(t, payload.Maybe)
			assert.True(t, want.Equal(*payload.Maybe), "got %v", *payload.Maybe)
		})
	}

	// Java's CompatibleTypeUtils layout, read in the local zone.
	local, err := mockMapGeneralizer.Realize(
		map[string]any{"at": "2026-11-11 08:30:00"}, reflect.TypeFor[timedPayload]())
	require.NoError(t, err)
	assert.True(t, time.Date(2026, 11, 11, 8, 30, 0, 0, time.Local).Equal(local.(timedPayload).At))

	for _, data := range []any{"2026-11-11", 1.5, json.Number("1.5")} {
		_, err := mockMapGeneralizer.Realize(map[string]any{"at": data}, reflect.TypeFor[timedPayload]())
		require.Error(t, err, "%v", data)
	}
}

type DatedItem struct {
	At time.Time
}

type datedTree struct {
	Items  []DatedItem
	ByName map[string]*DatedItem
	Times  []time.Time
	Index  map[string]time.Time
	Nested *DatedItem
}

// TestRealizeJavaDateNested checks that the conversion applies wherever a
// time.Time sits: inside slices, maps, pointers and nested structs.
func TestRealizeJavaDateNested(t *testing.T) {
	want := time.UnixMilli(1762826400123)
	millis := int64(1762826400123)
	got, err := mockMapGeneralizer.Realize(map[string]any{
		"items":  []any{map[string]any{"at": millis}},
		"byName": map[string]any{"a": map[string]any{"at": want.Format(time.RFC3339Nano)}},
		"times":  []any{float64(millis), want},
		"index":  map[any]any{"k": json.Number("1762826400123")},
		"nested": map[string]any{"at": millis},
	}, reflect.TypeFor[datedTree]())
	require.NoError(t, err)
	tree := got.(datedTree)
	require.Len(t, tree.Items, 1)
	assert.True(t, want.Equal(tree.Items[0].At))
	require.NotNil(t, tree.ByName["a"])
	assert.True(t, want.Equal(tree.ByName["a"].At))
	require.Len(t, tree.Times, 2)
	assert.True(t, want.Equal(tree.Times[0]))
	assert.True(t, want.Equal(tree.Times[1]))
	assert.True(t, want.Equal(tree.Index["k"]))
	require.NotNil(t, tree.Nested)
	assert.True(t, want.Equal(tree.Nested.At))

	top, err := mockMapGeneralizer.Realize(millis, reflect.TypeFor[time.Time]())
	require.NoError(t, err)
	assert.True(t, want.Equal(top.(time.Time)))
}

// TestGeneralizeJavaDate checks that a time.Time result reaches hessian2 as a
// time.Time, so it is written as java.util.Date, wherever it sits. It used to
// be walked as a struct and come out as an empty map.
func TestGeneralizeJavaDate(t *testing.T) {
	want := time.UnixMilli(1762826400123)

	top, err := mockMapGeneralizer.Generalize(want)
	require.NoError(t, err)
	assert.Equal(t, want, top)

	ptr, err := mockMapGeneralizer.Generalize(&want)
	require.NoError(t, err)
	assert.Equal(t, want, ptr)

	var nilTime *time.Time
	none, err := mockMapGeneralizer.Generalize(nilTime)
	require.NoError(t, err)
	assert.Nil(t, none)

	list, err := mockMapGeneralizer.Generalize([]time.Time{want})
	require.NoError(t, err)
	assert.Equal(t, []any{want}, list)

	dict, err := mockMapGeneralizer.Generalize(map[string]time.Time{"k": want})
	require.NoError(t, err)
	assert.Equal(t, map[any]any{"k": want}, dict)

	tree, err := mockMapGeneralizer.Generalize(datedTree{
		Items:  []DatedItem{{At: want}},
		ByName: map[string]*DatedItem{"a": {At: want}},
		Times:  []time.Time{want},
		Index:  map[string]time.Time{"k": want},
		Nested: &DatedItem{At: want},
	})
	require.NoError(t, err)
	m := tree.(map[string]any)
	assert.Equal(t, want, m["items"].([]any)[0].(map[string]any)["at"])
	assert.Equal(t, want, m["byName"].(map[any]any)["a"].(map[string]any)["at"])
	assert.Equal(t, []any{want}, m["times"])
	assert.Equal(t, map[any]any{"k": want}, m["index"])
	assert.Equal(t, want, m["nested"].(map[string]any)["at"])

	// Round trip through Realize gives the same values back.
	back, err := mockMapGeneralizer.Realize(tree, reflect.TypeFor[datedTree]())
	require.NoError(t, err)
	assert.True(t, want.Equal(back.(datedTree).Nested.At))
	assert.True(t, want.Equal(back.(datedTree).Index["k"]))
}

type bytePayload struct {
	Data []byte
}

func TestRealizeJavaSignedByteArray(t *testing.T) {
	got, err := mockMapGeneralizer.Realize(
		map[string]any{"data": []int8{0, 127, -128, -1}},
		reflect.TypeFor[bytePayload](),
	)
	require.NoError(t, err)
	assert.Equal(t, bytePayload{Data: []byte{0, 127, 128, 255}}, got)

	_, err = mockMapGeneralizer.Realize(
		map[string]any{"data": []int16{0, math.MaxUint8 + 1}},
		reflect.TypeFor[bytePayload](),
	)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "overflows")
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
	generalized, err := objToMap(testData)
	require.NoError(t, err)
	m := generalized.(map[string]any)

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
	m, err := objToMap(testData)
	require.NoError(t, err)

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

	r, err := mockMapGeneralizer.Realize(m, reflect.TypeFor[mockParent]())
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

	r, err := mockMapGeneralizer.Realize(m, reflect.TypeFor[[]*mockChild]())
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

	r, err := mockMapGeneralizer.Realize(m, reflect.TypeFor[mockParent]())
	require.NoError(t, err)
	rMockParent, ok := r.(mockParent)
	assert.True(t, ok)
	assert.Nil(t, rMockParent.Child)
}

func setGenericIncludeClass(t *testing.T, value *bool) {
	env := config.GetEnvInstance()
	if value == nil {
		env.UpdateAppExternalConfigMap(map[string]string{})
		env.UpdateExternalConfigMap(map[string]string{})
		return
	}

	env.UpdateAppExternalConfigMap(map[string]string{
		constant.GenericIncludeClassKey: strconv.FormatBool(*value),
	})

	t.Cleanup(func() {
		env.UpdateAppExternalConfigMap(map[string]string{})
		env.UpdateExternalConfigMap(map[string]string{})
	})
}

func TestRemoveClass(t *testing.T) {
	input := map[string]any{
		"class": "root",
		"name":  "n",
		"child": map[any]any{
			"class": "child",
			"x":     1,
			1:       "keep",
		},
		"list": []any{
			map[string]any{"class": "item", "y": 2},
			"v",
		},
	}

	output := removeClass(input).(map[string]any)
	if _, ok := output["class"]; ok {
		t.Fatalf("expected root class to be removed")
	}

	child := output["child"].(map[any]any)
	if _, ok := child["class"]; ok {
		t.Fatalf("expected child class to be removed")
	}
	assert.Equal(t, "keep", child[1])

	list := output["list"].([]any)
	item := list[0].(map[string]any)
	if _, ok := item["class"]; ok {
		t.Fatalf("expected list item class to be removed")
	}
}

func TestGenericIncludeClass_ConfigTrue(t *testing.T) {
	val := true
	setGenericIncludeClass(t, &val)

	child := &mockChild{
		Age:    20,
		Gender: "male",
		Email:  "lmc@example.com",
		Name:   "lmc",
	}
	parent := mockParent{
		Age:    30,
		Gender: "male",
		Email:  "xavierniu@example.com",
		Name:   "xavierniu",
		Child:  child,
	}

	m, err := mockMapGeneralizer.Generalize(parent)
	require.NoError(t, err)
	mMap := m.(map[string]any)
	_, ok := mMap["class"]
	assert.True(t, ok)
	_, ok = mMap["child"].(map[string]any)["class"]
	assert.True(t, ok)

	r, err := mockMapGeneralizer.Realize(m, reflect.TypeFor[mockParent]())
	require.NoError(t, err)
	rParent, ok := r.(mockParent)
	assert.True(t, ok)
	assert.Equal(t, "xavierniu", rParent.Name)
	assert.Equal(t, 30, rParent.Age)
	assert.Equal(t, "lmc", rParent.Child.Name)
	assert.Equal(t, 20, rParent.Child.Age)
}

func TestGenericIncludeClass_ConfigFalse(t *testing.T) {
	val := false
	setGenericIncludeClass(t, &val)

	child := &mockChild{
		Age:    20,
		Gender: "male",
		Email:  "lmc@example.com",
		Name:   "lmc",
	}
	parent := mockParent{
		Age:    30,
		Gender: "male",
		Email:  "xavierniu@example.com",
		Name:   "xavierniu",
		Child:  child,
	}

	m, err := mockMapGeneralizer.Generalize(parent)
	require.NoError(t, err)
	mMap := m.(map[string]any)
	_, ok := mMap["class"]
	assert.False(t, ok)
	_, ok = mMap["child"].(map[string]any)["class"]
	assert.False(t, ok)

	// ensure Realize drops class if provided
	mMap["class"] = "org.apache.dubbo.mockParent"
	mMap["child"].(map[string]any)["class"] = "org.apache.dubbo.mockChild"

	r, err := mockMapGeneralizer.Realize(m, reflect.TypeFor[mockParent]())
	require.NoError(t, err)
	rParent, ok := r.(mockParent)
	assert.True(t, ok)
	assert.Equal(t, "xavierniu", rParent.Name)
	assert.Equal(t, 30, rParent.Age)
	assert.Equal(t, "lmc", rParent.Child.Name)
	assert.Equal(t, 20, rParent.Child.Age)
}
