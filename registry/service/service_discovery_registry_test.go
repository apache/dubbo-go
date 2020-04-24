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

package service

import (
	"testing"
)

var (
	SERVICE_INTERFACE = "org.apache.dubbo.metadata.MetadataService"
	GROUP             = "dubbo-provider"
	VERSION           = "1.0.0"
)

func TestServiceDiscoveryRegistry_Register(t *testing.T) {
	//registryURL,_:=common.NewURL("in-memory://localhost:12345",
	//	common.WithParamsValue("registry-type","service"),
	//	common.WithParamsValue("subscribed-services","a, b , c,d,e ,"))
	//url,_:=common.NewURL("dubbo://192.168.0.102:20880/"+ SERVICE_INTERFACE +
	//	"?&application=" + GROUP +
	//	"&interface=" + SERVICE_INTERFACE +
	//	"&group=" + GROUP +
	//	"&version=" + VERSION +
	//	"&methods=getAllServiceKeys,getServiceRestMetadata,getExportedURLs,getAllExportedURLs" +
	//	"&side=provider")
	//registry,err:=newServiceDiscoveryRegistry(&registryURL)
	//if err!=nil{
	//	logger.Errorf("create service discovery registry catch error:%s",err.Error())
	//}
	//assert.Nil(t,err)
	//assert.NotNil(t,registry)
	//registry.Register(url)

}
