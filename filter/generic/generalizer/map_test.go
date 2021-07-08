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

type mockParent struct {
	Gender, Email, Name string
	Age int
	Child *mockChild
}

func (p mockParent) JavaClassName() string {
	return "org.apache.dubbo.mockParent"
}

type mockChild struct {
	Gender, Email, Name string
	Age int
}

func (c *mockChild) JavaClassName() string {
	return "org.apache.dubbo.mockChild"
}

func TestPOJOClassName(t *testing.T) {
	c := &mockChild{
		Age: 20,
		Gender: "male",
		Email: "lmc@example.com",
		Name: "lmc",
	}
	p := mockParent{
		Age: 30,
		Gender: "male",
		Email: "xavierniu@example.com",
		Name: "xavierniu",
		Child: c,
	}

	m := objToMap(p).(map[string]interface{})
	assert.Equal(t, "org.apache.dubbo.mockParent", m["class"].(string))
	assert.Equal(t, "org.apache.dubbo.mockChild", m["child"].(map[string]interface{})["class"].(string))
}

func TestPOJOArray(t *testing.T) {
	c1 := &mockChild{
		Age: 20,
		Gender: "male",
		Email: "lmc@example.com",
		Name: "lmc",
	}
	c2 := &mockChild{
		Age: 21,
		Gender: "male",
		Email: "lmc1@example.com",
		Name: "lmc1",
	}

	m := objToMap([]interface{}{c1, c2}).([]interface{})
	assert.Equal(t, "lmc", m[0].(map[string]interface{})["name"].(string))
	assert.Equal(t, "lmc1", m[1].(map[string]interface{})["name"].(string))
}

func TestNullField(t *testing.T) {
	p := mockParent{
		Age: 30,
		Gender: "male",
		Email: "xavierniu@example.com",
		Name: "xavierniu",
	}

	m := objToMap(p).(map[string]interface{})
	assert.Nil(t, m["child"])
}
