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

package router

import (
	"testing"
)

import (
	"github.com/stretchr/testify/assert"
)

import (
	"dubbo.apache.org/dubbo-go/v3/global"
)

func TestDefaultOptions(t *testing.T) {
	opts := defaultOptions()
	assert.NotNil(t, opts)
	assert.NotNil(t, opts.Router)
}

func TestNewOptionsWithoutArgs(t *testing.T) {
	opts := NewOptions()
	assert.NotNil(t, opts)
	assert.NotNil(t, opts.Router)
}

func TestWithScope(t *testing.T) {
	scope := "test-scope"
	opts := NewOptions(WithScope(scope))
	assert.Equal(t, scope, opts.Router.Scope)
}

func TestWithKey(t *testing.T) {
	key := "test-key"
	opts := NewOptions(WithKey(key))
	assert.Equal(t, key, opts.Router.Key)
}

func TestWithForce(t *testing.T) {
	force := true
	opts := NewOptions(WithForce(force))
	assert.NotNil(t, opts.Router.Force)
	assert.Equal(t, force, *opts.Router.Force)

	force = false
	opts = NewOptions(WithForce(force))
	assert.NotNil(t, opts.Router.Force)
	assert.Equal(t, force, *opts.Router.Force)
}

func TestWithRuntime(t *testing.T) {
	runtime := true
	opts := NewOptions(WithRuntime(runtime))
	assert.NotNil(t, opts.Router.Runtime)
	assert.Equal(t, runtime, *opts.Router.Runtime)

	runtime = false
	opts = NewOptions(WithRuntime(runtime))
	assert.NotNil(t, opts.Router.Runtime)
	assert.Equal(t, runtime, *opts.Router.Runtime)
}

func TestWithEnabled(t *testing.T) {
	enabled := true
	opts := NewOptions(WithEnabled(enabled))
	assert.NotNil(t, opts.Router.Enabled)
	assert.Equal(t, enabled, *opts.Router.Enabled)

	enabled = false
	opts = NewOptions(WithEnabled(enabled))
	assert.NotNil(t, opts.Router.Enabled)
	assert.Equal(t, enabled, *opts.Router.Enabled)
}

func TestWithValid(t *testing.T) {
	valid := true
	opts := NewOptions(WithValid(valid))
	assert.NotNil(t, opts.Router.Valid)
	assert.Equal(t, valid, *opts.Router.Valid)

	valid = false
	opts = NewOptions(WithValid(valid))
	assert.NotNil(t, opts.Router.Valid)
	assert.Equal(t, valid, *opts.Router.Valid)
}

func TestWithPriority(t *testing.T) {
	priority := 100
	opts := NewOptions(WithPriority(priority))
	assert.Equal(t, priority, opts.Router.Priority)
}

func TestWithConditions(t *testing.T) {
	conditions := []string{"condition1", "condition2", "condition3"}
	opts := NewOptions(WithConditions(conditions))
	assert.Equal(t, conditions, opts.Router.Conditions)
}

func TestWithTags(t *testing.T) {
	tags := []global.Tag{
		{Name: "tag1"},
		{Name: "tag2"},
	}
	opts := NewOptions(WithTags(tags))
	assert.Equal(t, tags, opts.Router.Tags)
}

func TestMultipleOptions(t *testing.T) {
	scope := "multi-scope"
	key := "multi-key"
	priority := 50
	force := true
	runtime := true
	enabled := false
	valid := false
	conditions := []string{"cond1", "cond2"}

	opts := NewOptions(
		WithScope(scope),
		WithKey(key),
		WithPriority(priority),
		WithForce(force),
		WithRuntime(runtime),
		WithEnabled(enabled),
		WithValid(valid),
		WithConditions(conditions),
	)

	assert.Equal(t, scope, opts.Router.Scope)
	assert.Equal(t, key, opts.Router.Key)
	assert.Equal(t, priority, opts.Router.Priority)
	assert.NotNil(t, opts.Router.Force)
	assert.Equal(t, force, *opts.Router.Force)
	assert.NotNil(t, opts.Router.Runtime)
	assert.Equal(t, runtime, *opts.Router.Runtime)
	assert.NotNil(t, opts.Router.Enabled)
	assert.Equal(t, enabled, *opts.Router.Enabled)
	assert.NotNil(t, opts.Router.Valid)
	assert.Equal(t, valid, *opts.Router.Valid)
	assert.Equal(t, conditions, opts.Router.Conditions)
}

func TestWithEmptyConditions(t *testing.T) {
	conditions := []string{}
	opts := NewOptions(WithConditions(conditions))
	assert.NotNil(t, opts.Router.Conditions)
	assert.Equal(t, 0, len(opts.Router.Conditions))
}

func TestWithEmptyTags(t *testing.T) {
	tags := []global.Tag{}
	opts := NewOptions(WithTags(tags))
	assert.NotNil(t, opts.Router.Tags)
	assert.Equal(t, 0, len(opts.Router.Tags))
}
