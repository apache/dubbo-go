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

package config

import "testing"

func TestNewServiceConfigBuilder(t *testing.T) {
	registryConfig := NewRegistryConfigWithProtocolDefaultPort("nacos")
	protocolConfig := NewProtocolConfigBuilder().
		SetName("dubbo").
		SetPort("20000").
		Build()

	serviceConfig := NewServiceConfigBuilder().
		SetRegistryIDs("nacos").
		SetProtocolIDs("dubbo").
		SetInterface("org.apache.dubbo").
		SetMetadataType("local").
		SetLoadBalancce("random").
		SetWarmUpTie("warmup").
		SetCluster("cluster").
		AddRCRegistry("nacos", registryConfig).
		AddRCProtocol("dubbo", protocolConfig).
		SetGroup("dubbo").
		SetVersion("1.0.0").SetProxyFactoryKey("defulte")
	Build()

	//serviceConfig.Init()
}
