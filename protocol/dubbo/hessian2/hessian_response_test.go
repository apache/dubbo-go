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

package hessian2

import (
	"reflect"
	"testing"
)

import (
	hessian "github.com/apache/dubbo-go-hessian2"
	"github.com/stretchr/testify/assert"
)

func doTestReflectResponse(t *testing.T, in interface{}, out interface{}) {
	err := ReflectResponse(in, out)
	if err != nil {
		t.Error(err)
		t.FailNow()
	}

	result := hessian.UnpackPtrValue(reflect.ValueOf(out)).Interface()

	equal := reflect.DeepEqual(in, result)
	if !equal {
		t.Errorf("expect [%v]: %v, but got [%v]: %v", reflect.TypeOf(in), in, reflect.TypeOf(result), result)
	}
}

func TestReflectResponse(t *testing.T) {
	var b bool
	doTestReflectResponse(t, true, &b)
	doTestReflectResponse(t, false, &b)

	var i int
	doTestReflectResponse(t, 123, &i)
	doTestReflectResponse(t, 234, &i)

	var i16 int16
	doTestReflectResponse(t, int16(456), &i16)

	var i64 int64
	doTestReflectResponse(t, int64(789), &i64)

	var s string
	doTestReflectResponse(t, "hello world", &s)

	type rr struct {
		Name string
		Num  int
	}

	var r1 rr
	doTestReflectResponse(t, rr{"dubbogo", 32}, &r1)

	// ------ map test -------
	m1 := make(map[interface{}]interface{})
	var m1r map[interface{}]interface{}
	m1["hello"] = "world"
	m1[1] = "go"
	m1["dubbo"] = 666
	doTestReflectResponse(t, m1, &m1r)

	m2 := make(map[string]string)
	var m2r map[string]string
	m2["hello"] = "world"
	m2["dubbo"] = "666"
	doTestReflectResponse(t, m2, &m2r)

	m3 := make(map[string]rr)
	var m3r map[string]rr
	m3["dubbo"] = rr{"hello", 123}
	m3["go"] = rr{"world", 456}
	doTestReflectResponse(t, m3, &m3r)

	// ------ slice test -------
	s1 := []string{"abc", "def", "hello", "world"}
	var s1r []string
	doTestReflectResponse(t, s1, &s1r)

	s2 := []rr{rr{"dubbo", 666}, rr{"go", 999}}
	var s2r []rr
	doTestReflectResponse(t, s2, &s2r)

	s3 := []interface{}{rr{"dubbo", 666}, 123, "hello"}
	var s3r []interface{}
	doTestReflectResponse(t, s3, &s3r)

	// ------ interface test -------
	in1 := []interface{}{rr{"dubbo", 666}, 123, "hello"}
	var inr1 *interface{}
	doTestReflectResponse(t, in1, reflect.New(reflect.TypeOf(inr1).Elem()).Interface())

	in2 := make(map[string]rr)
	var inr2 map[string]rr
	m3["dubbo"] = rr{"hello", 123}
	m3["go"] = rr{"world", 456}
	doTestReflectResponse(t, in2, &inr2)
}

// separately test copy normal map to map[interface{}]interface{}
func TestCopyMap(t *testing.T) {
	type rr struct {
		Name string
		Num  int
	}

	m3 := make(map[string]rr)
	var m3r map[interface{}]interface{}
	r1 := rr{"hello", 123}
	r2 := rr{"world", 456}
	m3["dubbo"] = r1
	m3["go"] = r2

	err := ReflectResponse(m3, &m3r)
	if err != nil {
		t.Error(err)
		t.FailNow()
	}

	assert.Equal(t, 2, len(m3r))

	rr1, ok := m3r["dubbo"]
	assert.True(t, ok)
	assert.True(t, reflect.DeepEqual(r1, rr1))

	rr2, ok := m3r["go"]
	assert.True(t, ok)
	assert.True(t, reflect.DeepEqual(r2, rr2))
}

// separately test copy normal slice to []interface{}
func TestCopySlice(t *testing.T) {
	type rr struct {
		Name string
		Num  int
	}

	r1 := rr{"hello", 123}
	r2 := rr{"world", 456}

	s1 := []rr{r1, r2}
	var s1r []interface{}

	err := ReflectResponse(s1, &s1r)
	if err != nil {
		t.Error(err)
		t.FailNow()
	}

	assert.Equal(t, 2, len(s1r))
	assert.True(t, reflect.DeepEqual(r1, s1r[0]))
	assert.True(t, reflect.DeepEqual(r2, s1r[1]))
}

func TestIsSupportResponseAttachment(t *testing.T) {
	is := isSupportResponseAttachment("2.X")
	assert.False(t, is)

	is = isSupportResponseAttachment("2.0.10")
	assert.False(t, is)

	is = isSupportResponseAttachment("2.5.3")
	assert.False(t, is)

	is = isSupportResponseAttachment("2.6.2")
	assert.False(t, is)

	is = isSupportResponseAttachment("1.5.5")
	assert.False(t, is)

	is = isSupportResponseAttachment("0.0.0")
	assert.False(t, is)

	is = isSupportResponseAttachment("2.0.2")
	assert.True(t, is)

	is = isSupportResponseAttachment("2.7.2")
	assert.True(t, is)
}

func TestVersion2Int(t *testing.T) {
	v := version2Int("2.1.3")
	assert.Equal(t, 2010300, v)

	v = version2Int("22.11.33")
	assert.Equal(t, 22113300, v)

	v = version2Int("222.111.333")
	assert.Equal(t, 223143300, v)

	v = version2Int("220.110.333")
	assert.Equal(t, 221133300, v)

	v = version2Int("229.119.333")
	assert.Equal(t, 230223300, v)

	v = version2Int("2222.1111.3333")
	assert.Equal(t, 2233443300, v)

	v = version2Int("2.11")
	assert.Equal(t, 211, v)

	v = version2Int("2.1.3.4")
	assert.Equal(t, 2010304, v)

	v = version2Int("2.1.3.4.5")
	assert.Equal(t, 201030405, v)

}
