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

package identifier

import (
	"github.com/apache/dubbo-go/common/constant"
)

// ServiceMetadataIdentifier is inherit baseMetaIdentifier with service params: Revision and Protocol
type ServiceMetadataIdentifier struct {
	revision string
	protocol string
	BaseMetadataIdentifier
}

// GetIdentifierKey will return string format as service:Version:Group:Side:Protocol:"revision"+Revision
func (mdi *ServiceMetadataIdentifier) getIdentifierKey(params ...string) string {
	return mdi.BaseMetadataIdentifier.getIdentifierKey(mdi.protocol, constant.KEY_REVISON_PREFIX+mdi.revision)
}

// GetFilePathKey will return string format as metadata/path/Version/Group/Side/Protocol/"revision"+Revision
func (mdi *ServiceMetadataIdentifier) getFilePathKey(params ...string) string {
	return mdi.BaseMetadataIdentifier.getFilePathKey(mdi.protocol, constant.KEY_REVISON_PREFIX+mdi.revision)
}
