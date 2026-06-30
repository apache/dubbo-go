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

package metadata

// MetadataErrorKind identifies the metadata loading stage that failed.
type MetadataErrorKind string

const (
	MetadataErrorKindReportLoad MetadataErrorKind = "metadata_report_load"
	MetadataErrorKindRPCLoad    MetadataErrorKind = "rpc_metadata_load"
	MetadataErrorKindURLBuild   MetadataErrorKind = "metadata_url_build"
	MetadataErrorKindNil        MetadataErrorKind = "metadata_nil"
)

// MetadataError carries categorizable metadata loading failure context.
type MetadataError struct {
	Kind        MetadataErrorKind
	Source      string
	App         string
	Revision    string
	RegistryID  string
	StorageType string
	Err         error
}

func (e *MetadataError) Error() string {
	if e == nil {
		return ""
	}
	if e.Err != nil {
		return e.Err.Error()
	}
	return string(e.Kind)
}

func (e *MetadataError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Err
}

func withMetadataErrorContext(err error, fallbackKind MetadataErrorKind, source string, app string, revision string, storageType string) error {
	if err == nil {
		return nil
	}
	if metadataErr, ok := err.(*MetadataError); ok {
		next := *metadataErr
		if next.Kind == "" {
			next.Kind = fallbackKind
		}
		if next.Source == "" {
			next.Source = source
		}
		if next.App == "" {
			next.App = app
		}
		if next.Revision == "" {
			next.Revision = revision
		}
		if next.StorageType == "" {
			next.StorageType = storageType
		}
		return &next
	}
	return &MetadataError{
		Kind:        fallbackKind,
		Source:      source,
		App:         app,
		Revision:    revision,
		StorageType: storageType,
		Err:         err,
	}
}
