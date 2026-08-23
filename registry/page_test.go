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
	"testing"
)

import (
	"github.com/stretchr/testify/assert"
)

func TestNewPage_ExactDivision(t *testing.T) {
	data := []any{1, 2, 3, 4, 5}
	p := NewPage(0, 5, data, 10)

	assert.Equal(t, 0, p.GetOffset())
	assert.Equal(t, 5, p.GetPageSize())
	assert.Equal(t, 2, p.GetTotalPages())
	assert.Equal(t, data, p.GetData())
	assert.Equal(t, 5, p.GetDataSize())
	assert.True(t, p.HasData())
	assert.True(t, p.HasNext())
}

func TestNewPage_RemainderRoundsUpTotalPages(t *testing.T) {
	data := []any{9, 10}
	p := NewPage(8, 4, data, 10)

	assert.Equal(t, 3, p.GetTotalPages())
	assert.Equal(t, 2, p.GetDataSize())
	// 10 - 8 - 4 < 0, so this is the last page
	assert.False(t, p.HasNext())
}

func TestNewPage_LastPageHasNoNext(t *testing.T) {
	data := []any{6, 7, 8, 9, 10}
	p := NewPage(5, 5, data, 10)

	assert.Equal(t, 2, p.GetTotalPages())
	assert.Equal(t, 5, p.GetDataSize())
	assert.False(t, p.HasNext())
}

func TestNewPage_EmptyData(t *testing.T) {
	p := NewPage(0, 5, nil, 0)

	assert.Equal(t, 0, p.GetTotalPages())
	assert.Equal(t, 0, p.GetDataSize())
	assert.False(t, p.HasData())
	assert.False(t, p.HasNext())
}

func TestNewPage_RequestOffsetBeyondTotal(t *testing.T) {
	p := NewPage(100, 5, nil, 10)

	assert.Equal(t, 2, p.GetTotalPages())
	assert.False(t, p.HasNext())
}
