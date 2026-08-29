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

// Java type names used by the published contract.
//
// The definition speaks Java's type vocabulary rather than Go's. That is not a
// concession to any particular consumer: it is the vocabulary dubbo-go's own
// generic runtime already uses. filter/generic matches the caller-supplied
// $invoke types against protocol/dubbo/hessian2.GetJavaName output when it
// decides whether to unwrap a packed variadic tail, so a Go-spelled contract
// would describe names that dubbo-go itself does not recognize.
//
// Struct names are the exception and stay Go-derived unless the type declares a
// Java class name; see resolveStruct.
const (
	javaBoolean = "boolean"
	javaByte    = "byte"
	javaShort   = "short"
	javaInt     = "int"
	javaLong    = "long"
	javaFloat   = "float"
	javaDouble  = "double"
	javaString  = "java.lang.String"
	javaMap     = "java.util.Map"
)

// javaWrappers maps each primitive spelling to its boxed counterpart.
//
// Go's T versus *T is exactly Java's primitive versus wrapper distinction, and
// consumers read nullability off that: a primitive rejects null, a reference
// type accepts it. Expressing it through the type name means the contract needs
// no separate nullability flag, which Provider metadata has never carried.
var javaWrappers = map[string]string{
	javaBoolean: "java.lang.Boolean",
	javaByte:    "java.lang.Byte",
	javaShort:   "java.lang.Short",
	javaInt:     "java.lang.Integer",
	javaLong:    "java.lang.Long",
	javaFloat:   "java.lang.Float",
	javaDouble:  "java.lang.Double",
}

// javaScalarName returns the Java spelling of a Go scalar.
//
// Unsigned kinds widen to the next signed type that holds their whole range.
// The schema is then wider than the Go field on the low end — nothing stops a
// caller sending -1 for a uint16 — but the provider rejects that during
// realization, so the failure is a clean error rather than a wrapped value.
//
// uint and uint64 have no such landing spot: their range runs past Java's long.
// They are refused rather than published under an invented name, which is what
// the existing hessian helper does with its non-Java "unsigned long".
func javaScalarName(t reflect.Type, nullable bool) (string, error) {
	var primitive string
	switch t.Kind() {
	case reflect.Bool:
		primitive = javaBoolean
	case reflect.Int8:
		primitive = javaByte
	case reflect.Uint8:
		// byte, not short, even though Go's uint8 is 0..255 and Java's byte is
		// signed. []byte and []uint8 are one type in Go, so this choice also
		// decides byte slices — and those carry binary payloads that hessian2
		// already puts on the wire as Java byte[] (see GetClassDesc's "[B").
		// Keeping the container faithful matters more than the 128..255 range,
		// which the schema simply rejects.
		primitive = javaByte
	case reflect.Int16:
		primitive = javaShort
	case reflect.Uint16, reflect.Int32:
		primitive = javaInt
	case reflect.Uint32, reflect.Int, reflect.Int64:
		primitive = javaLong
	case reflect.Uint, reflect.Uint64:
		return "", unsupported(t.String(),
			"unsigned 64-bit integers exceed the range of every Java integer type")
	case reflect.Float32:
		primitive = javaFloat
	case reflect.Float64:
		primitive = javaDouble
	case reflect.String:
		// Already a reference type; there is no unboxed spelling to choose.
		return javaString, nil
	default:
		return "", unsupported(t.String(), "not a scalar kind")
	}

	if nullable {
		return javaWrappers[primitive], nil
	}
	return primitive, nil
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

	// Pointers carry nullability, not structure. Java expresses the same
	// distinction with primitive versus wrapper class rather than in the type
	// expression, so strip the indirection here and let the scalar branch pick
	// the right spelling. For struct, list and map the wrapper form is already
	// nullable, so the flag changes nothing.
	nullable := false
	for t.Kind() == reflect.Pointer {
		t = t.Elem()
		nullable = true
		depth++
		if depth > maxTypeDepth {
			return "", unsupported(t.String(),
				fmt.Sprintf("type nesting exceeds %d levels", maxTypeDepth))
		}
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
		// Scalars are published under their Java spelling even when the Go type
		// is named. A `type UserID int64` travels the generic wire as a bare
		// 64-bit integer, so publishing "UserID" would name something the caller
		// can neither construct nor recognize.
		return javaScalarName(t, nullable)

	case reflect.Slice, reflect.Array:
		// Array syntax, not List<T>. Java generics cannot hold a primitive, so
		// List<byte> would not be a type anyone could write; the array form is
		// valid for every element kind. It is also what
		// protocol/dubbo/hessian2.GetJavaName produces, which keeps the
		// published contract and the runtime's own spelling in agreement — and
		// it is what makes a Go []byte land on Java's byte[], the form hessian2
		// already puts on the wire.
		//
		// A Go array's length has no Java counterpart, so [N]T and []T are
		// spelled the same.
		return c.resolveContainer(t.Elem(), depth, func(elem string) string {
			return elem + "[]"
		})

	case reflect.Map:
		if t.Key().Kind() != reflect.String {
			return "", unsupported(t.String(),
				"only string-keyed maps can be expressed as JSON objects")
		}
		// The key is fixed at String rather than resolved: a named string key
		// still serializes as a JSON object key, and admitting map[UserID]T
		// would imply a key schema JSON cannot carry.
		return c.resolveMap(t.Elem(), depth)

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

// resolveContainer resolves a slice or array element and records a list entry
// whose single item is the element type.
//
// Container entries exist because Admin resolves a parameter by looking up its
// exact ParameterTypes string in types[], then walks items/properties from
// there. Publishing only the element T while the parameter reads List<T> leaves
// Admin with no path to T's fields.
//
// Exactly one item is what marks this as a collection rather than a map: Java's
// MapTypeBuilder appends one entry per actual type argument, so a Map yields two
// and consumers tell the two apart by arity, not by name.
func (c *typeCollector) resolveContainer(
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

// resolveMap records a map entry carrying both type arguments.
//
// Two items, key first. A single item would be read as a collection of that
// item, silently turning an object schema into an array one.
//
// The value is boxed: it sits inside a generic argument list, where Java admits
// only reference types. Map<java.lang.String,long> is not a type.
func (c *typeCollector) resolveMap(value reflect.Type, depth int) (string, error) {
	valueExpr, err := c.resolveAt(value, depth+1)
	if err != nil {
		return "", err
	}
	valueExpr = boxed(valueExpr)
	expr := javaMap + "<" + javaString + "," + valueExpr + ">"

	if _, seen := c.defs[expr]; !seen {
		c.put(expr, &TypeDefinition{Type: expr, Items: []string{javaString, valueExpr}})
	}
	return expr, nil
}

// boxed returns the wrapper spelling of a primitive, or expr unchanged when it
// is already a reference type.
func boxed(expr string) string {
	if wrapper, primitive := javaWrappers[expr]; primitive {
		return wrapper
	}
	return expr
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
// wire format that only one direction honors.
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

// javaClassNamed is implemented by types that declare their own Java class
// name. It is hessian.POJO, restated here so this package does not take a
// dependency on the codec just to read one method.
type javaClassNamed interface {
	JavaClassName() string
}

// namedTypeKey returns the name a struct is published under.
//
// A type that declares a Java class name is published under it: that is the
// name hessian2 puts in the wire form's "class" key and the name a Java peer
// knows the type by, so using anything else would describe a different type
// than the one on the wire. Everything else falls back to the Go import path
// plus type name — no Java name exists to borrow, and the path keeps two
// same-named types from different packages distinct, which is the same job
// Java's FQN does.
//
// Returns "" for an unnamed type.
func namedTypeKey(t reflect.Type) string {
	if declared := declaredJavaClassName(t); declared != "" {
		return declared
	}
	if t.Name() == "" {
		return ""
	}
	if t.PkgPath() == "" {
		return t.Name()
	}
	return t.PkgPath() + "." + t.Name()
}

// declaredJavaClassName reads JavaClassName from a type or its pointer,
// covering both receiver forms, and returns "" when the type does not declare
// one or panics trying.
func declaredJavaClassName(t reflect.Type) (name string) {
	if t.Kind() != reflect.Struct {
		return ""
	}
	named, ok := reflect.New(t).Interface().(javaClassNamed)
	if !ok {
		return ""
	}
	defer func() {
		// A JavaClassName that dereferences its receiver's fields would panic on
		// the zero value. Fall back to the Go name rather than fail the build.
		if recover() != nil {
			name = ""
		}
	}()
	return named.JavaClassName()
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
