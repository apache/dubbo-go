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

// Package definition builds and publishes interface-level service definitions.
//
// A service definition describes the RPC contract of a single exported service
// interface: its methods, their signatures, and the structure of every type
// reachable from those signatures. Dubbo Admin consumes these definitions to
// render service documentation and to build generic-invocation request schemas.
//
// The JSON produced here is wire-compatible with Java's FullServiceDefinition
// (dubbo-common/.../definition/model/), because Admin deserializes both into the
// same ServiceProviderMetadata structure. Nothing in this package imports Admin
// code — the compatibility contract is the JSON shape alone, pinned by the
// golden tests in definition_test.go.
package definition

// ServiceDefinition is the interface-level contract published for one exported
// service. It mirrors Java's FullServiceDefinition.
//
// Java's codeSource field is deliberately omitted: Admin never reads it, and Go
// has no equivalent notion of a class file origin.
type ServiceDefinition struct {
	// CanonicalName is the service interface name, always taken verbatim from
	// the exported URL's Interface(). See BuildFromURL for why this must not be
	// re-derived by reflection.
	CanonicalName string `json:"canonicalName"`
	// Methods holds one entry per canonical method the builder considers safe to
	// expose. This is not the same set as Parameters["methods"]; see Parameters.
	Methods []MethodDefinition `json:"methods"`
	// Parameters carries the full provider URL parameter map, matching Java's
	// serviceDefinition.setParameters(url.getParameters()).
	//
	// Parameters["methods"] is the runtime method set from the exported URL and
	// includes the SwapCaseFirstRune aliases dubbo-go registers for Java
	// interop. Methods above holds only canonical names. The two intentionally
	// differ in size; consumers must not assume they match.
	Parameters map[string]string `json:"parameters"`
	// Types holds one entry for every composite type reachable from a method
	// signature, including pointer/slice/array/map wrappers. See the package
	// comment on typeCollector for why wrappers get their own entries.
	Types []TypeDefinition `json:"types"`
}

// MethodDefinition describes a single RPC method's signature.
type MethodDefinition struct {
	// Name is the canonical wire name: the MethodMapper mapping when one exists,
	// otherwise the Go exported method name. The SwapCaseFirstRune alias is
	// never published here even though it is routable at runtime.
	Name string `json:"name"`
	// ParameterTypes holds the type expression of each parameter, in order,
	// excluding the receiver and a leading context.Context.
	ParameterTypes []string `json:"parameterTypes"`
	// Parameters pairs each parameter type with a positional name.
	//
	// Java's equivalent field is @Deprecated and carries TypeDefinition elements
	// with no name at all. Go publishes {name, type} instead because Admin's
	// Parameter message has both fields, and Go reflection cannot recover source
	// parameter names — so the names here are always generated (arg0..argN).
	Parameters []ParameterDefinition `json:"parameters"`
	// ReturnType is the type expression of the non-error return value, or
	// VoidReturnType for a method that returns only error.
	ReturnType string `json:"returnType"`
}

// ParameterDefinition is one positional parameter of a method.
type ParameterDefinition struct {
	// Name is always generated as arg0..argN. Go reflection cannot see source
	// parameter names, and inventing business-looking names would mislead
	// callers building generic requests.
	Name string `json:"name"`
	Type string `json:"type"`
}

// TypeDefinition describes one type reachable from a method signature.
//
// The three optional fields are mutually exclusive in practice: Properties for
// structs, Items for pointer/slice/array/map wrappers, Enums for enumerated
// values (never populated by Go, retained for Java JSON compatibility).
type TypeDefinition struct {
	// Type is the type expression, in Go's native syntax. It is the key Admin
	// uses to resolve a ParameterTypes or ReturnType entry, so it must match
	// those strings byte for byte.
	Type string `json:"type"`
	// Items holds the element type expression for a wrapper type: the pointee
	// for *T, the element for []T and [N]T, the value type for map[string]T.
	Items []string `json:"items,omitempty"`
	// Enums is never populated by the Go builder. Go has no enum type that
	// reflection can enumerate; named integer constants are indistinguishable
	// from any other named integer at runtime.
	Enums []string `json:"enums,omitempty"`
	// Properties maps a struct field's wire name to its type expression. The
	// wire name follows the same rule MapGeneralizer uses when serializing, so
	// that a schema built from this map round-trips through generic invocation.
	Properties map[string]string `json:"properties,omitempty"`
}

// VoidReturnType is the ReturnType of a method that returns only error.
//
// An empty string would be indistinguishable from "the builder failed to
// resolve this type", so void gets an explicit marker. The spelling matches
// Java, where a void method's returnType is literally "void".
const VoidReturnType = "void"
