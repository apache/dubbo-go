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

package mapping

import (
	"io/ioutil"
	"net/http"
	"testing"
	"time"
)

import (
	structpb "github.com/golang/protobuf/ptypes/struct"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/remoting/xds/common"
	"dubbo.apache.org/dubbo-go/v3/xds/client/mocks"
)

const (
	istioDebugPortFoo     = "38080"
	istiodDebugAddrStrFoo = "127.0.0.1:" + istioDebugPortFoo
	localPodServiceAddr   = "dubbo-go-app.default.svc.cluster.local:20000"
	serviceKey1           = "providers:api.Greeter::"
	serviceKey2           = "providers:grpc.reflection.v1alpha.ServerReflection::"
	metadataService1      = `{"` + serviceKey1 + `":"` + localPodServiceAddr + `"}`
	metadataService1And2  = `{"` + serviceKey1 + `":"` + localPodServiceAddr + `","` + serviceKey2 + `":"` + localPodServiceAddr + `"}`
	metadataService2      = `{"` + serviceKey2 + `":"` + localPodServiceAddr + `"}`
	metadataNoService     = `{}`
	istioTokenPathFoo     = "/tmp/dubbo-go-mesh-test-token"
	istioTokenFoo         = "mock-token"
)

func TestNewInterfaceMapHandler(t *testing.T) {
	mockXDSClient := &mocks.XDSClient{}
	interfaceMapHandler := NewInterfaceMapHandlerImpl(mockXDSClient, istioTokenPathFoo, common.NewHostNameOrIPAddr(istiodDebugAddrStrFoo), common.NewHostNameOrIPAddr(localPodServiceAddr), false)
	assert.NotNil(t, interfaceMapHandler)
}

func TestInterfaceMapHandlerRegisterAndUnregister(t *testing.T) {
	mockXDSClient := &mocks.XDSClient{}
	mockXDSClient.On("SetMetadata", mock.AnythingOfType("*structpb.Struct")).Return(nil)

	interfaceMapHandler := NewInterfaceMapHandlerImpl(mockXDSClient, istioTokenPathFoo, common.NewHostNameOrIPAddr(istiodDebugAddrStrFoo), common.NewHostNameOrIPAddr(localPodServiceAddr), false)

	assert.Nil(t, interfaceMapHandler.Register(serviceKey1))
	assert.Nil(t, interfaceMapHandler.Register(serviceKey2))
	assert.Nil(t, interfaceMapHandler.UnRegister(serviceKey1))
	assert.Nil(t, interfaceMapHandler.UnRegister(serviceKey2))

	mockXDSClient.AssertCalled(t, "SetMetadata", mock.MatchedBy(getMatchFunction(metadataService1)))
	mockXDSClient.AssertCalled(t, "SetMetadata", mock.MatchedBy(getMatchFunction(metadataService1And2)))
	mockXDSClient.AssertCalled(t, "SetMetadata", mock.MatchedBy(getMatchFunction(metadataService2)))
	mockXDSClient.AssertCalled(t, "SetMetadata", mock.MatchedBy(getMatchFunction(metadataNoService)))
}

func TestGetServiceUniqueKeyHostAddrMapFromPilot(t *testing.T) {
	mockXDSClient := &mocks.XDSClient{}
	interfaceMapHandler := NewInterfaceMapHandlerImpl(mockXDSClient, istioTokenPathFoo, common.NewHostNameOrIPAddr(istiodDebugAddrStrFoo), common.NewHostNameOrIPAddr(localPodServiceAddr), false)
	assert.Nil(t, generateMockToken())

	// 1. start mock http server
	http.HandleFunc("/debug/adsz", func(writer http.ResponseWriter, request *http.Request) {
		if len(request.Header[authorizationHeader]) != 1 {
			writer.Write([]byte(""))
			return
		}
		if request.Header[authorizationHeader][0] != istiodTokenPrefix+istioTokenFoo {
			writer.Write([]byte(""))
			return
		}
		writer.Write([]byte(debugAdszDataFoo))
	})
	go http.ListenAndServe(":"+istioDebugPortFoo, nil)
	time.Sleep(time.Second)

	// 2. get map from client
	hostAddr1, err := interfaceMapHandler.GetHostAddrMap(serviceKey1)
	assert.Nil(t, err)
	assert.Equal(t, localPodServiceAddr, hostAddr1)

	// 3. assert
	hostAddr2, err := interfaceMapHandler.GetHostAddrMap(serviceKey2)
	assert.Nil(t, err)
	assert.Equal(t, localPodServiceAddr, hostAddr2)

}

func getMatchFunction(metadata string) func(abc *structpb.Struct) bool {
	return func(abc *structpb.Struct) bool {
		metadatas, ok := abc.Fields[constant.XDSMetadataLabelsKey]
		if !ok {
			return false
		}
		subFields := metadatas.GetStructValue().GetFields()
		subValue, ok2 := subFields[constant.XDSMetadataDubboGoMapperKey]
		if !ok2 {
			return false
		}
		if subValue.GetStringValue() != metadata {
			return false
		}
		return true
	}
}

func generateMockToken() error {
	return ioutil.WriteFile(istioTokenPathFoo, []byte(istioTokenFoo), 0777)
}
