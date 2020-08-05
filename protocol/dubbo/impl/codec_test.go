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
	"testing"
	"time"
)

import (
	"github.com/stretchr/testify/assert"
)

import (
	"github.com/apache/dubbo-go/common/constant"
)

func TestDubboPackage_MarshalAndUnmarshal(t *testing.T) {
	pkg := NewDubboPackage(nil)
	pkg.Body = []interface{}{"a"}
	pkg.Header.Type = PackageHeartbeat
	pkg.Header.SerialID = constant.S_Hessian2
	pkg.Header.ID = 10086
	pkg.SetSerializer(HessianSerializer{})

	// heartbeat
	data, err := pkg.Marshal()
	assert.NoError(t, err)

	pkgres := NewDubboPackage(data)
	pkgres.SetSerializer(HessianSerializer{})

	pkgres.Body = []interface{}{}
	err = pkgres.Unmarshal()
	assert.NoError(t, err)
	assert.Equal(t, PackageHeartbeat|PackageRequest|PackageRequest_TwoWay, pkgres.Header.Type)
	assert.Equal(t, constant.S_Hessian2, pkgres.Header.SerialID)
	assert.Equal(t, int64(10086), pkgres.Header.ID)
	assert.Equal(t, 0, len(pkgres.Body.([]interface{})))

	// request
	pkg.Header.Type = PackageRequest
	pkg.Service.Interface = "Service"
	pkg.Service.Path = "path"
	pkg.Service.Version = "2.6"
	pkg.Service.Method = "Method"
	pkg.Service.Timeout = time.Second
	data, err = pkg.Marshal()
	assert.NoError(t, err)

	pkgres = NewDubboPackage(data)
	pkgres.SetSerializer(HessianSerializer{})
	pkgres.Body = make([]interface{}, 7)
	err = pkgres.Unmarshal()
	reassembleBody := pkgres.GetBody().(map[string]interface{})
	assert.NoError(t, err)
	assert.Equal(t, PackageRequest, pkgres.Header.Type)
	assert.Equal(t, constant.S_Hessian2, pkgres.Header.SerialID)
	assert.Equal(t, int64(10086), pkgres.Header.ID)
	assert.Equal(t, "2.0.2", reassembleBody["dubboVersion"].(string))
	assert.Equal(t, "path", pkgres.Service.Path)
	assert.Equal(t, "2.6", pkgres.Service.Version)
	assert.Equal(t, "Method", pkgres.Service.Method)
	assert.Equal(t, "Ljava/lang/String;", reassembleBody["argsTypes"].(string))
	assert.Equal(t, []interface{}{"a"}, reassembleBody["args"])
	assert.Equal(t, map[string]string{"dubbo": "2.0.2", "interface": "Service", "path": "path", "timeout": "1000", "version": "2.6"}, reassembleBody["attachments"].(map[string]string))
}

//func TestDubboPackage_Protobuf_Serialization_Request(t *testing.T) {
//	pkg := NewDubboPackage(nil)
//	pkg.Body = []interface{}{"a"}
//	pkg.Header.Type = PackageHeartbeat
//	pkg.Header.SerialID = constant.S_Proto
//	pkg.Header.ID = 10086
//	pkg.SetSerializer(ProtoSerializer{})
//
//	// heartbeat
//	data, err := pkg.Marshal()
//	assert.NoError(t, err)
//
//	pkgres := NewDubboPackage(data)
//	pkgres.SetSerializer(HessianSerializer{})
//
//	pkgres.Body = []interface{}{}
//	err = pkgres.Unmarshal()
//	assert.NoError(t, err)
//	assert.Equal(t, PackageHeartbeat|PackageRequest|PackageRequest_TwoWay, pkgres.Header.Type)
//	assert.Equal(t, constant.S_Proto, pkgres.Header.SerialID)
//	assert.Equal(t, int64(10086), pkgres.Header.ID)
//	assert.Equal(t, 0, len(pkgres.Body.([]interface{})))
//
//	// request
//	pkg.Header.Type = PackageRequest
//	pkg.Service.Interface = "Service"
//	pkg.Service.Path = "path"
//	pkg.Service.Version = "2.6"
//	pkg.Service.Method = "Method"
//	pkg.Service.Timeout = time.Second
//	pkg.SetBody([]interface{}{&pb.StringValue{Value: "hello world"}})
//	data, err = pkg.Marshal()
//	assert.NoError(t, err)
//
//	pkgres = NewDubboPackage(data)
//	pkgres.SetSerializer(ProtoSerializer{})
//	err = pkgres.Unmarshal()
//	assert.NoError(t, err)
//	body, ok := pkgres.Body.(map[string]interface{})
//	assert.Equal(t, ok, true)
//	req, ok := body["args"].([]interface{})
//	assert.Equal(t, ok, true)
//	// protobuf rpc just has exact one parameter
//	assert.Equal(t, len(req), 1)
//	argsBytes, ok := req[0].([]byte)
//	assert.Equal(t, true, ok)
//	sv := pb.StringValue{}
//	buf := proto.NewBuffer(argsBytes)
//	err = buf.Unmarshal(&sv)
//	assert.NoError(t, err)
//	assert.Equal(t, "hello world", sv.Value)
//}

//func TestDubboCodec_Protobuf_Serialization_Response(t *testing.T) {
//	{
//		pkg := NewDubboPackage(nil)
//		pkg.Header.Type = PackageResponse
//		pkg.Header.SerialID = constant.S_Proto
//		pkg.Header.ID = 10086
//		pkg.SetSerializer(ProtoSerializer{})
//		pkg.SetBody(&pb.StringValue{Value: "hello world"})
//
//		// heartbeat
//		data, err := pkg.Marshal()
//		assert.NoError(t, err)
//
//		pkgres := NewDubboPackage(data)
//		pkgres.SetSerializer(ProtoSerializer{})
//
//		pkgres.SetBody(&pb.StringValue{})
//		err = pkgres.Unmarshal()
//
//		assert.NoError(t, err)
//		assert.Equal(t, pkgres.Header.Type, PackageResponse)
//		assert.Equal(t, constant.S_Proto, pkgres.Header.SerialID)
//		assert.Equal(t, int64(10086), pkgres.Header.ID)
//
//		res, ok := pkgres.Body.(*pb.StringValue)
//		assert.Equal(t, ok, true)
//		assert.Equal(t, res.Value, "hello world")
//	}
//
//	// with attachments
//	{
//		attas := make(map[string]string)
//		attas["k1"] = "test"
//		resp := NewResponsePayload(&pb.StringValue{Value: "attachments"}, nil, attas)
//		p := NewDubboPackage(nil)
//		p.Header.Type = PackageResponse
//		p.Header.SerialID = constant.S_Proto
//		p.SetSerializer(ProtoSerializer{})
//		p.SetBody(resp)
//		data, err := p.Marshal()
//		assert.NoError(t, err)
//
//		pkgres := NewDubboPackage(data)
//		pkgres.Header.Type = PackageResponse
//		pkgres.Header.SerialID = constant.S_Proto
//		pkgres.Header.ID = 10086
//		pkgres.SetSerializer(ProtoSerializer{})
//
//		resAttachment := make(map[string]string)
//		resBody := &pb.StringValue{}
//		pkgres.SetBody(NewResponsePayload(resBody, nil, resAttachment))
//
//		err = pkgres.Unmarshal()
//		assert.NoError(t, err)
//		assert.Equal(t, "attachments", resBody.Value)
//		assert.Equal(t, "test", resAttachment["k1"])
//	}
//
//}
