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

package extension

import (
	"fmt"
	"strings"
)

// Scope identifies the lifecycle level at which an extension is initialized.
// The values are bit flags when declared by an extension, so one extension can
// support multiple lifecycle levels. A single initialization always receives
// one concrete scope value.
type Scope uint8

const (
	// InstanceScope is the lifecycle of a dubbo.Instance.
	InstanceScope Scope = 1 << iota
	// ClientScope is the lifecycle of a client/consumer.
	ClientScope
	// ServerScope is the lifecycle of a server/provider.
	ServerScope
)

func (s Scope) valid() bool {
	return s == InstanceScope || s == ClientScope || s == ServerScope
}

// Supports reports whether declared contains one concrete supported scope.
// It is used for capability declarations; each runtime initialization still
// selects only one concrete scope.
func (declared Scope) Supports(scope Scope) bool {
	return declared != 0 && scope.valid() && declared&scope == scope
}

// Option applies typed configuration to one extension. The core only groups
// options by Prefix and applies them in declaration order; the extension owns
// the concrete configuration type and option semantics.
type Option interface {
	Prefix() string
	Apply(config any) error
}

// Definition is the contract between the core and an external extension.
// The core creates one config instance for each lifecycle initialization,
// decodes the extension's YAML subtree into it, applies typed options, and
// invokes only the callback selected by the concrete entry-point scope.
//
// Filter names are extension-internal factory keys registered through
// SetFilter. They are not part of the user-facing configuration API.
type Definition struct {
	Prefix          string
	Scopes          Scope
	New             func() any
	Decode          func(raw map[string]any, config any) error
	Init            func(config any) error
	ConsumerFilters func(config any) []string
	ProviderFilters func(config any) []string
}

// Supports reports whether the definition declares support for one concrete
// lifecycle scope.
func (d Definition) Supports(scope Scope) bool {
	return d.Scopes.Supports(scope)
}

func (d Definition) validate() error {
	if strings.TrimSpace(d.Prefix) == "" {
		return fmt.Errorf("extension: definition prefix is required")
	}
	if d.Scopes == 0 || d.Scopes&^(InstanceScope|ClientScope|ServerScope) != 0 {
		return fmt.Errorf("extension %q: definition scopes %d are invalid", d.Prefix, d.Scopes)
	}
	if d.New == nil {
		return fmt.Errorf("extension %q: New is required", d.Prefix)
	}
	return nil
}

var definitions = NewRegistry[Definition]("extension definition")

// Register adds an extension definition. Duplicate prefixes are rejected so
// side-effect import order cannot silently select different behavior.
func Register(def Definition) error {
	if err := def.validate(); err != nil {
		return err
	}
	if !definitions.RegisterIfAbsent(def.Prefix, def) {
		return fmt.Errorf("extension %q: definition is already registered", def.Prefix)
	}
	return nil
}

// MustRegister is intended for init functions in side-effect imported
// extension packages.
func MustRegister(def Definition) {
	if err := Register(def); err != nil {
		panic(err)
	}
}

// Lookup returns the definition registered for prefix.
func Lookup(prefix string) (Definition, bool) {
	return definitions.Get(prefix)
}

// Unregister removes a definition. It is intended for isolated tests.
func Unregister(prefix string) {
	definitions.Unregister(prefix)
}
