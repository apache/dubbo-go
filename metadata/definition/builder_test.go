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
	"context"
	"reflect"
	"testing"
	"time"
)

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
)

// ---------------------------------------------------------------------------
// fixtures
// ---------------------------------------------------------------------------
type Address struct {
	City   string
	Zip    string `m:"zipCode"`
	hidden string //nolint:unused // asserts unexported fields stay out of the contract
}

type User struct {
	Name     string
	Age      int32
	Home     *Address
	Tags     []string
	Scores   map[string]float64
	Quadrant [4]int
}

// Node is self-referential, to prove the collector terminates on cycles.
type Node struct {
	Label string
	Next  *Node
}

type basicService struct{}

func (s *basicService) GetUser(ctx context.Context, id string) (*User, error) {
	return nil, nil
}

// Ping returns only an error, so its returnType must be the explicit void marker.
func (s *basicService) Ping(ctx context.Context) error { return nil }

// NoContext omits context.Context entirely; the receiver is still skipped.
func (s *basicService) NoContext(a string, b int64) (bool, error) { return false, nil }

// Walk exercises the recursive type.
func (s *basicService) Walk(ctx context.Context, n *Node) (*Node, error) { return nil, nil }

type variadicService struct{}

func (s *variadicService) Sum(ctx context.Context, nums ...int) (int, error) { return 0, nil }
func (s *variadicService) Fine(ctx context.Context, n int) (int, error)      { return 0, nil }

type chanService struct{}

func (s *chanService) Stream(ctx context.Context, ch chan string) error { return nil }
func (s *chanService) Fine(ctx context.Context, n int) (int, error)     { return 0, nil }

type intMapService struct{}

func (s *intMapService) Lookup(ctx context.Context, m map[int]string) error { return nil }

type timeService struct{}

func (s *timeService) At(ctx context.Context, t time.Time) error { return nil }

type ifaceService struct{}

func (s *ifaceService) Any(ctx context.Context, v any) error { return nil }

// mapperService renames methods; only the mapped name may be published.
type mapperService struct{}

func (s *mapperService) MethodMapper() map[string]string {
	return map[string]string{"OriginalName": "renamed"}
}
func (s *mapperService) OriginalName(ctx context.Context, a string) error { return nil }

// collidingService maps two distinct methods onto first-rune-case variants of
// one another, so their runtime wire-name sets intersect.
type collidingService struct{}

func (s *collidingService) MethodMapper() map[string]string {
	return map[string]string{"Alpha": "echo", "Beta": "Echo"}
}
func (s *collidingService) Alpha(ctx context.Context, a string) error { return nil }
func (s *collidingService) Beta(ctx context.Context, a string) error  { return nil }
func (s *collidingService) Gamma(ctx context.Context, a string) error { return nil }

// ---------------------------------------------------------------------------
// helpers
// ---------------------------------------------------------------------------

func testURL(t *testing.T, iface, version, group string) *common.URL {
	t.Helper()
	opts := []common.Option{
		common.WithProtocol("dubbo"),
		common.WithIp("127.0.0.1"),
		common.WithPort("20000"),
		common.WithParamsValue(constant.InterfaceKey, iface),
		common.WithParamsValue(constant.ApplicationKey, "test-app"),
		common.WithParamsValue(constant.ReleaseKey, "dubbo-golang-3.3.0"),
		common.WithParamsValue(constant.SideKey, "provider"),
	}
	if version != "" {
		opts = append(opts, common.WithParamsValue(constant.VersionKey, version))
	}
	if group != "" {
		opts = append(opts, common.WithParamsValue(constant.GroupKey, group))
	}
	return common.NewURLWithOptions(opts...)
}

func build(t *testing.T, svc any) (*ServiceDefinition, []SkippedMethod) {
	t.Helper()
	def, skips, err := BuildFromURL(testURL(t, "org.example.Svc", "", ""), reflect.TypeOf(svc))
	require.NoError(t, err)
	return def, skips
}

func methodByName(t *testing.T, def *ServiceDefinition, name string) MethodDefinition {
	t.Helper()
	for _, m := range def.Methods {
		if m.Name == name {
			return m
		}
	}
	t.Fatalf("method %q not found in %+v", name, def.Methods)
	return MethodDefinition{}
}

func typeByName(t *testing.T, def *ServiceDefinition, name string) TypeDefinition {
	t.Helper()
	for _, ty := range def.Types {
		if ty.Type == name {
			return ty
		}
	}
	t.Fatalf("type %q not found; have %v", name, typeNames(def))
	return TypeDefinition{}
}

func typeNames(def *ServiceDefinition) []string {
	names := make([]string, 0, len(def.Types))
	for _, ty := range def.Types {
		names = append(names, ty.Type)
	}
	return names
}

func skipReasons(skips []SkippedMethod) map[string]string {
	out := make(map[string]string, len(skips))
	for _, s := range skips {
		out[s.Name] = s.Reason
	}
	return out
}

// ---------------------------------------------------------------------------
// signature trimming
// ---------------------------------------------------------------------------

func TestBuildTrimsReceiverAndContext(t *testing.T) {
	def, _ := build(t, &basicService{})

	get := methodByName(t, def, "GetUser")
	assert.Equal(t, []string{"java.lang.String"}, get.ParameterTypes,
		"receiver and leading context.Context must not appear as parameters")

	noCtx := methodByName(t, def, "NoContext")
	assert.Equal(t, []string{"java.lang.String", "long"}, noCtx.ParameterTypes,
		"a method without context.Context keeps all its declared parameters")
}

func TestBuildReturnTypes(t *testing.T) {
	def, _ := build(t, &basicService{})

	// *User is published as User: the pointer only says the value may be
	// absent, which Java reads off the type being a reference type.
	assert.Equal(t, userType, methodByName(t, def, "GetUser").ReturnType)
	assert.Equal(t, "boolean", methodByName(t, def, "NoContext").ReturnType)

	// An error-only method must be distinguishable from a failed resolution.
	ping := methodByName(t, def, "Ping")
	assert.Equal(t, VoidReturnType, ping.ReturnType)
	assert.NotEmpty(t, ping.ReturnType)
	assert.Empty(t, ping.ParameterTypes)
}

func TestBuildParameterNamesArePositional(t *testing.T) {
	def, _ := build(t, &basicService{})

	noCtx := methodByName(t, def, "NoContext")
	require.Len(t, noCtx.Parameters, 2)
	assert.Equal(t, "arg0", noCtx.Parameters[0].Name)
	assert.Equal(t, "arg1", noCtx.Parameters[1].Name)
	assert.Equal(t, "java.lang.String", noCtx.Parameters[0].Type)
	assert.Equal(t, "long", noCtx.Parameters[1].Type)

	for i, p := range noCtx.Parameters {
		assert.Equal(t, noCtx.ParameterTypes[i], p.Type,
			"Parameters and ParameterTypes must agree positionally")
	}
}

// ---------------------------------------------------------------------------
// type expression and container entries
// ---------------------------------------------------------------------------

const (
	userType = "dubbo.apache.org/dubbo-go/v3/metadata/definition.User"
	addrType = "dubbo.apache.org/dubbo-go/v3/metadata/definition.Address"
	nodeType = "dubbo.apache.org/dubbo-go/v3/metadata/definition.Node"
)

func TestBuildScalarsUseJavaSpelling(t *testing.T) {
	// The contract speaks Java's type vocabulary, which is the vocabulary
	// dubbo-go's own generic runtime matches against.
	def, _ := build(t, &basicService{})

	user := typeByName(t, def, userType)
	assert.Equal(t, "java.lang.String", user.Properties["name"])
	assert.Equal(t, "int", user.Properties["age"], "Go int32 is Java int")
}

func TestBuildPointerBecomesWrapperOrReference(t *testing.T) {
	def, _ := build(t, &basicService{})

	// A pointer to a struct is just the struct: Java objects are already
	// nullable, so there is nothing extra to say.
	assert.Equal(t, addrType, typeByName(t, def, userType).Properties["home"])
	for _, ty := range def.Types {
		assert.NotContains(t, ty.Type, "*", "pointers must not survive into type names")
	}

	// A pointer to a scalar boxes it, which is exactly how Java distinguishes
	// a nullable value from a primitive.
	c := newTypeCollector()
	expr, err := c.resolve(reflect.TypeFor[*int64]())
	require.NoError(t, err)
	assert.Equal(t, "java.lang.Long", expr)

	expr, err = c.resolve(reflect.TypeFor[int64]())
	require.NoError(t, err)
	assert.Equal(t, "long", expr)
}

// TestBuildContainerArity is the distinction consumers rely on: Java's
// MapTypeBuilder appends one entry per actual type argument, so a collection
// carries one item and a map carries two. Publishing a map with a single item
// would have it read as an array of that item.
func TestBuildContainerArity(t *testing.T) {
	def, _ := build(t, &basicService{})

	user := typeByName(t, def, userType)
	assert.Equal(t, "java.lang.String[]", user.Properties["tags"])
	assert.Equal(t, "java.util.Map<java.lang.String,java.lang.Double>", user.Properties["scores"])
	assert.Equal(t, "long[]", user.Properties["quadrant"], "a Go array has no length in Java")

	list := typeByName(t, def, "java.lang.String[]")
	require.Len(t, list.Items, 1, "a collection carries exactly one item")
	assert.Equal(t, []string{"java.lang.String"}, list.Items)

	m := typeByName(t, def, "java.util.Map<java.lang.String,java.lang.Double>")
	require.Len(t, m.Items, 2, "a map carries key and value")
	assert.Equal(t, []string{"java.lang.String", "java.lang.Double"}, m.Items,
		"the value is boxed because Java generics admit only reference types")
}

func TestBuildUint8WidensSoTheWholeRangeIsCallable(t *testing.T) {
	// Java's byte is signed, so spelling uint8 as byte would leave 128..255
	// rejected by a consumer's range check before the call left. []byte and
	// []uint8 are one type in Go, so the slice follows.
	c := newTypeCollector()

	expr, err := c.resolve(reflect.TypeFor[uint8]())
	require.NoError(t, err)
	assert.Equal(t, "short", expr)

	expr, err = c.resolve(reflect.TypeFor[[]byte]())
	require.NoError(t, err)
	assert.Equal(t, "short[]", expr)
	assert.Equal(t, []string{"short"}, c.defs["short[]"].Items)
}

func TestBuildRejectsUnsigned64(t *testing.T) {
	// No Java integer type holds the range, and publishing the existing
	// hessian helper's non-Java "unsigned long" would name nothing.
	for _, typ := range []reflect.Type{
		reflect.TypeFor[uint64](),
		reflect.TypeFor[uint](),
	} {
		_, err := newTypeCollector().resolve(typ)
		require.Error(t, err, "%s", typ)
		assert.True(t, IsUnsupported(err))
		assert.Contains(t, err.Error(), "exceed the range")
	}
}

func TestBuildWidensSmallUnsigned(t *testing.T) {
	// Widening keeps the whole Go range representable; the schema is looser on
	// the low end, which realization rejects rather than wraps.
	for typ, want := range map[reflect.Type]string{
		reflect.TypeFor[uint8]():  "short",
		reflect.TypeFor[uint16](): "int",
		reflect.TypeFor[uint32](): "long",
	} {
		expr, err := newTypeCollector().resolve(typ)
		require.NoError(t, err, "%s", typ)
		assert.Equal(t, want, expr, "%s", typ)
	}
}

func TestBuildUsesDeclaredJavaClassName(t *testing.T) {
	// A type that declares a Java class name is published under it: that is the
	// name hessian2 writes into the wire form and the name a Java peer knows.
	c := newTypeCollector()
	expr, err := c.resolve(reflect.TypeFor[javaNamed]())
	require.NoError(t, err)
	assert.Equal(t, "com.example.Named", expr)
	assert.Equal(t, map[string]string{"label": "java.lang.String"}, c.defs[expr].Properties)
}

func TestBuildStructPropertiesFollowGeneralizerNaming(t *testing.T) {
	def, _ := build(t, &basicService{})

	addr := typeByName(t, def, addrType)
	assert.Equal(t, map[string]string{
		"city":    "java.lang.String", // no tag: first rune lowercased
		"zipCode": "java.lang.String", // m tag wins verbatim
	}, addr.Properties)
	assert.NotContains(t, addr.Properties, "hidden", "unexported fields are not on the wire")
	assert.NotContains(t, addr.Properties, "Zip", "the Go name is not the wire name when a tag exists")
}

func TestBuildRecursiveTypeTerminates(t *testing.T) {
	def, _ := build(t, &basicService{})

	node := typeByName(t, def, nodeType)
	assert.Equal(t, map[string]string{
		"label": "java.lang.String",
		"next":  nodeType,
	}, node.Properties, "the cycle is preserved as a reference, not expanded")
}

func TestBuildNamedScalarUsesUnderlyingType(t *testing.T) {
	// A named scalar travels the generic wire as its underlying value, so the
	// contract names that rather than the Go type name.
	c := newTypeCollector()
	expr, err := c.resolve(reflect.TypeFor[time.Month]())
	require.NoError(t, err)
	assert.Equal(t, "long", expr, "Go int is 64-bit here, matching getBasicJavaName")
}

// ---------------------------------------------------------------------------
// canonical names and conflicts
// ---------------------------------------------------------------------------

func TestBuildCanonicalNameComesFromURL(t *testing.T) {
	u := testURL(t, "org.example.UserService", "1.0.0", "g1")
	def, _, err := BuildFromURL(u, reflect.TypeFor[*basicService]())
	require.NoError(t, err)

	assert.Equal(t, "org.example.UserService", def.CanonicalName,
		"the interface name is taken from the URL, never re-derived from the struct")
	assert.Equal(t, "1.0.0", def.Parameters[constant.VersionKey])
	assert.Equal(t, "g1", def.Parameters[constant.GroupKey])
	assert.Equal(t, "test-app", def.Parameters[constant.ApplicationKey])
	assert.Equal(t, "dubbo-golang-3.3.0", def.Parameters[constant.ReleaseKey],
		"Admin identifies Go providers by this prefix")
	assert.NotContains(t, def.Parameters, "language", "no Go-only metadata dialect")
}

func TestBuildPublishesMappedNameOnceWithoutAlias(t *testing.T) {
	def, _ := build(t, &mapperService{})

	names := make([]string, 0, len(def.Methods))
	for _, m := range def.Methods {
		names = append(names, m.Name)
	}
	assert.Equal(t, []string{"renamed"}, names,
		"the mapped name is published once; neither the Go name nor the swapped-case alias appears")
}

func TestBuildDropsBothSidesOfANameCollision(t *testing.T) {
	def, skips := build(t, &collidingService{})

	names := make([]string, 0, len(def.Methods))
	for _, m := range def.Methods {
		names = append(names, m.Name)
	}
	assert.Equal(t, []string{"Gamma"}, names,
		"echo and Echo collide through the alias mechanism; neither may be published")

	reasons := skipReasons(skips)
	assert.Contains(t, reasons, "echo")
	assert.Contains(t, reasons, "Echo")
	assert.Contains(t, reasons["echo"], "routable")
}

func TestRuntimeStillRegistersCollidingMethods(t *testing.T) {
	// The builder refuses to publish a collision, but registration must keep
	// working so existing services still start after an upgrade.
	methods := common.CanonicalMethods(reflect.TypeFor[*collidingService]())
	conflicts := common.MethodNameConflicts(methods)
	require.Len(t, conflicts, 1)
	assert.Equal(t, "Echo", conflicts[0].WireName)
}

// ---------------------------------------------------------------------------
// unsupported types
// ---------------------------------------------------------------------------

func TestBuildRejectsUnsupportedMethods(t *testing.T) {
	cases := []struct {
		name    string
		svc     any
		method  string
		wantMsg string
	}{
		{"variadic", &variadicService{}, "Sum", "variadic"},
		{"chan", &chanService{}, "Stream", "cannot cross an RPC boundary"},
		{"non-string map key", &intMapService{}, "Lookup", "string-keyed"},
		{"time.Time", &timeService{}, "At", "time.Time"},
		{"empty interface", &ifaceService{}, "Any", "no declarable structure"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			def, skips := build(t, tc.svc)

			for _, m := range def.Methods {
				assert.NotEqual(t, tc.method, m.Name, "unsupported method must not be published")
			}
			reasons := skipReasons(skips)
			require.Contains(t, reasons, tc.method)
			assert.Contains(t, reasons[tc.method], tc.wantMsg)
		})
	}
}

func TestBuildKeepsSupportedSiblingsOfRejectedMethods(t *testing.T) {
	// One bad method must not take the whole service down with it.
	def, skips := build(t, &variadicService{})

	assert.Equal(t, []string{"Fine"}, []string{def.Methods[0].Name})
	assert.Len(t, def.Methods, 1)
	assert.Contains(t, skipReasons(skips), "Sum")
}

func TestRejectedMethodLeavesNoOrphanTypes(t *testing.T) {
	def, _ := build(t, &chanService{})
	assert.NotContains(t, typeNames(def), "chan string")
}

// ---------------------------------------------------------------------------
// field-name transition constraints (proposal §4.2)
// ---------------------------------------------------------------------------

type dashTagged struct {
	Keep string
	Drop string `m:"-"`
}

type optionTagged struct {
	Name string `m:"name,omitempty"`
}

type nonASCIINamed struct {
	Ünicode string
}

type aliasColliding struct {
	Name  string
	Other string `m:"name"`
}

func TestFieldNameTransitionConstraints(t *testing.T) {
	cases := []struct {
		name    string
		typ     reflect.Type
		wantMsg string
	}{
		{`m:"-"`, reflect.TypeFor[dashTagged](), `skipped by Realize`},
		{"tag option", reflect.TypeFor[optionTagged](), "interpreted differently"},
		{"non-ASCII field", reflect.TypeFor[nonASCIINamed](), "non-ASCII"},
		{"canonical/legacy collision", reflect.TypeFor[aliasColliding](), "both reachable as"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := newTypeCollector().resolve(tc.typ)
			require.Error(t, err)
			assert.True(t, IsUnsupported(err), "must be an unsupported marker, not an internal error")
			assert.Contains(t, err.Error(), tc.wantMsg)
		})
	}
}

func TestUnsupportedIsDistinguishableFromInternalError(t *testing.T) {
	_, err := newTypeCollector().resolve(reflect.TypeFor[chan int]())
	require.Error(t, err)
	assert.True(t, IsUnsupported(err))
}

// ---------------------------------------------------------------------------
// determinism
// ---------------------------------------------------------------------------

func TestBuildIsDeterministic(t *testing.T) {
	// Republishing identical content on every restart is what keeps the
	// metadata center from churning, so map iteration must not leak into output.
	first, _, err := BuildFromURL(testURL(t, "org.example.Svc", "", ""), reflect.TypeFor[*basicService]())
	require.NoError(t, err)

	for range 20 {
		next, _, err := BuildFromURL(testURL(t, "org.example.Svc", "", ""), reflect.TypeFor[*basicService]())
		require.NoError(t, err)
		assert.Equal(t, first.Types, next.Types)
		assert.Equal(t, first.Methods, next.Methods)
	}
}

// ---------------------------------------------------------------------------
// guards
// ---------------------------------------------------------------------------

func TestBuildRejectsUnusableInput(t *testing.T) {
	_, _, err := BuildFromURL(nil, reflect.TypeFor[*basicService]())
	require.Error(t, err)

	_, _, err = BuildFromURL(testURL(t, "org.example.Svc", "", ""), nil)
	require.Error(t, err)

	_, _, err = BuildFromURL(testURL(t, "", "", ""), reflect.TypeFor[*basicService]())
	require.Error(t, err, "an empty interface name cannot identify a definition")
}

// javaNamed declares a Java class name, as a hessian POJO does.
type javaNamed struct {
	Label string
}

func (javaNamed) JavaClassName() string { return "com.example.Named" }
