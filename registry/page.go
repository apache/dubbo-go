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

package registry

import (
	gxpage "github.com/dubbogo/gost/hash/page"
)

// Pager is an alias of gxpage.Pager from github.com/dubbogo/gost/hash/page.
// It is kept as a type alias so that the signatures of the exported
// ServiceDiscovery pagination methods stay source-compatible for external
// implementors across the v3.x line.
type Pager = gxpage.Pager

// NewPage will create a Pager instance.
func NewPage(requestOffset int, pageSize int, data []any, totalSize int) Pager {
	return gxpage.NewPage(requestOffset, pageSize, data, totalSize)
}
