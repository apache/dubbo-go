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
	"testing"

	"github.com/stretchr/testify/assert"
)

var serviceMetadataId = &ServiceMetadataIdentifier{
	Revision: "1.0",
	Protocol: "dubbo",
	BaseMetadataIdentifier: BaseMetadataIdentifier{
		ServiceInterface: "org.apache.pkg.mockService",
		Version:          "1.0.0",
		Group:            "Group",
		Side:             "provider",
	},
}

func TestServiceGetFilePathKey(t *testing.T) {
	assert.Equal(t, "metadata/org.apache.pkg.mockService/1.0.0/Group/provider/dubbo/revision1.0", serviceMetadataId.GetFilePathKey())
}

func TestServiceGetIdentifierKey(t *testing.T) {
	assert.Equal(t, "org.apache.pkg.mockService:1.0.0:Group:provider:dubbo:revision1.0", serviceMetadataId.GetIdentifierKey())
}
