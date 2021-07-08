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

package generic

import (
	"context"
	"net/url"
	"reflect"
	"testing"
	"time"
)

import (
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/protocol"
	"dubbo.apache.org/dubbo-go/v3/protocol/invocation"
	"dubbo.apache.org/dubbo-go/v3/protocol/mock"
)

// test isCallingToGenericService branch
func TestInvoke(t *testing.T) {
	invokeUrl := common.NewURLWithOptions(
		common.WithParams(url.Values{}),
		common.WithParamsValue(constant.GENERIC_KEY, constant.GENERIC_SERIALIZATION_DEFAULT))
	filter := &Filter{}

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	normalInvocation := invocation.NewRPCInvocation("hello", []interface{}{"arg1"}, make(map[string]interface{}))

	mockInvoker := mock.NewMockInvoker(ctrl)
	mockInvoker.EXPECT().GetUrl().Return(invokeUrl).Times(2)
	mockInvoker.EXPECT().Invoke(gomock.Not(normalInvocation)).DoAndReturn(
		func(invocation protocol.Invocation) protocol.Result {
			assert.Equal(t, constant.GENERIC, invocation.MethodName())
			args := invocation.Arguments()
			assert.Equal(t, "hello", args[0])
			assert.Equal(t, "java.lang.String", args[1].([]interface{})[0].(string))
			assert.Equal(t, "arg1", args[2].([]interface{})[0].(string))
			assert.Equal(t, constant.GENERIC_SERIALIZATION_DEFAULT, invocation.AttachmentsByKey(constant.GENERIC_KEY, ""))
			return &protocol.RPCResult{}
		})

	result := filter.Invoke(context.Background(), mockInvoker, normalInvocation)
	assert.NotNil(t, result)
}

// test isMakingAGenericCall branch
func TestInvokeWithGenericCall(t *testing.T) {
	invokeUrl := common.NewURLWithOptions(
		common.WithParams(url.Values{}),
		common.WithParamsValue(constant.GENERIC_KEY, constant.GENERIC_SERIALIZATION_DEFAULT))
	filter := &Filter{}

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	genericInvocation := invocation.NewRPCInvocation(constant.GENERIC, []interface{}{
		"hello",
		[]string{"java.lang.String"},
		[]string{"arg1"},
	}, make(map[string]interface{}))

	mockInvoker := mock.NewMockInvoker(ctrl)
	mockInvoker.EXPECT().GetUrl().Return(invokeUrl).Times(3)
	mockInvoker.EXPECT().Invoke(gomock.Any()).DoAndReturn(
		func(invocation protocol.Invocation) protocol.Result {
			assert.Equal(t, constant.GENERIC, invocation.MethodName())
			args := invocation.Arguments()
			assert.Equal(t, "hello", args[0])
			assert.Equal(t, "java.lang.String", args[1].([]string)[0])
			assert.Equal(t, "arg1", args[2].([]string)[0])
			assert.Equal(t, constant.GENERIC_SERIALIZATION_DEFAULT, invocation.AttachmentsByKey(constant.GENERIC_KEY, ""))
			return &protocol.RPCResult{}
		})

	result := filter.Invoke(context.Background(), mockInvoker, genericInvocation)
	assert.NotNil(t, result)
}

func TestObjToMap(t *testing.T) {
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
		DaDa time.Time
		EeEe int
	}
	testData.AaAa = "1"
	testData.BaBa = "1"
	testData.CaCa.BaBa = "2"
	testData.CaCa.AaAa = "2"
	testData.CaCa.XxYy.xxXx = "3"
	testData.CaCa.XxYy.Xx = "3"
	testData.DaDa = time.Date(2020, 10, 29, 2, 34, 0, 0, time.Local)
	testData.EeEe = 100
	m := objToMap(testData).(map[string]interface{})
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
