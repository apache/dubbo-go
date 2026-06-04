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
	"dubbo.apache.org/dubbo-go/v3/metadata"
	"dubbo.apache.org/dubbo-go/v3/registry"
)

// TestExportedRevisionIsRegistryScoped verifies that when two registries export different
// service sets, their instances get different revision values — not a merged cross-registry one.
func TestExportedRevisionIsRegistryScoped(t *testing.T) {
	metadata.ClearMetadataReportInstances()
	t.Cleanup(func() {
		metadata.ClearMetadataReportInstances()
	})

	urlA := common.NewURLWithOptions(
		common.WithInterface("org.example.ServiceA"),
		common.WithParamsValue(constant.ApplicationKey, "app"),
		common.WithPort("20880"),
	)
	urlB := common.NewURLWithOptions(
		common.WithInterface("org.example.ServiceB"),
		common.WithParamsValue(constant.ApplicationKey, "app"),
		common.WithPort("20881"),
	)

	metadata.AddService("rev-reg-a", urlA)
	metadata.AddService("rev-reg-b", urlB)

	cus := &exportedServicesRevisionMetadataCustomizer{}

	instA := &registry.DefaultServiceInstance{
		Metadata: map[string]string{constant.RegistryIdKey: "rev-reg-a"},
	}
	cus.Customize(instA)
	revA := instA.GetMetadata()[constant.ExportedServicesRevisionPropertyName]

	instB := &registry.DefaultServiceInstance{
		Metadata: map[string]string{constant.RegistryIdKey: "rev-reg-b"},
	}
	cus.Customize(instB)
	revB := instB.GetMetadata()[constant.ExportedServicesRevisionPropertyName]

	assert.NotEqual(t, revA, revB, "different registries with different services should produce different revisions")
	assert.NotEqual(t, "0", revA, "reg-a has a service, revision should not be 0")
	assert.NotEqual(t, "0", revB, "reg-b has a service, revision should not be 0")
}

// TestExportedRevisionMissingRegistryIdYieldsZero verifies that an instance with no
// RegistryIdKey gets revision "0" (no services found), and does not panic or use
// another registry's service list.
func TestExportedRevisionMissingRegistryIdYieldsZero(t *testing.T) {
	metadata.ClearMetadataReportInstances()
	t.Cleanup(metadata.ClearMetadataReportInstances)

	// Register a service under a real registry so we can confirm it is NOT used
	urlA := common.NewURLWithOptions(
		common.WithInterface("org.example.ShouldBeIsolated"),
		common.WithParamsValue(constant.ApplicationKey, "app"),
		common.WithPort("20880"),
	)
	metadata.AddService("some-registry", urlA)

	cus := &exportedServicesRevisionMetadataCustomizer{}

	inst := &registry.DefaultServiceInstance{
		Metadata: map[string]string{}, // no RegistryIdKey
	}
	cus.Customize(inst)
	rev := inst.GetMetadata()[constant.ExportedServicesRevisionPropertyName]

	// GetMetadataInfo("") returns nil → resolveRevision(nil) == "0"
	assert.Equal(t, "0", rev, "instance with no registryId should get revision 0, not borrow another registry's service list")
}

// TestSubscribedRevisionIsRegistryScoped mirrors the exported test for subscribed URLs.
func TestSubscribedRevisionIsRegistryScoped(t *testing.T) {
	metadata.ClearMetadataReportInstances()
	t.Cleanup(metadata.ClearMetadataReportInstances)

	urlA := common.NewURLWithOptions(
		common.WithInterface("org.example.SubA"),
		common.WithParamsValue(constant.ApplicationKey, "app"),
		common.WithPort("20880"),
	)
	urlB := common.NewURLWithOptions(
		common.WithInterface("org.example.SubB"),
		common.WithParamsValue(constant.ApplicationKey, "app"),
		common.WithPort("20881"),
	)

	metadata.AddSubscribeURL("sub-reg-a", urlA)
	metadata.AddSubscribeURL("sub-reg-b", urlB)

	cus := &subscribedServicesRevisionMetadataCustomizer{}

	instA := &registry.DefaultServiceInstance{
		Metadata: map[string]string{constant.RegistryIdKey: "sub-reg-a"},
	}
	cus.Customize(instA)
	revA := instA.GetMetadata()[constant.SubscribedServicesRevisionPropertyName]

	instB := &registry.DefaultServiceInstance{
		Metadata: map[string]string{constant.RegistryIdKey: "sub-reg-b"},
	}
	cus.Customize(instB)
	revB := instB.GetMetadata()[constant.SubscribedServicesRevisionPropertyName]

	assert.NotEqual(t, revA, revB, "different registries with different subscriptions should produce different revisions")
}

// TestSubscribedRevisionMissingRegistryIdYieldsZero mirrors the exported missing-key test.
func TestSubscribedRevisionMissingRegistryIdYieldsZero(t *testing.T) {
	metadata.ClearMetadataReportInstances()
	t.Cleanup(metadata.ClearMetadataReportInstances)

	urlA := common.NewURLWithOptions(
		common.WithInterface("org.example.SubIsolated"),
		common.WithParamsValue(constant.ApplicationKey, "app"),
		common.WithPort("20880"),
	)
	metadata.AddSubscribeURL("some-registry", urlA)

	cus := &subscribedServicesRevisionMetadataCustomizer{}
	inst := &registry.DefaultServiceInstance{
		Metadata: map[string]string{},
	}
	cus.Customize(inst)
	rev := inst.GetMetadata()[constant.SubscribedServicesRevisionPropertyName]

	assert.Equal(t, "0", rev, "instance with no registryId should get revision 0, not borrow another registry's subscriptions")
}
