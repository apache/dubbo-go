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

package definition

import (
	"errors"
	"fmt"
	"reflect"
	"sort"
	"strings"
	"unicode"
	"unicode/utf8"
)

// maxTypeDepth bounds structural nesting. The visited set already terminates
// reference cycles, so this only catches pathologically deep composite types
// (a [][][]...[]T built by code generation, say) before they exhaust the stack.
const maxTypeDepth = 64

// unsupportedError marks a type or method the builder refuses to publish.
//
// Refusing is deliberate: a schema that silently degrades an unrepresentable
// type produces requests the provider cannot realize, and the failure surfaces
// at call time on the Admin side rather than at publish time here. Java has no
// equivalent problem — erasure yields an incomplete TypeDefinition, but Java has
// no chan or func to describe in the first place.
type unsupportedError struct {
	subject string
	reason  string
}

func (e *unsupportedError) Error() string {
	return fmt.Sprintf("%s is not supported: %s", e.subject, e.reason)
}

func unsupported(subject, reason string) error {
	return &unsupportedError{subject: subject, reason: reason}
}

// IsUnsupported reports whether err marks a type or method the builder
// deliberately declined to publish, as opposed to an internal failure.
func IsUnsupported(err error) bool {
	var target *unsupportedError
	return errors.As(err, &target)
}

// blockedNamedTypes are named types the Generalizer or hessian2 codec treats
// specially, but for which the MCP proposal has not yet defined a JSON wire
// representation. Expanding them as plain structs would publish their internal
// layout (time.Time's wall/ext/loc, big.Int's neg/abs) as if it were the
// contract, which is worse than not publishing the method at all.
//
// Named scalars are deliberately absent: time.Duration is an int64 that
// round-trips as a number like any other, so the scalar branch below handles it
// correctly. Only types whose structure would be misrepresented belong here.
var blockedNamedTypes = map[string]string{
	"time.Time":                "no JSON wire representation is defined for time.Time yet",
	"math/big.Int":             "no JSON wire representation is defined for math/big types yet",
	"math/big.Float":           "no JSON wire representation is defined for math/big types yet",
	"math/big.Rat":             "no JSON wire representation is defined for math/big types yet",
	"encoding/json.RawMessage": "raw JSON has no declarable structure",
}

// typeCollector resolves reflect.Types into type expressions and accumulates a
// TypeDefinition for every composite type it walks through.
type typeCollector struct {
	defs  map[string]*TypeDefinition
	order []string
}

func newTypeCollector() *typeCollector {
	return &typeCollector{defs: make(map[string]*TypeDefinition)}
}

// resolve returns the type expression for t, recording definitions for t and
// everything reachable from it.
func (c *typeCollector) resolve(t reflect.Type) (string, error) {
	return c.resolveAt(t, 0)
}

func (c *typeCollector) resolveAt(t reflect.Type, depth int) (string, error) {
	if t == nil {
		return "", unsupported("<nil type>", "type is nil")
	}
	if depth > maxTypeDepth {
		return "", unsupported(t.String(), fmt.Sprintf("type nesting exceeds %d levels", maxTypeDepth))
	}

	if name := namedTypeKey(t); name != "" {
		if reason, blocked := blockedNamedTypes[name]; blocked {
			return "", unsupported(name, reason)
		}
	}

	switch t.Kind() {
	case reflect.Bool, reflect.String,
		reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
		reflect.Float32, reflect.Float64:
		// Scalars are published under their builtin spelling even when the type
		// is named. A `type UserID int64` travels the generic wire as a bare
		// int64 — MapGeneralizer's objToMap returns it unchanged through the
		// default branch — so publishing "UserID" would name something the
		// caller can neither construct nor recognize. Only structs keep their
		// declared name, because only structs have a structure to describe.
		return t.Kind().String(), nil

	case reflect.Pointer:
		return c.resolveWrapper(t.Elem(), depth, func(elem string) string {
			return "*" + elem
		})

	case reflect.Slice:
		return c.resolveWrapper(t.Elem(), depth, func(elem string) string {
			return "[]" + elem
		})

	case reflect.Array:
		return c.resolveWrapper(t.Elem(), depth, func(elem string) string {
			return fmt.Sprintf("[%d]%s", t.Len(), elem)
		})

	case reflect.Map:
		if t.Key().Kind() != reflect.String {
			return "", unsupported(t.String(),
				"only string-keyed maps can be expressed as JSON objects")
		}
		// The key type is fixed at map[string] rather than resolved: a named
		// string key still serializes as a JSON object key, and admitting
		// map[UserID]T would imply a key schema that JSON cannot carry.
		return c.resolveWrapper(t.Elem(), depth, func(elem string) string {
			return "map[string]" + elem
		})

	case reflect.Struct:
		return c.resolveStruct(t, depth)

	case reflect.Interface:
		// Including the empty interface would publish a field whose schema is
		// "anything", which Admin cannot turn into a request form and the
		// provider cannot realize into a concrete Go value.
		return "", unsupported(t.String(), "interface types have no declarable structure")

	case reflect.Chan, reflect.Func, reflect.UnsafePointer:
		return "", unsupported(t.String(), "type cannot cross an RPC boundary")

	case reflect.Complex64, reflect.Complex128:
		return "", unsupported(t.String(), "complex numbers have no cross-language representation")

	case reflect.Uintptr:
		return "", unsupported(t.String(), "uintptr is a process-local address")

	default:
		return "", unsupported(t.String(), "unhandled reflect kind "+t.Kind().String())
	}
}

// resolveWrapper resolves a pointer/slice/array/map element and records a
// wrapper TypeDefinition linking the composite expression to its element.
//
// Wrappers get their own entries because Admin resolves a parameter by looking
// up its exact ParameterTypes string in types[], then walks items/properties
// from there. Publishing only the element T while the parameter reads []T
// leaves Admin with no path to T's fields.
func (c *typeCollector) resolveWrapper(
	elem reflect.Type,
	depth int,
	compose func(string) string,
) (string, error) {
	elemExpr, err := c.resolveAt(elem, depth+1)
	if err != nil {
		return "", err
	}
	expr := compose(elemExpr)

	if _, seen := c.defs[expr]; !seen {
		c.put(expr, &TypeDefinition{Type: expr, Items: []string{elemExpr}})
	}
	return expr, nil
}

func (c *typeCollector) resolveStruct(t reflect.Type, depth int) (string, error) {
	expr := namedTypeKey(t)
	if expr == "" {
		// An anonymous struct literal in a signature has no name for Admin to
		// key on, and no stable identity across builds.
		return "", unsupported(t.String(), "anonymous struct types cannot be named in a contract")
	}

	// Reserve the key before descending so a self-referential type terminates.
	// The placeholder is replaced below once fields resolve; on failure it is
	// removed again so a later, successful path can retry the same type.
	if _, seen := c.defs[expr]; seen {
		return expr, nil
	}
	c.put(expr, &TypeDefinition{Type: expr})

	properties, err := c.structProperties(t, expr, depth)
	if err != nil {
		c.remove(expr)
		return "", err
	}
	c.defs[expr].Properties = properties
	return expr, nil
}

func (c *typeCollector) structProperties(t reflect.Type, expr string, depth int) (map[string]string, error) {
	properties := make(map[string]string)
	// matchSets records, per wire name, the field that claimed it. mapstructure
	// matches case-insensitively, so collisions are detected on the folded name.
	matchSets := make(map[string]string)

	for i := range t.NumField() {
		field := t.Field(i)
		if field.PkgPath != "" {
			// Unexported. MapGeneralizer skips these too (CanInterface is
			// false), so they are genuinely absent from the wire format.
			continue
		}

		wireName, err := fieldWireName(field)
		if err != nil {
			return nil, err
		}

		// A field is reachable under its wire name and, for backwards
		// compatibility with generic callers written before m tags existed, its
		// original Go field name — both case-insensitively. Two fields whose
		// reachable sets overlap make both the schema property and the Realize
		// target depend on field iteration order.
		for _, alias := range []string{wireName, field.Name} {
			folded := strings.ToLower(alias)
			if owner, taken := matchSets[folded]; taken && owner != field.Name {
				return nil, unsupported(expr, fmt.Sprintf(
					"fields %q and %q are both reachable as %q", owner, field.Name, alias))
			}
			matchSets[folded] = field.Name
		}

		fieldExpr, err := c.resolveAt(field.Type, depth+1)
		if err != nil {
			return nil, err
		}
		properties[wireName] = fieldExpr
	}

	return properties, nil
}

// fieldWireName returns the map key MapGeneralizer.setInMap would emit for this
// field, rejecting every case where Generalize and Realize currently disagree.
//
// These rejections are the proposal's transition constraint. They lift once the
// shared GenericFieldName resolver lands and both directions read the tag the
// same way; until then, publishing a schema for these fields would describe a
// wire format that only one direction honours.
func fieldWireName(field reflect.StructField) (string, error) {
	tag, tagged := field.Tag.Lookup("m")
	if !tagged || tag == "" {
		first, size := utf8.DecodeRuneInString(field.Name)
		if size > 1 {
			// toUnexport lowercases via strings.ToLower(name[:1]), a byte slice.
			// On a multi-byte leading rune that splits the rune and produces
			// invalid UTF-8, so there is no wire name to publish.
			return "", unsupported(field.Name,
				"non-ASCII field names are mangled by the current Generalize path")
		}
		return string(unicode.ToLower(first)) + field.Name[size:], nil
	}

	if tag == "-" {
		// Generalize writes a literal "-" key; mapstructure reads "-" as skip.
		// The field is therefore emitted but never read back.
		return "", unsupported(field.Name,
			`m:"-" is written as a literal "-" key by Generalize but skipped by Realize`)
	}

	if strings.Contains(tag, ",") {
		// Generalize uses the entire tag as the key ("name,omitempty");
		// mapstructure parses off the option and uses "name".
		return "", unsupported(field.Name,
			fmt.Sprintf("m tag option in %q is interpreted differently by Generalize and Realize", tag))
	}

	return tag, nil
}

// namedTypeKey returns the fully qualified name of a named type, or "" if the
// type is unnamed. Package path is included so two same-named types from
// different packages stay distinct, matching Java's use of the class FQN.
func namedTypeKey(t reflect.Type) string {
	if t.Name() == "" {
		return ""
	}
	if t.PkgPath() == "" {
		return t.Name()
	}
	return t.PkgPath() + "." + t.Name()
}

func (c *typeCollector) put(expr string, def *TypeDefinition) {
	c.defs[expr] = def
	c.order = append(c.order, expr)
}

// merge copies other's definitions into c without overwriting existing entries.
//
// The builder resolves each method into a throwaway collector and merges only on
// success, so a method rejected halfway through — after some of its parameter
// types already resolved — leaves no orphan entries describing types no
// published method can reach.
func (c *typeCollector) merge(other *typeCollector) {
	for _, expr := range other.order {
		if _, seen := c.defs[expr]; !seen {
			c.put(expr, other.defs[expr])
		}
	}
}

func (c *typeCollector) remove(expr string) {
	delete(c.defs, expr)
	for i, name := range c.order {
		if name == expr {
			c.order = append(c.order[:i], c.order[i+1:]...)
			break
		}
	}
}

// definitions returns the collected types sorted by expression.
//
// Sorting rather than preserving insertion order keeps the published JSON
// byte-stable across runs: Go randomizes map iteration, and method resolution
// order already varies, so an unsorted slice would produce a different document
// on every restart and defeat idempotent republishing.
func (c *typeCollector) definitions() []TypeDefinition {
	exprs := make([]string, 0, len(c.defs))
	for expr := range c.defs {
		exprs = append(exprs, expr)
	}
	sort.Strings(exprs)

	out := make([]TypeDefinition, 0, len(exprs))
	for _, expr := range exprs {
		out = append(out, *c.defs[expr])
	}
	return out
}
