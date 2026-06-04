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

package customizer

import (
	"testing"
)

import (
	"github.com/stretchr/testify/assert"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
)

// helper to create a URL with common service discovery fields
func newTestURL(protocol string, port string, path string, application string, group string, version string, methods []string, extraParams map[string]string) *common.URL {
	opts := []common.Option{
		common.WithProtocol(protocol),
		common.WithPort(port),
		common.WithPath(path),
		common.WithParamsValue(constant.ApplicationKey, application),
		common.WithParamsValue(constant.GroupKey, group),
		common.WithParamsValue(constant.VersionKey, version),
		common.WithParamsValue(constant.SideKey, constant.SideProvider),
	}
	if len(methods) > 0 {
		opts = append(opts, common.WithMethods(methods))
	}
	for k, v := range extraParams {
		opts = append(opts, common.WithParamsValue(k, v))
	}
	u, _ := common.NewURL(protocol+"://127.0.0.1:"+port+"/"+path, opts...)
	return u
}

// 1. group change → revision changes
func TestRevisionChangesOnGroupChange(t *testing.T) {
	u1 := newTestURL("dubbo", "20880", "com.example.TestService", "test-app", "groupA", "1.0.0", []string{"sayHello"}, nil)
	u2 := newTestURL("dubbo", "20880", "com.example.TestService", "test-app", "groupB", "1.0.0", []string{"sayHello"}, nil)

	r1 := resolveRevision([]*common.URL{u1})
	r2 := resolveRevision([]*common.URL{u2})

	assert.NotEmpty(t, r1)
	assert.NotEmpty(t, r2)
	assert.NotEqual(t, r1, r2, "revision should change when group changes")
}

// 2. protocol change → revision changes
func TestRevisionChangesOnProtocolChange(t *testing.T) {
	u1 := newTestURL("dubbo", "20880", "com.example.TestService", "test-app", "groupA", "1.0.0", []string{"sayHello"}, nil)
	u2 := newTestURL("tri", "20880", "com.example.TestService", "test-app", "groupA", "1.0.0", []string{"sayHello"}, nil)

	r1 := resolveRevision([]*common.URL{u1})
	r2 := resolveRevision([]*common.URL{u2})

	assert.NotEmpty(t, r1)
	assert.NotEmpty(t, r2)
	assert.NotEqual(t, r1, r2, "revision should change when protocol changes")
}

// 3. params change (timeout, loadbalance) → revision changes
func TestRevisionChangesOnParamsChange(t *testing.T) {
	u1 := newTestURL("dubbo", "20880", "com.example.TestService", "test-app", "groupA", "1.0.0", []string{"sayHello"},
		map[string]string{constant.TimeoutKey: "3000", constant.LoadbalanceKey: "random"})
	u2 := newTestURL("dubbo", "20880", "com.example.TestService", "test-app", "groupA", "1.0.0", []string{"sayHello"},
		map[string]string{constant.TimeoutKey: "5000", constant.LoadbalanceKey: "roundrobin"})

	r1 := resolveRevision([]*common.URL{u1})
	r2 := resolveRevision([]*common.URL{u2})

	assert.NotEmpty(t, r1)
	assert.NotEmpty(t, r2)
	assert.NotEqual(t, r1, r2, "revision should change when params (timeout/loadbalance) change")
}

// 4. methods change → revision changes
func TestRevisionChangesOnMethodChange(t *testing.T) {
	u1 := newTestURL("dubbo", "20880", "com.example.TestService", "test-app", "groupA", "1.0.0", []string{"sayHello"}, nil)
	u2 := newTestURL("dubbo", "20880", "com.example.TestService", "test-app", "groupA", "1.0.0", []string{"sayHello", "sayGoodbye"}, nil)

	r1 := resolveRevision([]*common.URL{u1})
	r2 := resolveRevision([]*common.URL{u2})

	assert.NotEmpty(t, r1)
	assert.NotEmpty(t, r2)
	assert.NotEqual(t, r1, r2, "revision should change when methods change")
}

// 5. version change → revision changes
func TestRevisionChangesOnVersionChange(t *testing.T) {
	u1 := newTestURL("dubbo", "20880", "com.example.TestService", "test-app", "groupA", "1.0.0", []string{"sayHello"}, nil)
	u2 := newTestURL("dubbo", "20880", "com.example.TestService", "test-app", "groupA", "2.0.0", []string{"sayHello"}, nil)

	r1 := resolveRevision([]*common.URL{u1})
	r2 := resolveRevision([]*common.URL{u2})

	assert.NotEmpty(t, r1)
	assert.NotEmpty(t, r2)
	assert.NotEqual(t, r1, r2, "revision should change when version changes")
}

// 6. same input → same revision (deterministic)
func TestRevisionStable(t *testing.T) {
	u := newTestURL("dubbo", "20880", "com.example.TestService", "test-app", "groupA", "1.0.0", []string{"sayHello", "sayGoodbye"},
		map[string]string{constant.TimeoutKey: "3000"})

	r1 := resolveRevision([]*common.URL{u})
	r2 := resolveRevision([]*common.URL{u})
	r3 := resolveRevision([]*common.URL{u})

	assert.Equal(t, r1, r2, "same input should produce same revision")
	assert.Equal(t, r2, r3, "same input should produce same revision")
}

// 7. empty URL list → "0"
func TestRevisionEmptyServices(t *testing.T) {
	r := resolveRevision(nil)
	assert.Equal(t, "0", r)

	r = resolveRevision([]*common.URL{})
	assert.Equal(t, "0", r)
}

// 8. different insertion order → same revision
func TestRevisionOrderingIndependent(t *testing.T) {
	u1 := newTestURL("dubbo", "20880", "com.example.TestService", "test-app", "groupA", "1.0.0", []string{"sayHello"}, nil)
	u2 := newTestURL("dubbo", "20881", "com.example.AnotherService", "test-app", "groupA", "1.0.0", []string{"getUser"}, nil)

	r1 := resolveRevision([]*common.URL{u1, u2})
	r2 := resolveRevision([]*common.URL{u2, u1})

	assert.Equal(t, r1, r2, "revision should be the same regardless of URL order")
}

// 9. params key-value ordering → same revision
func TestRevisionParamsOrderStable(t *testing.T) {
	// Params in different order within URL — NewURL handles param insertion,
	// so we build two URLs that end up with the same effective params.
	// The key test is that ServiceInfo.toDescString() sorts params by key.
	u1 := newTestURL("dubbo", "20880", "com.example.TestService", "test-app", "groupA", "1.0.0", []string{"sayHello"},
		map[string]string{constant.TimeoutKey: "3000", constant.ClusterKey: "failover"})
	u2 := newTestURL("dubbo", "20880", "com.example.TestService", "test-app", "groupA", "1.0.0", []string{"sayHello"},
		map[string]string{constant.ClusterKey: "failover", constant.TimeoutKey: "3000"})

	r1 := resolveRevision([]*common.URL{u1})
	r2 := resolveRevision([]*common.URL{u2})

	assert.Equal(t, r1, r2, "revision should be the same regardless of param key-value ordering")
}

// 10. non-IncludeKeys params → revision unchanged
func TestRevisionIgnoresNonIncludeKeys(t *testing.T) {
	u1 := newTestURL("dubbo", "20880", "com.example.TestService", "test-app", "groupA", "1.0.0", []string{"sayHello"},
		map[string]string{constant.TimeoutKey: "3000"})
	// add a param NOT in IncludeKeys (e.g., a custom arbitrary param)
	u2 := newTestURL("dubbo", "20880", "com.example.TestService", "test-app", "groupA", "1.0.0", []string{"sayHello"},
		map[string]string{constant.TimeoutKey: "3000", "custom.arbitrary.key": "someValue"})

	r1 := resolveRevision([]*common.URL{u1})
	r2 := resolveRevision([]*common.URL{u2})

	// Note: custom.arbitrary.key is NOT in IncludeKeys, so it should be filtered by NewServiceInfoWithURL
	assert.Equal(t, r1, r2, "revision should ignore params not in IncludeKeys")
}
