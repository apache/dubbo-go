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

package dubbo3

import (
	"context"
	"reflect"
	"testing"
	"time"
)

import (
	hessian "github.com/apache/dubbo-go-hessian2"
	"github.com/stretchr/testify/assert"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/protocol"
)

const (
	mockDubbo3CommonUrl = "tri://127.0.0.1:20002/DubboGreeterImpl?accesslog=&anyhost=true&app.version=0.0.1&application=BDTService&async=false&bean.name=DubboGreeterImpl" +
		"&category=providers&cluster=failover&dubbo=dubbo-provider-golang-2.6.0&environment=dev&execute.limit=&execute.limit.rejected.handler=&generic=false&group=&interface=org.apache.dubbo.DubboGreeterImpl" +
		"&ip=192.168.1.106&loadbalance=random&methods.SayHello.loadbalance=random&methods.SayHello.retries=1&methods.SayHello.tps.limit.interval=&methods.SayHello.tps.limit.rate=&methods.SayHello.tps.limit.strategy=" +
		"&methods.SayHello.weight=0&module=dubbogo+say-hello+client&name=BDTService&organization=ikurento.com&owner=ZX&pid=49427&reference.filter=cshutdown&registry.role=3&remote.timestamp=1576923717&retries=" +
		"&service.filter=echo%2Ctoken%2Caccesslog%2Ctps%2Cexecute%2Cpshutdown&side=provider&timestamp=1576923740&tps.limit.interval=&tps.limit.rate=&tps.limit.rejected.handler=&tps.limit.strategy=&tps.limiter=&version=&warmup=100!"
)

func TestDubboProtocolExport(t *testing.T) {
	// Export
	addService()

	proto := GetProtocol()
	url, err := common.NewURL(mockDubbo3CommonUrl)
	assert.NoError(t, err)
	exporter := proto.Export(protocol.NewBaseInvoker(url))
	time.Sleep(time.Second)

	// make sure url
	eq := exporter.GetInvoker().GetURL().URLEqual(url)
	assert.True(t, eq)

	// make sure exporterMap after 'Unexport'
	_, ok := proto.(*DubboProtocol).ExporterMap().Load(url.ServiceKey())
	assert.True(t, ok)
	exporter.Unexport()
	_, ok = proto.(*DubboProtocol).ExporterMap().Load(url.ServiceKey())
	assert.False(t, ok)

	// make sure serverMap after 'Destroy'
	_, ok = proto.(*DubboProtocol).serverMap[url.Location]
	assert.True(t, ok)
	proto.Destroy()
	_, ok = proto.(*DubboProtocol).serverMap[url.Location]
	assert.False(t, ok)
}

func TestDubboProtocolRefer(t *testing.T) {
	proto := GetProtocol()
	url, err := common.NewURL(mockDubbo3CommonUrl)
	assert.NoError(t, err)
	invoker := proto.Refer(url)

	// make sure url
	eq := invoker.GetURL().URLEqual(url)
	assert.True(t, eq)

	// make sure invokers after 'Destroy'
	invokersLen := len(proto.(*DubboProtocol).Invokers())
	assert.Equal(t, 1, invokersLen)
	proto.Destroy()
	invokersLen = len(proto.(*DubboProtocol).Invokers())
	assert.Equal(t, 0, invokersLen)
}

type MockUser struct {
	Name string
}

func (m *MockUser) JavaClassName() string {
	return "mockuser"
}

type MockService struct {
}

func (m *MockService) GetUser(ctx context.Context, user, user2 *MockUser) (*MockUser, error) {
	return user, nil
}

func TestDubbo3UnaryService_GetReqParamsInterfaces(t *testing.T) {
	hessian.RegisterPOJO(&MockUser{})
	srv := UnaryService{}
	valueOf := reflect.ValueOf(&MockService{})
	typeOf := valueOf.Type()
	numField := valueOf.NumMethod()
	for i := 0; i < numField; i++ {
		ft := typeOf.Method(i)
		// num in/out is checked in common/rpc_service.go
		typs := make([]reflect.Type, 0)
		for j := 2; j < ft.Type.NumIn(); j++ {
			typs = append(typs, ft.Type.In(j))
		}
		srv.setReqParamsTypes("GetUser", typs)
	}
	paramsInterfaces, ok := srv.GetReqParamsInterfaces("GetUser")
	assert.True(t, ok)
	enc := hessian.NewEncoder()
	err := enc.Encode(&MockUser{
		Name: "laurence",
	})
	assert.Nil(t, err)
	data := enc.Buffer()
	decoder := hessian.NewDecoder(data)
	val, err := decoder.Decode()
	assert.Nil(t, err)
	assert.Equal(t, 2, len(paramsInterfaces))
	subTest(t, val, paramsInterfaces)
	args := make([]interface{}, 0, 1)
	for _, v := range paramsInterfaces {
		tempParamObj := reflect.ValueOf(v).Elem().Interface()
		args = append(args, tempParamObj)
	}
	assert.Equal(t, "laurence", args[0].(*MockUser).Name)
	assert.Equal(t, "laurence", args[1].(*MockUser).Name)
}

func subTest(t *testing.T, val, paramsInterfaces interface{}) {
	list := paramsInterfaces.([]interface{})
	for k, _ := range list {
		err := hessian.ReflectResponse(val, list[k])
		assert.Nil(t, err)
	}
}
