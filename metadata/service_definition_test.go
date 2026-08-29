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

import (
	"context"
	"encoding/json"
	"sync"
	"testing"
)

import (
	gxset "github.com/dubbogo/gost/container/set"

	perrors "github.com/pkg/errors"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/metadata/definition"
	"dubbo.apache.org/dubbo-go/v3/metadata/info"
	"dubbo.apache.org/dubbo-go/v3/metadata/mapping"
	"dubbo.apache.org/dubbo-go/v3/metadata/report"
)

// ---------------------------------------------------------------------------
// fakes
// ---------------------------------------------------------------------------
type publishedDefinition struct {
	ServiceInterface string
	Version          string
	Group            string
	Application      string
	JSON             string
}

// baseReport satisfies report.MetadataReport without the definition capability.
type baseReport struct {
	url *common.URL
}

func (r *baseReport) GetAppMetadata(string, string) (*info.MetadataInfo, error) { return nil, nil }
func (r *baseReport) PublishAppMetadata(string, string, *info.MetadataInfo) error {
	return nil
}
func (r *baseReport) RegisterServiceAppMapping(string, string, string) error { return nil }
func (r *baseReport) GetServiceAppMapping(string, string, mapping.MappingListener) (*gxset.HashSet, error) {
	return nil, nil
}
func (r *baseReport) RemoveServiceAppMappingListener(string, string) error  { return nil }
func (r *baseReport) UnPublishAppMetadata(string, string) error             { return nil }
func (r *baseReport) ListAppRevisions(string) ([]report.AppRevision, error) { return nil, nil }
func (r *baseReport) URL() *common.URL                                      { return r.url }

// capableReport additionally implements report.ServiceDefinitionPublisher.
type capableReport struct {
	baseReport
	mu        sync.Mutex
	published []publishedDefinition
	err       error
}

func (r *capableReport) PublishServiceDefinition(serviceInterface, version, group, application, definitionJSON string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.err != nil {
		return r.err
	}
	r.published = append(r.published, publishedDefinition{
		ServiceInterface: serviceInterface,
		Version:          version,
		Group:            group,
		Application:      application,
		JSON:             definitionJSON,
	})
	return nil
}

func (r *capableReport) snapshot() []publishedDefinition {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]publishedDefinition(nil), r.published...)
}

// ---------------------------------------------------------------------------
// fixtures
// ---------------------------------------------------------------------------

type EchoService struct{}

func (s *EchoService) Echo(ctx context.Context, msg string) (string, error) { return msg, nil }

const echoInterface = "org.example.EchoService"

func reportURL(t *testing.T, params map[string]string) *common.URL {
	t.Helper()
	u := common.NewURLWithOptions(
		common.WithProtocol("nacos"),
		common.WithIp("127.0.0.1"),
		common.WithPort("8848"),
	)
	for k, v := range params {
		u.SetParam(k, v)
	}
	return u
}

// installReports replaces the package instance table for the duration of a test.
func installReports(t *testing.T, table map[string]report.MetadataReport) {
	t.Helper()
	instancesMu.Lock()
	instances = table
	instancesMu.Unlock()
	t.Cleanup(func() {
		instancesMu.Lock()
		instances = make(map[string]report.MetadataReport)
		instancesMu.Unlock()
	})
}

func installCapableReport(t *testing.T, params map[string]string) *capableReport {
	t.Helper()
	backend := &capableReport{baseReport: baseReport{url: reportURL(t, params)}}
	installReports(t, map[string]report.MetadataReport{
		constant.DefaultKey: &DelegateMetadataReport{instance: backend},
	})
	return backend
}

// exportService registers a handler in ServiceMap the way Export does and
// returns its exported URL.
func exportService(t *testing.T, protocol string, idl bool) *common.URL {
	t.Helper()
	_, err := common.ServiceMap.Register(echoInterface, protocol, "", "", &EchoService{})
	require.NoError(t, err)
	t.Cleanup(func() {
		_ = common.ServiceMap.UnRegister(echoInterface, protocol,
			common.ServiceKey(echoInterface, "", ""))
	})

	u := common.NewURLWithOptions(
		common.WithProtocol(protocol),
		common.WithIp("127.0.0.1"),
		common.WithPort("20000"),
		common.WithParamsValue(constant.InterfaceKey, echoInterface),
		common.WithParamsValue(constant.ApplicationKey, "demo-app"),
		common.WithParamsValue(constant.ReleaseKey, "dubbo-golang-3.3.0"),
		common.WithParamsValue(constant.SideKey, "provider"),
	)
	if idl {
		// Export sets this attribute only when Server.Register threaded a
		// ServiceInfo through, which is exactly the IDL path.
		u.SetAttribute(constant.ServiceInfoKey, &common.ServiceInfo{InterfaceName: echoInterface})
	}
	return u
}

// ---------------------------------------------------------------------------
// capability discovery
// ---------------------------------------------------------------------------

func TestCapabilityIsVisibleThroughTheDelegateWrapper(t *testing.T) {
	// The instance table stores wrappers, so a direct type assertion on the
	// value returned by GetMetadataReports would hide a capable backend.
	backend := installCapableReport(t, nil)

	reports := GetMetadataReports()
	require.Len(t, reports, 1)

	_, direct := reports[0].(report.ServiceDefinitionPublisher)
	assert.False(t, direct, "the wrapper itself must not satisfy the capability interface")

	delegate, ok := reports[0].(*DelegateMetadataReport)
	require.True(t, ok)
	publisher, supported := delegate.ServiceDefinitionPublisher()
	require.True(t, supported, "capability must be reachable through the wrapper")

	require.NoError(t, publisher.PublishServiceDefinition("i", "v", "g", "a", "{}"))
	assert.Len(t, backend.snapshot(), 1)
}

func TestBackendWithoutCapabilityReportsUnsupported(t *testing.T) {
	installReports(t, map[string]report.MetadataReport{
		constant.DefaultKey: &DelegateMetadataReport{instance: &baseReport{url: reportURL(t, nil)}},
	})

	delegate := GetMetadataReports()[0].(*DelegateMetadataReport)
	_, supported := delegate.ServiceDefinitionPublisher()
	assert.False(t, supported, "zookeeper/etcd/third-party reports must be unaffected")
	assert.Empty(t, serviceDefinitionPublishers())
}

// ---------------------------------------------------------------------------
// publishing
// ---------------------------------------------------------------------------

func TestPublishServiceDefinitions(t *testing.T) {
	backend := installCapableReport(t, nil)
	u := exportService(t, constant.DubboProtocol, false)

	PublishServiceDefinitions([]*common.URL{u})

	published := backend.snapshot()
	require.Len(t, published, 1)
	assert.Equal(t, echoInterface, published[0].ServiceInterface)
	assert.Equal(t, "demo-app", published[0].Application)
	assert.Empty(t, published[0].Version)
	assert.Empty(t, published[0].Group)

	var def definition.ServiceDefinition
	require.NoError(t, json.Unmarshal([]byte(published[0].JSON), &def))
	assert.Equal(t, echoInterface, def.CanonicalName)
	require.Len(t, def.Methods, 1)
	assert.Equal(t, "Echo", def.Methods[0].Name)
	assert.Equal(t, "java.lang.String", def.Methods[0].ReturnType)
}

func TestPublishIsIdempotent(t *testing.T) {
	backend := installCapableReport(t, nil)
	u := exportService(t, constant.DubboProtocol, false)

	PublishServiceDefinitions([]*common.URL{u})
	PublishServiceDefinitions([]*common.URL{u})

	published := backend.snapshot()
	require.Len(t, published, 2, "each call publishes; the backend overwrites in place")
	assert.Equal(t, published[0].JSON, published[1].JSON,
		"republishing must produce byte-identical content or the metadata center churns")
}

func TestPublishFailureDoesNotPanicOrBlock(t *testing.T) {
	backend := installCapableReport(t, nil)
	backend.err = perrors.New("nacos unavailable")
	u := exportService(t, constant.DubboProtocol, false)

	// A metadata-center outage must not keep the provider out of traffic, so
	// this returns normally and the caller proceeds to register instances.
	assert.NotPanics(t, func() { PublishServiceDefinitions([]*common.URL{u}) })
}

func TestPublishSkippedWhenSwitchedOff(t *testing.T) {
	backend := installCapableReport(t, map[string]string{
		constant.MetadataReportReportDefinitionKey: "false",
	})
	u := exportService(t, constant.DubboProtocol, false)

	PublishServiceDefinitions([]*common.URL{u})
	assert.Empty(t, backend.snapshot())
}

func TestPublishEnabledByDefault(t *testing.T) {
	// An absent key must behave as enabled, matching Java's report-definition,
	// which defaults to true.
	backend := installCapableReport(t, nil)
	u := exportService(t, constant.DubboProtocol, false)

	PublishServiceDefinitions([]*common.URL{u})
	assert.Len(t, backend.snapshot(), 1)
}

// TestServiceDefinitionsEnabled covers the predicate the daily re-publish uses
// to decide whether it has anything to do. That pass is what keeps a live
// contract's timestamp fresh, so scheduling it must not depend on where
// application metadata happens to live.
func TestServiceDefinitionsEnabled(t *testing.T) {
	t.Run("capable and switched on", func(t *testing.T) {
		installCapableReport(t, nil)
		assert.True(t, ServiceDefinitionsEnabled())
	})

	t.Run("capable but switched off", func(t *testing.T) {
		installCapableReport(t, map[string]string{
			constant.MetadataReportReportDefinitionKey: "false",
		})
		assert.False(t, ServiceDefinitionsEnabled())
	})

	t.Run("backend without the capability", func(t *testing.T) {
		installReports(t, map[string]report.MetadataReport{
			constant.DefaultKey: &DelegateMetadataReport{instance: &baseReport{url: reportURL(t, nil)}},
		})
		assert.False(t, ServiceDefinitionsEnabled())
	})

	t.Run("no reports at all", func(t *testing.T) {
		installReports(t, map[string]report.MetadataReport{})
		assert.False(t, ServiceDefinitionsEnabled())
	})
}

// ---------------------------------------------------------------------------
// protocol gating
// ---------------------------------------------------------------------------

func TestDescribableProtocols(t *testing.T) {
	cases := []struct {
		name     string
		protocol string
		idl      bool
		want     bool
	}{
		{"dubbo", constant.DubboProtocol, false, true},
		{"triple non-IDL", constant.TriProtocol, false, true},
		{"triple IDL", constant.TriProtocol, true, false},
		{"rest", "rest", false, false},
		{"grpc", "grpc", false, false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			u := common.NewURLWithOptions(
				common.WithProtocol(tc.protocol),
				common.WithParamsValue(constant.InterfaceKey, echoInterface),
			)
			if tc.idl {
				u.SetAttribute(constant.ServiceInfoKey, &common.ServiceInfo{})
			}
			assert.Equal(t, tc.want, describable(u))
		})
	}
}

func TestTripleIDLIsNotPublished(t *testing.T) {
	// A protobuf service's contract must come from the descriptor, not from the
	// generated struct, so it is excluded until that builder exists.
	backend := installCapableReport(t, nil)
	u := exportService(t, constant.TriProtocol, true)

	PublishServiceDefinitions([]*common.URL{u})
	assert.Empty(t, backend.snapshot())
}

func TestTripleNonIDLIsPublished(t *testing.T) {
	backend := installCapableReport(t, nil)
	u := exportService(t, constant.TriProtocol, false)

	PublishServiceDefinitions([]*common.URL{u})
	assert.Len(t, backend.snapshot(), 1)
}

// ---------------------------------------------------------------------------
// multi-protocol de-duplication
// ---------------------------------------------------------------------------

func TestMultiProtocolPublishesOneDefinition(t *testing.T) {
	// The definition key holds no protocol, so publishing per-protocol would
	// have the documents overwrite one another non-deterministically.
	backend := installCapableReport(t, nil)
	dubboURL := exportService(t, constant.DubboProtocol, false)
	tripleURL := exportService(t, constant.TriProtocol, false)

	PublishServiceDefinitions([]*common.URL{tripleURL, dubboURL})

	published := backend.snapshot()
	require.Len(t, published, 1)
	assert.Equal(t, echoInterface, published[0].ServiceInterface)
}

func TestDedupeSelectionIsOrderIndependent(t *testing.T) {
	dubboURL := exportService(t, constant.DubboProtocol, false)
	tripleURL := exportService(t, constant.TriProtocol, false)

	forward := dedupeByService([]*common.URL{dubboURL, tripleURL})
	reverse := dedupeByService([]*common.URL{tripleURL, dubboURL})

	require.Len(t, forward, 1)
	require.Len(t, reverse, 1)
	assert.Equal(t, forward[0].Protocol, reverse[0].Protocol,
		"input order must not decide which URL wins, or restarts would flap")
	assert.Equal(t, constant.DubboProtocol, forward[0].Protocol)
}

func TestDedupeKeepsDistinctServices(t *testing.T) {
	a := exportService(t, constant.DubboProtocol, false)
	b := common.NewURLWithOptions(
		common.WithProtocol(constant.DubboProtocol),
		common.WithParamsValue(constant.InterfaceKey, "org.example.Other"),
	)
	assert.Len(t, dedupeByService([]*common.URL{a, b}), 2)
}

func TestUnregisteredServiceIsSkipped(t *testing.T) {
	backend := installCapableReport(t, nil)
	u := common.NewURLWithOptions(
		common.WithProtocol(constant.DubboProtocol),
		common.WithParamsValue(constant.InterfaceKey, "org.example.NeverExported"),
	)

	assert.NotPanics(t, func() { PublishServiceDefinitions([]*common.URL{u}) })
	assert.Empty(t, backend.snapshot())
}

func TestNoReportsIsANoOp(t *testing.T) {
	installReports(t, map[string]report.MetadataReport{})
	u := exportService(t, constant.DubboProtocol, false)
	assert.NotPanics(t, func() { PublishServiceDefinitions([]*common.URL{u}) })
}
