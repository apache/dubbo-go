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
	assert.Equal(t, []string{"string"}, get.ParameterTypes,
		"receiver and leading context.Context must not appear as parameters")

	noCtx := methodByName(t, def, "NoContext")
	assert.Equal(t, []string{"string", "int64"}, noCtx.ParameterTypes,
		"a method without context.Context keeps all its declared parameters")
}

func TestBuildReturnTypes(t *testing.T) {
	def, _ := build(t, &basicService{})

	assert.Equal(t, "*dubbo.apache.org/dubbo-go/v3/metadata/definition.User",
		methodByName(t, def, "GetUser").ReturnType)
	assert.Equal(t, "bool", methodByName(t, def, "NoContext").ReturnType)

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
	assert.Equal(t, "string", noCtx.Parameters[0].Type)
	assert.Equal(t, "int64", noCtx.Parameters[1].Type)

	for i, p := range noCtx.Parameters {
		assert.Equal(t, noCtx.ParameterTypes[i], p.Type,
			"Parameters and ParameterTypes must agree positionally")
	}
}

// ---------------------------------------------------------------------------
// type expression and wrapper entries
// ---------------------------------------------------------------------------

func TestBuildEmitsWrapperTypeEntries(t *testing.T) {
	def, _ := build(t, &basicService{})

	const userType = "dubbo.apache.org/dubbo-go/v3/metadata/definition.User"
	const addrType = "dubbo.apache.org/dubbo-go/v3/metadata/definition.Address"

	// GetUser returns *User, so Admin looks up "*User" first and needs a path
	// from there down to User's fields.
	ptr := typeByName(t, def, "*"+userType)
	assert.Equal(t, []string{userType}, ptr.Items)

	user := typeByName(t, def, userType)
	assert.Equal(t, map[string]string{
		"name":     "string",
		"age":      "int32",
		"home":     "*" + addrType,
		"tags":     "[]string",
		"scores":   "map[string]float64",
		"quadrant": "[4]int",
	}, user.Properties)

	// Each composite property must itself be resolvable to its element.
	assert.Equal(t, []string{"string"}, typeByName(t, def, "[]string").Items)
	assert.Equal(t, []string{"float64"}, typeByName(t, def, "map[string]float64").Items)
	assert.Equal(t, []string{"int"}, typeByName(t, def, "[4]int").Items)
	assert.Equal(t, []string{addrType}, typeByName(t, def, "*"+addrType).Items)
}

func TestBuildStructPropertiesFollowGeneralizerNaming(t *testing.T) {
	def, _ := build(t, &basicService{})

	addr := typeByName(t, def, "dubbo.apache.org/dubbo-go/v3/metadata/definition.Address")
	assert.Equal(t, map[string]string{
		"city":    "string", // no tag: first rune lowercased
		"zipCode": "string", // m tag wins verbatim
	}, addr.Properties)
	assert.NotContains(t, addr.Properties, "hidden", "unexported fields are not on the wire")
	assert.NotContains(t, addr.Properties, "Zip", "the Go name is not the wire name when a tag exists")
}

func TestBuildRecursiveTypeTerminates(t *testing.T) {
	def, _ := build(t, &basicService{})

	const nodeType = "dubbo.apache.org/dubbo-go/v3/metadata/definition.Node"
	node := typeByName(t, def, nodeType)
	assert.Equal(t, map[string]string{
		"label": "string",
		"next":  "*" + nodeType,
	}, node.Properties, "the cycle is preserved as a reference, not expanded")
	assert.Equal(t, []string{nodeType}, typeByName(t, def, "*"+nodeType).Items)
}

func TestBuildNamedScalarUsesUnderlyingType(t *testing.T) {
	// A named scalar travels the generic wire as its underlying builtin, so the
	// contract names the builtin.
	c := newTypeCollector()
	expr, err := c.resolve(reflect.TypeOf(time.Month(1)))
	require.NoError(t, err)
	assert.Equal(t, "int", expr)
}

// ---------------------------------------------------------------------------
// canonical names and conflicts
// ---------------------------------------------------------------------------

func TestBuildCanonicalNameComesFromURL(t *testing.T) {
	u := testURL(t, "org.example.UserService", "1.0.0", "g1")
	def, _, err := BuildFromURL(u, reflect.TypeOf(&basicService{}))
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
	methods := common.CanonicalMethods(reflect.TypeOf(&collidingService{}))
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
		{`m:"-"`, reflect.TypeOf(dashTagged{}), `skipped by Realize`},
		{"tag option", reflect.TypeOf(optionTagged{}), "interpreted differently"},
		{"non-ASCII field", reflect.TypeOf(nonASCIINamed{}), "non-ASCII"},
		{"canonical/legacy collision", reflect.TypeOf(aliasColliding{}), "both reachable as"},
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
	_, err := newTypeCollector().resolve(reflect.TypeOf(make(chan int)))
	require.Error(t, err)
	assert.True(t, IsUnsupported(err))
}

// ---------------------------------------------------------------------------
// determinism
// ---------------------------------------------------------------------------

func TestBuildIsDeterministic(t *testing.T) {
	// Republishing identical content on every restart is what keeps the
	// metadata center from churning, so map iteration must not leak into output.
	first, _, err := BuildFromURL(testURL(t, "org.example.Svc", "", ""), reflect.TypeOf(&basicService{}))
	require.NoError(t, err)

	for range 20 {
		next, _, err := BuildFromURL(testURL(t, "org.example.Svc", "", ""), reflect.TypeOf(&basicService{}))
		require.NoError(t, err)
		assert.Equal(t, first.Types, next.Types)
		assert.Equal(t, first.Methods, next.Methods)
	}
}

// ---------------------------------------------------------------------------
// guards
// ---------------------------------------------------------------------------

func TestBuildRejectsUnusableInput(t *testing.T) {
	_, _, err := BuildFromURL(nil, reflect.TypeOf(&basicService{}))
	assert.Error(t, err)

	_, _, err = BuildFromURL(testURL(t, "org.example.Svc", "", ""), nil)
	assert.Error(t, err)

	_, _, err = BuildFromURL(testURL(t, "", "", ""), reflect.TypeOf(&basicService{}))
	assert.Error(t, err, "an empty interface name cannot identify a definition")
}
