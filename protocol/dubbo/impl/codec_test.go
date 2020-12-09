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
	tmpData := map[string]interface{}{
		"dubbo":     "2.0.2",
		"interface": "Service",
		"path":      "path",
		"timeout":   "1000",
		"version":   "2.6",
	}
	assert.Equal(t, tmpData, reassembleBody["attachments"])
}
