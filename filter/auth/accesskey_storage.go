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

package auth

import (
	"sync"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/common/extension"
	"dubbo.apache.org/dubbo-go/v3/filter"
	"dubbo.apache.org/dubbo-go/v3/protocol"
)

var (
	storageOnce sync.Once
	storage     *defaultAccesskeyStorage
)

func init() {
	extension.SetAccessKeyStorages(constant.DefaultAccessKeyStorage, newDefaultAccesskeyStorage)
}

// defaultAccesskeyStorage is the default implementation of AccesskeyStorage
type defaultAccesskeyStorage struct{}

func newDefaultAccesskeyStorage() filter.AccessKeyStorage {
	if storage == nil {
		storageOnce.Do(func() {
			storage = &defaultAccesskeyStorage{}
		})
	}
	return storage
}

// GetAccessKeyPair retrieves AccessKeyPair from url by the key "accessKeyId" and "secretAccessKey"
func (storage *defaultAccesskeyStorage) GetAccessKeyPair(invocation protocol.Invocation, url *common.URL) *filter.AccessKeyPair {
	return &filter.AccessKeyPair{
		AccessKey: url.GetParam(constant.AccessKeyIDKey, ""),
		SecretKey: url.GetParam(constant.SecretAccessKeyKey, ""),
	}
}
