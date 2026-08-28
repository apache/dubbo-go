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
	"encoding/json"
	"sort"
)

import (
	"github.com/dubbogo/gost/log/logger"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/metadata/definition"
	"dubbo.apache.org/dubbo-go/v3/metadata/report"
)

// PublishServiceDefinitions builds and publishes an interface-level service
// definition for each exported URL that carries a describable contract, and
// returns the URLs whose publish failed for a reason worth retrying.
//
// No error is returned and nothing here blocks. A provider whose definition did
// not reach the metadata center is still a working provider — it is only
// invisible to Admin's console — so a metadata-center outage must not keep
// instances out of traffic. Java reaches the same outcome by a different route:
// AbstractMetadataReport defaults sync-report to false and hands the write to an
// executor, so export never observes the result at all.
//
// Only write failures come back. A service that has no describable contract —
// unregistered, unbuildable, or with no publishable methods — is reported in the
// log and dropped, because retrying cannot change any of those.
//
// Publishing is idempotent and keyed only by service identity, so calling this
// on every start, on each cycle-report pass, and on every retry simply
// overwrites the previous document.
func PublishServiceDefinitions(urls []*common.URL) []*common.URL {
	publishers := serviceDefinitionPublishers()
	if len(publishers) == 0 {
		return nil
	}

	var failed []*common.URL
	for _, u := range dedupeByService(urls) {
		if !publishServiceDefinition(u, publishers) {
			failed = append(failed, u)
		}
	}
	return failed
}

// publishServiceDefinition reports whether the caller should retry.
//
// A partial failure across several reports counts as a failure: the retry
// republishes to all of them, which is harmless because each write overwrites.
func publishServiceDefinition(u *common.URL, publishers []report.ServiceDefinitionPublisher) bool {
	svc := common.ServiceMap.GetServiceByServiceKey(u.Protocol, u.ServiceKey())
	if svc == nil {
		logger.Warnf("[Metadata][Definition] no registered service for %s/%s, skipping definition",
			u.Protocol, u.ServiceKey())
		return true
	}

	def, skips, err := definition.BuildFromURL(u, svc.ServiceType())
	if err != nil {
		logger.Errorf("[Metadata][Definition] could not build definition for %s: %v", u.ServiceKey(), err)
		return true
	}
	for _, skip := range skips {
		logger.Warnf("[Metadata][Definition] method %s.%s is not published: %s",
			def.CanonicalName, skip.Name, skip.Reason)
	}
	if len(def.Methods) == 0 {
		logger.Warnf("[Metadata][Definition] %s has no publishable methods, skipping definition",
			def.CanonicalName)
		return true
	}

	payload, err := json.Marshal(def)
	if err != nil {
		logger.Errorf("[Metadata][Definition] could not serialize definition for %s: %v",
			def.CanonicalName, err)
		return true
	}

	application := u.GetParam(constant.ApplicationKey, "")
	published := true
	for _, publisher := range publishers {
		if err := publisher.PublishServiceDefinition(
			def.CanonicalName, u.Version(), u.Group(), application, string(payload),
		); err != nil {
			logger.Errorf("[Metadata][Definition] could not publish definition for %s: %v",
				def.CanonicalName, err)
			published = false
			continue
		}
		logger.Infof("[Metadata][Definition] published definition for %s, methods=%d types=%d",
			def.CanonicalName, len(def.Methods), len(def.Types))
	}
	return published
}

// ServiceDefinitionsEnabled reports whether any configured metadata report will
// accept interface-level service definitions.
//
// Callers use this to decide whether work that only exists to serve definitions
// — the daily re-publish, for one — is worth scheduling at all.
func ServiceDefinitionsEnabled() bool {
	return len(serviceDefinitionPublishers()) > 0
}

// serviceDefinitionPublishers returns the configured reports that both support
// the capability and have it switched on.
//
// The instance table stores *DelegateMetadataReport, so the capability has to be
// queried through the wrapper rather than type-asserted off the interface value.
func serviceDefinitionPublishers() []report.ServiceDefinitionPublisher {
	var publishers []report.ServiceDefinitionPublisher
	for _, r := range GetMetadataReports() {
		delegate, ok := r.(*DelegateMetadataReport)
		if !ok {
			continue
		}
		publisher, supported := delegate.ServiceDefinitionPublisher()
		if !supported {
			continue
		}
		if url := delegate.URL(); url != nil &&
			!url.GetParamBool(constant.MetadataReportReportDefinitionKey, true) {
			continue
		}
		publishers = append(publishers, publisher)
	}
	return publishers
}

// dedupeByService reduces the exported URLs to one per service identity.
//
// A service exported over several protocols produces several URLs, but the
// definition key holds no protocol, so publishing each would have them
// overwrite one another in an order that depends on map iteration. The contract
// is identical across protocols anyway — the method set comes from one handler's
// reflection — so one document per service is the whole truth. Protocol belongs
// to the instance's exported-services revision, which Admin reads separately.
//
// Selection is by lowest protocol name so restarts converge on the same URL,
// which puts "dubbo" ahead of "tri" for a dual-exported service.
func dedupeByService(urls []*common.URL) []*common.URL {
	selected := make(map[string]*common.URL, len(urls))
	for _, u := range urls {
		if u == nil || !describable(u) {
			continue
		}
		key := u.ServiceKey()
		if current, seen := selected[key]; seen && current.Protocol <= u.Protocol {
			continue
		}
		selected[key] = u
	}

	keys := make([]string, 0, len(selected))
	for key := range selected {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	out := make([]*common.URL, 0, len(keys))
	for _, key := range keys {
		out = append(out, selected[key])
	}
	return out
}

// describable reports whether a URL's export carries a contract this package can
// derive from Go reflection.
//
// Only two exports qualify: Dubbo/Hessian2, and Triple in non-IDL mode. An IDL
// (protobuf) service must not be published from reflection — the generated
// struct is a lossy view of the proto, missing real field names, enums, oneofs
// and well-known type semantics — so its contract needs a descriptor-based
// builder that does not exist yet.
//
// Non-IDL is detected by the absence of the ServiceInfo attribute rather than
// the IDL-mode URL parameter the proposal suggested. Server.Register threads a
// *common.ServiceInfo through for IDL services and Server.RegisterService
// passes nil for non-IDL ones, and enhanceServiceInfo preserves that nil, so the
// attribute is set exactly for IDL exports. The IDL-mode parameter would work
// today but is marked for removal along with constant.IDLMode, constant.NONIDL
// and WithIDLMode.
func describable(u *common.URL) bool {
	switch u.Protocol {
	case constant.DubboProtocol:
		return true
	case constant.TriProtocol:
		_, isIDL := u.GetAttribute(constant.ServiceInfoKey)
		return !isIDL
	default:
		// REST, gRPC and anything else either has its own contract format or no
		// generic-invocation path for Admin to call through.
		return false
	}
}
