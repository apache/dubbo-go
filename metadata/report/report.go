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

package report

import (
	"encoding/json"
)

import (
	gxset "github.com/dubbogo/gost/container/set"
)

import (
	"dubbo.apache.org/dubbo-go/v3/metadata/info"
	"dubbo.apache.org/dubbo-go/v3/metadata/mapping"
)

// AppRevision represents a revision entry with its last modification time.
type AppRevision struct {
	Revision   string
	ModifyTime int64 // unix timestamp in milliseconds
}

// ParseMetadataLastUpdatedTime extracts lastUpdatedTime from a metadata JSON blob.
func ParseMetadataLastUpdatedTime(data []byte) int64 {
	var meta info.MetadataInfo
	if err := json.Unmarshal(data, &meta); err != nil {
		return 0
	}
	return meta.LastUpdatedTime
}

// MetadataReport is an interface of remote metadata report.
type MetadataReport interface {
	// GetAppMetadata get metadata info from report
	GetAppMetadata(application, revision string) (*info.MetadataInfo, error)

	// PublishAppMetadata publish metadata info to report
	PublishAppMetadata(application, revision string, info *info.MetadataInfo) error

	// RegisterServiceAppMapping map the specified Dubbo service interface to current Dubbo app name
	RegisterServiceAppMapping(interfaceName, group string, application string) error

	// GetServiceAppMapping get the app names from the specified Dubbo service interface
	GetServiceAppMapping(interfaceName, group string, l mapping.MappingListener) (*gxset.HashSet, error)

	// RemoveServiceAppMappingListener remove the serviceMapping listener by key and group
	RemoveServiceAppMappingListener(interfaceName, group string) error

	// UnPublishAppMetadata removes metadata for a specific revision from the metadata center.
	// This operation is idempotent — deleting a non-existent revision should not return an error.
	UnPublishAppMetadata(application, revision string) error

	// ListAppRevisions lists all stored revisions for an application.
	// Each AppRevision contains the revision string and the last modification time (unix ms).
	ListAppRevisions(application string) ([]AppRevision, error)
}
