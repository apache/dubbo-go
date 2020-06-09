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

// SubscriberMetadataIdentifier is inherit baseMetaIdentifier with service params: Revision
type SubscriberMetadataIdentifier struct {
	revision string
	BaseMetadataIdentifier
}

// GetIdentifierKey will return string format is service:Version:Group:Side:Revision
func (mdi *SubscriberMetadataIdentifier) getIdentifierKey(params ...string) string {
	return mdi.BaseMetadataIdentifier.getIdentifierKey(mdi.revision)
}

// GetFilePathKey will return string format is metadata/path/Version/Group/Side/Revision
func (mdi *SubscriberMetadataIdentifier) getFilePathKey(params ...string) string {
	return mdi.BaseMetadataIdentifier.getFilePathKey(mdi.revision)
}
