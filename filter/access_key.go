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

package filter

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/protocol"
)

// AccessKeyPair stores the basic attributes for authentication.
type AccessKeyPair struct {
	AccessKey    string `yaml:"accessKey"   json:"accessKey,omitempty" property:"accessKey"`
	SecretKey    string `yaml:"secretKey"   json:"secretKey,omitempty" property:"secretKey"`
	ConsumerSide string `yaml:"consumerSide"   json:"consumerSide,omitempty" property:"consumerSide"`
	ProviderSide string `yaml:"providerSide"   json:"providerSide,omitempty" property:"providerSide"`
	Creator      string `yaml:"creator"   json:"creator,omitempty" property:"creator"`
	Options      string `yaml:"options"   json:"options,omitempty" property:"options"`
}

// AccessKeyStorage supports us to store our AccessKeyPair or load AccessKeyPair from other
// storage, such as filesystem.
type AccessKeyStorage interface {
	GetAccessKeyPair(protocol.Invocation, *common.URL) *AccessKeyPair
}
