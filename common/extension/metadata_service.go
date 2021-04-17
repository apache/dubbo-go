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

package extension

import (
	"fmt"
)

import (
	perrors "github.com/pkg/errors"
)

import (
	"github.com/apache/dubbo-go/common/constant"
	"github.com/apache/dubbo-go/metadata/service"
)

var (
	metadataServiceInsMap = make(map[string]func() (service.MetadataService, error), 2)
)

// SetLocalMetadataService will store the msType => creator pair
func SetLocalMetadataService(key string, creator func() (service.MetadataService, error)) {
	metadataServiceInsMap[key] = creator
}

// GetMetadataService will create a inmemory MetadataService instance
func GetLocalMetadataService(key string) (service.MetadataService, error) {
	if key == "" {
		key = constant.DEFAULT_KEY
	}
	if creator, ok := metadataServiceInsMap[key]; ok {
		return creator()
	}
	return nil, perrors.New(fmt.Sprintf("could not find the metadata service creator for metadataType: local, please check whether you have imported relative packages, \n" +
		"local - github.com/apache/dubbo-go/metadata/service/inmemory, "))
}
