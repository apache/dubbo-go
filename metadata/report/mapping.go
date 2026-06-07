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

package report

import (
	"errors"
	"strings"
)

import (
	gxset "github.com/dubbogo/gost/container/set"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common/constant"
)

// ErrMappingCASConflict is returned by MetadataReport.RegisterServiceAppMapping when an
// optimistic-concurrency (compare-and-swap) write loses a race against a concurrent writer.
//
// It is a *retriable* error: the caller should re-read the latest value, re-merge its
// application name, and try again. Each backend wraps its native conflict error
// (etcd ErrCompareFail, ZooKeeper ErrBadVersion / ErrNodeExists, Nacos CAS publish failure)
// with this sentinel using fmt.Errorf("...: %w", ...), so the upper layer can classify
// retriable vs. permanent failures with errors.Is and only burn its retry budget on the
// former.
var ErrMappingCASConflict = errors.New("service-app mapping CAS conflict")

// MergeServiceAppMapping merges application into oldVal, the comma-separated set of
// application names stored under a single interface key in the metadata center.
//
// It compares whole elements rather than substrings. The previous implementations used
// strings.Contains(oldVal, application), which produced false positives — e.g. registering
// "order" was wrongly treated as already-present when the set contained "order-service",
// so "order" could never be written. It also never emits an empty element, fixing the
// leading-comma bug ("" + "," + app => ",app") triggered when oldVal is empty.
//
// It returns the merged value and whether oldVal changed. changed == false means application
// was already present and no write (hence no CAS round-trip) is required.
//
// New names are appended to the end rather than rebuilt from a set: this keeps existing
// ordering and makes the written bytes change minimally and deterministically, which is
// what optimistic concurrency relies on. Rebuilding from a Go map/set would reorder the
// value nondeterministically, producing spurious CAS conflicts and MD5 churn even when the
// logical content is unchanged.
func MergeServiceAppMapping(oldVal, application string) (string, bool) {
	if oldVal == "" {
		return application, true
	}
	for _, app := range strings.Split(oldVal, constant.CommaSeparator) {
		if app == application {
			return oldVal, false
		}
	}
	return oldVal + constant.CommaSeparator + application, true
}

// DecodeServiceAppNames parses the comma-separated application set stored under an interface
// key into a HashSet, skipping empty elements. A blank value yields an empty set rather than a
// set containing one empty string, which strings.Split("", ",") would otherwise produce.
func DecodeServiceAppNames(val string) *gxset.HashSet {
	set := gxset.NewSet()
	if val == "" {
		return set
	}
	for _, app := range strings.Split(val, constant.CommaSeparator) {
		if app != "" {
			set.Add(app)
		}
	}
	return set
}
