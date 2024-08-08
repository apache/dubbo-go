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
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/metadata/service"
)

type localMetadataServiceCreatorV1 func() (service.MetadataServiceV1, error)

var (
	localMetadataServiceInsMapV1 = make(map[string]localMetadataServiceCreatorV1, 2)
)

// SetLocalMetadataService will store the msType => creator pair
func SetLocalMetadataServiceV1(key string, creator localMetadataServiceCreatorV1) {
	localMetadataServiceInsMapV1[key] = creator
}

// GetLocalMetadataService will create a local MetadataService instance
func GetLocalMetadataServiceV1(key string) (service.MetadataServiceV1, error) {
	if key == "" {
		key = constant.DefaultKey
	}
	if creator, ok := localMetadataServiceInsMapV1[key]; ok {
		return creator()
	}
	return nil, perrors.New(fmt.Sprintf("could not find the metadata service creator for metadataType: local, " +
		"please check whether you have imported relative packages, " +
		"local - dubbo.apache.org/dubbo-go/v3/metadata/service/local"))
}
