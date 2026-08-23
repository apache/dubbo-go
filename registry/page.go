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

// Pager is the abstraction for pagination usage.
// It is the domain interface of dubbo-go registry, replacing the previous
// gxpage.Pager from github.com/dubbogo/gost/hash/page.
type Pager interface {

	// GetOffset will return the offset
	GetOffset() int

	// GetPageSize will return the page size
	GetPageSize() int

	// GetTotalPages will return the number of total pages
	GetTotalPages() int

	// GetData will return the data
	GetData() []interface{}

	// GetDataSize will return the size of data.
	// Usually it's len(GetData())
	GetDataSize() int

	// HasNext will return whether has next page
	HasNext() bool

	// HasData will return whether this page has data.
	HasData() bool
}

// page is the default implementation of Pager interface.
type page struct {
	requestOffset int
	pageSize      int
	totalSize     int
	data          []interface{}
	totalPages    int
	hasNext       bool
}

// GetOffset will return the offset
func (d *page) GetOffset() int {
	return d.requestOffset
}

// GetPageSize will return the page size
func (d *page) GetPageSize() int {
	return d.pageSize
}

// GetTotalPages will return the number of total pages
func (d *page) GetTotalPages() int {
	return d.totalPages
}

// GetData will return the data
func (d *page) GetData() []interface{} {
	return d.data
}

// GetDataSize will return the size of data.
// it's len(GetData())
func (d *page) GetDataSize() int {
	return len(d.GetData())
}

// HasNext will return whether has next page
func (d *page) HasNext() bool {
	return d.hasNext
}

// HasData will return whether this page has data.
func (d *page) HasData() bool {
	return d.GetDataSize() > 0
}

// NewPage will create a Pager instance.
func NewPage(requestOffset int, pageSize int, data []interface{}, totalSize int) Pager {
	remain := totalSize % pageSize
	totalPages := totalSize / pageSize
	if remain > 0 {
		totalPages++
	}

	hasNext := totalSize-requestOffset-pageSize > 0

	return &page{
		requestOffset: requestOffset,
		pageSize:      pageSize,
		data:          data,
		totalSize:     totalSize,
		totalPages:    totalPages,
		hasNext:       hasNext,
	}
}
