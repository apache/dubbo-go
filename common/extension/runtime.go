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

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/filter"
)

// Scope identifies an extension lifecycle level. Definition.Scopes uses the
// values as a bit mask; Context.Scope always contains one concrete scope.
type Scope uint8

const (
	// InstanceScope is the lifecycle of a dubbo.Instance.
	InstanceScope Scope = 1 << iota
	// ClientScope is the lifecycle of a client/consumer.
	ClientScope
	// ServerScope is the lifecycle of a server/provider.
	ServerScope
)

// RoleNone identifies an Instance context, which has no consumer/provider
// role. It is intentionally outside common.RoleType's URL role constants.
const RoleNone common.RoleType = -1

// Resource identifies the service or method to which an extension is bound.
// The core constructs Resource values and extensions treat them as read-only.
type Resource struct {
	ServiceKey string
	Interface  string
	Method     string
	Group      string
	Version    string
}

// Context carries the lifecycle, role, extension configuration, and optional
// RPC resource supplied to an extension callback. A nil Resource means that
// the lifecycle context has not yet been bound to a concrete RPC resource.
type Context struct {
	Scope    Scope
	Role     common.RoleType
	Config   any
	Resource *Resource
}

// Validate verifies that Scope and Role form one of the supported context
// combinations. It is intentionally independent of any concrete extension.
func (c Context) Validate() error {
	switch c.Scope {
	case InstanceScope:
		if c.Role != RoleNone {
			return fmt.Errorf("extension: InstanceScope requires RoleNone, got role %d", c.Role)
		}
	case ClientScope:
		if c.Role != common.CONSUMER {
			return fmt.Errorf("extension: ClientScope requires consumer role, got role %d", c.Role)
		}
	case ServerScope:
		if c.Role != common.PROVIDER {
			return fmt.Errorf("extension: ServerScope requires provider role, got role %d", c.Role)
		}
	default:
		return fmt.Errorf("extension: invalid context scope %d", c.Scope)
	}
	return nil
}

// RawNode is the parser-independent view of an extension configuration tree.
// Child performs an exact key lookup; dots and colons in a key have no path
// semantics.
type RawNode interface {
	Child(key string) (RawNode, bool)
	Value() any
	Present() bool
}

// RawConfig contains the complete extension subtree and the scope-selected
// consumer/provider view. The core owns the tree and extensions must treat it
// as immutable.
type RawConfig struct {
	Full     RawNode
	Selected RawNode
}

// Option configures one extension. The concrete configuration type is owned by
// the extension; the core only groups options by Prefix and applies them in
// declaration order.
type Option interface {
	Prefix() string
	Apply(config any) error
}

// FilterSpec describes one filter contribution from an extension. ID is an
// extension-owned internal identity used by the core for deduplication and
// diagnostics; it is not a user-facing filter name. Factory is required and
// creates the filter for the current resource context.
type FilterSpec struct {
	ID      string
	Factory func() filter.Filter
	Order   int
}

// Definition describes an external extension without exposing its concrete
// configuration or runtime behavior to the core. Definitions are immutable
// after registration.
type Definition struct {
	Prefix    string
	Scopes    Scope
	NewConfig func() any
	Decode    func(RawConfig, any) error
	Init      func(*Context) error
	Filters   func(*Context) ([]FilterSpec, error)
	Close     func(*Context) error
}

// Supports reports whether the definition declares support for scope.
func (d Definition) Supports(scope Scope) bool {
	return scope != 0 && d.Scopes&scope == scope
}

func (d Definition) validate() error {
	if strings.TrimSpace(d.Prefix) == "" {
		return fmt.Errorf("extension: definition prefix is required")
	}
	if d.Scopes == 0 || d.Scopes&^(InstanceScope|ClientScope|ServerScope) != 0 {
		return fmt.Errorf("extension %q: definition scopes %d are invalid", d.Prefix, d.Scopes)
	}
	if d.NewConfig == nil {
		return fmt.Errorf("extension %q: NewConfig is required", d.Prefix)
	}
	return nil
}

var definitions = NewRegistry[Definition]("extension definition")

// Register adds an immutable extension Definition. Duplicate Prefix values are
// rejected so registration order cannot silently change runtime behavior.
func Register(def Definition) error {
	if err := def.validate(); err != nil {
		return err
	}
	if !definitions.RegisterIfAbsent(def.Prefix, def) {
		return fmt.Errorf("extension %q: definition is already registered", def.Prefix)
	}
	return nil
}

// MustRegister registers a Definition and panics when the Definition is
// invalid or its Prefix has already been registered. It is intended for init
// functions in side-effect imported extension packages.
func MustRegister(def Definition) {
	if err := Register(def); err != nil {
		panic(err)
	}
}

// Lookup returns the immutable Definition registered for prefix.
func Lookup(prefix string) (Definition, bool) {
	return definitions.Get(prefix)
}

// Unregister removes a Definition. It is primarily useful for isolated tests;
// production extensions should register once during package initialization.
func Unregister(prefix string) {
	definitions.Unregister(prefix)
}
