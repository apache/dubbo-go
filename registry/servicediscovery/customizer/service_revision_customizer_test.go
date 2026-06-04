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

	metadata.AddService("reg-a", urlA)
	metadata.AddService("reg-b", urlB)

	cus := &exportedServicesRevisionMetadataCustomizer{}

	instA := &registry.DefaultServiceInstance{
		Metadata: map[string]string{constant.RegistryIdKey: "reg-a"},
	}
	cus.Customize(instA)
	revA := instA.GetMetadata()[constant.ExportedServicesRevisionPropertyName]

	instB := &registry.DefaultServiceInstance{
		Metadata: map[string]string{constant.RegistryIdKey: "reg-b"},
	}
	cus.Customize(instB)
	revB := instB.GetMetadata()[constant.ExportedServicesRevisionPropertyName]

	assert.NotEqual(t, revA, revB, "different registries with different services should produce different revisions")
	assert.NotEqual(t, "0", revA, "reg-a has a service, revision should not be 0")
	assert.NotEqual(t, "0", revB, "reg-b has a service, revision should not be 0")
}

// TestSubscribedRevisionIsRegistryScoped mirrors the exported test for subscribed URLs.
func TestSubscribedRevisionIsRegistryScoped(t *testing.T) {
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

	metadata.AddSubscribeURL("reg-a", urlA)
	metadata.AddSubscribeURL("reg-b", urlB)

	cus := &subscribedServicesRevisionMetadataCustomizer{}

	instA := &registry.DefaultServiceInstance{
		Metadata: map[string]string{constant.RegistryIdKey: "reg-a"},
	}
	cus.Customize(instA)
	revA := instA.GetMetadata()[constant.SubscribedServicesRevisionPropertyName]

	instB := &registry.DefaultServiceInstance{
		Metadata: map[string]string{constant.RegistryIdKey: "reg-b"},
	}
	cus.Customize(instB)
	revB := instB.GetMetadata()[constant.SubscribedServicesRevisionPropertyName]

	assert.NotEqual(t, revA, revB, "different registries with different subscriptions should produce different revisions")
}
