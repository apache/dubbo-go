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

package main

import (
	"go/ast"
	"go/token"
	"go/types"
	"testing"
)

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"golang.org/x/tools/go/packages"
)

func TestNormalizePatterns(t *testing.T) {
	assert.Equal(t, []string{"./..."}, normalizePatterns(nil))
	assert.Equal(t, []string{"./internal/..."}, normalizePatterns([]string{"./internal/..."}))
}

func TestSortFindings(t *testing.T) {
	t.Run("sorts by filename", func(t *testing.T) {
		findings := []Finding{
			{Position: token.Position{Filename: "b.go", Line: 1, Column: 1}},
			{Position: token.Position{Filename: "a.go", Line: 1, Column: 1}},
		}

		sortFindings(findings)
		assert.Equal(t, "a.go", findings[0].Position.Filename)
	})

	t.Run("sorts by line", func(t *testing.T) {
		findings := []Finding{
			{Position: token.Position{Filename: "a.go", Line: 2, Column: 1}},
			{Position: token.Position{Filename: "a.go", Line: 1, Column: 1}},
		}

		sortFindings(findings)
		assert.Equal(t, 1, findings[0].Position.Line)
	})

	t.Run("sorts by column kind type and method", func(t *testing.T) {
		findings := []Finding{
			{Position: token.Position{Filename: "a.go", Line: 1, Column: 2}, Kind: "implementation", TypeName: "B", MethodName: "B"},
			{Position: token.Position{Filename: "a.go", Line: 1, Column: 1}, Kind: "interface", TypeName: "A", MethodName: "A"},
		}

		sortFindings(findings)
		assert.Equal(t, 1, findings[0].Position.Column)
	})

	t.Run("sorts by kind", func(t *testing.T) {
		findings := []Finding{
			{Position: token.Position{Filename: "a.go", Line: 1, Column: 1}, Kind: "implementation", TypeName: "A", MethodName: "A"},
			{Position: token.Position{Filename: "a.go", Line: 1, Column: 1}, Kind: "interface", TypeName: "A", MethodName: "A"},
		}

		sortFindings(findings)
		assert.Equal(t, "implementation", findings[0].Kind)
	})

	t.Run("sorts by type and method name", func(t *testing.T) {
		findings := []Finding{
			{Position: token.Position{Filename: "a.go", Line: 1, Column: 1}, Kind: "interface", TypeName: "B", MethodName: "B"},
			{Position: token.Position{Filename: "a.go", Line: 1, Column: 1}, Kind: "interface", TypeName: "A", MethodName: "A"},
		}

		sortFindings(findings)
		assert.Equal(t, "A", findings[0].TypeName)

		findings = []Finding{
			{Position: token.Position{Filename: "a.go", Line: 1, Column: 1}, Kind: "interface", TypeName: "A", MethodName: "B"},
			{Position: token.Position{Filename: "a.go", Line: 1, Column: 1}, Kind: "interface", TypeName: "A", MethodName: "A"},
		}

		sortFindings(findings)
		assert.Equal(t, "A", findings[0].MethodName)
	})
}

func TestPackageAndSignatureHelpers(t *testing.T) {
	t.Run("packageErrors handles nil empty and populated packages", func(t *testing.T) {
		assert.Nil(t, packageErrors(nil))
		assert.Nil(t, packageErrors(&packages.Package{}))
		assert.Equal(t, []string{"-: load failed"}, packageErrors(&packages.Package{
			Errors: []packages.Error{{Msg: "load failed"}},
		}))
	})

	t.Run("typeQualifier returns package path", func(t *testing.T) {
		assert.Empty(t, typeQualifier(nil))
		assert.Equal(t, "example.com/test", typeQualifier(types.NewPackage("example.com/test", "test")))
	})

	t.Run("receiverTypeName unwraps supported receiver forms", func(t *testing.T) {
		assert.Equal(t, "Svc", receiverTypeName(&ast.Ident{Name: "Svc"}))
		assert.Equal(t, "Svc", receiverTypeName(&ast.StarExpr{X: &ast.Ident{Name: "Svc"}}))
		assert.Equal(t, "Svc", receiverTypeName(&ast.IndexExpr{X: &ast.Ident{Name: "Svc"}}))
		assert.Equal(t, "Svc", receiverTypeName(&ast.IndexListExpr{X: &ast.Ident{Name: "Svc"}}))
		assert.Empty(t, receiverTypeName(&ast.ArrayType{}))
	})
}

func TestTypeFilters(t *testing.T) {
	pkg := types.NewPackage("example.com/test", "test")
	exported := types.NewNamed(types.NewTypeName(token.NoPos, pkg, "Visible", nil), types.Typ[types.Int], nil)
	unexported := types.NewNamed(types.NewTypeName(token.NoPos, pkg, "hidden", nil), types.Typ[types.Int], nil)
	option := types.NewNamed(types.NewTypeName(token.NoPos, pkg, "ClientOption", nil), types.Typ[types.Int], nil)
	callOptionAlias := types.NewAlias(types.NewTypeName(token.NoPos, pkg, "CallOption", nil), types.Typ[types.Int])

	assert.True(t, isExportedOrBuiltinGoType(types.NewPointer(exported)))
	assert.False(t, isExportedOrBuiltinGoType(unexported))
	assert.True(t, isExportedOrBuiltinGoType(types.NewSlice(types.Typ[types.String])))
	assert.True(t, isExportedOrBuiltinGoType(callOptionAlias))

	assert.True(t, isExportedOrBuiltinTypeName(nil))
	assert.True(t, isExportedOrBuiltinTypeName(types.Universe.Lookup("string").(*types.TypeName)))
	assert.False(t, isExportedOrBuiltinTypeName(types.NewTypeName(token.NoPos, pkg, "hiddenType", nil)))

	assert.True(t, isOptionLikeVariadic(types.NewSlice(option)))
	assert.False(t, isOptionLikeVariadic(types.Typ[types.Int]))

	assert.True(t, isOptionLikeType(types.NewPointer(option)))
	assert.True(t, isOptionLikeType(callOptionAlias))
	assert.False(t, isOptionLikeType(types.NewSlice(types.Typ[types.String])))
}

func TestVariadicSignatureChecks(t *testing.T) {
	errorType := types.Universe.Lookup("error").Type()
	pkg := types.NewPackage("example.com/test", "test")
	exportedReply := types.NewNamed(types.NewTypeName(token.NoPos, pkg, "VisibleReply", nil), types.Typ[types.Int], nil)
	unexportedReply := types.NewNamed(types.NewTypeName(token.NoPos, pkg, "hiddenReply", nil), types.Typ[types.Int], nil)
	unexportedArg := types.NewNamed(types.NewTypeName(token.NoPos, pkg, "hiddenArg", nil), types.Typ[types.Int], nil)
	option := types.NewNamed(types.NewTypeName(token.NoPos, pkg, "ClientOption", nil), types.Typ[types.Int], nil)

	assert.False(t, isVariadicRPCSignature(nil))
	assert.False(t, isVariadicRPCSignature(testSignature(false, []types.Type{types.NewSlice(types.Typ[types.String])}, []types.Type{errorType})))
	assert.False(t, isVariadicRPCSignature(testSignature(true, []types.Type{types.NewSlice(types.Typ[types.String])}, nil)))
	assert.False(t, isVariadicRPCSignature(testSignature(true, []types.Type{types.NewSlice(types.Typ[types.String])}, []types.Type{types.Typ[types.Int]})))
	assert.False(t, isVariadicRPCSignature(testSignature(true, []types.Type{types.NewSlice(types.Typ[types.String])}, []types.Type{unexportedReply, errorType})))
	assert.False(t, isVariadicRPCSignature(testSignature(true, []types.Type{unexportedArg, types.NewSlice(types.Typ[types.String])}, []types.Type{errorType})))
	assert.False(t, isVariadicRPCSignature(testSignature(true, []types.Type{types.NewSlice(option)}, []types.Type{errorType})))
	assert.True(t, isVariadicRPCSignature(testSignature(true, []types.Type{types.NewSlice(types.Typ[types.String])}, []types.Type{exportedReply, errorType})))
}

func TestAstAndTypeHelpers(t *testing.T) {
	t.Run("collectPackageFindings skips syntax entries without compiled files", func(t *testing.T) {
		pkg := &packages.Package{
			Syntax: []*ast.File{{}},
		}
		assert.Empty(t, collectPackageFindings(pkg))
	})

	t.Run("embeddedInterfaceType handles nil missing and valid type info", func(t *testing.T) {
		assert.Nil(t, embeddedInterfaceType(nil, &ast.Ident{Name: "Embedded"}))

		expr := &ast.Ident{Name: "Missing"}
		pkg := &packages.Package{TypesInfo: &types.Info{Types: map[ast.Expr]types.TypeAndValue{}}}
		assert.Nil(t, embeddedInterfaceType(pkg, expr))

		iface := types.NewInterfaceType(nil, nil)
		iface.Complete()
		validExpr := &ast.Ident{Name: "Embedded"}
		pkg.TypesInfo.Types[validExpr] = types.TypeAndValue{Type: iface}
		assert.Same(t, iface, embeddedInterfaceType(pkg, validExpr))
	})

	t.Run("interfaceTypeForSpec and signatureForIdent return false without type info", func(t *testing.T) {
		pkg := &packages.Package{TypesInfo: &types.Info{Defs: map[*ast.Ident]types.Object{}}}
		typeSpec := &ast.TypeSpec{Name: &ast.Ident{Name: "Service"}}
		_, ok := interfaceTypeForSpec(pkg, typeSpec)
		assert.False(t, ok)

		_, ok = signatureForIdent(pkg, &ast.Ident{Name: "MultiArgs"})
		assert.False(t, ok)

		assert.Nil(t, typeSpecInterfaceFindings(pkg, typeSpec, &ast.InterfaceType{}))
	})

	t.Run("interfaceMethodFinding covers fallback and rejection cases", func(t *testing.T) {
		fset := token.NewFileSet()
		file := fset.AddFile("service.go", -1, 64)
		typePos := file.Pos(1)
		typeSpec := &ast.TypeSpec{Name: &ast.Ident{Name: "Service", NamePos: typePos}}
		pkg := &packages.Package{Fset: fset}

		finding, ok := interfaceMethodFinding(pkg, typeSpec, types.NewFunc(token.NoPos, nil, "hidden", testSignature(true, []types.Type{types.NewSlice(types.Typ[types.String])}, []types.Type{types.Universe.Lookup("error").Type()})), nil)
		assert.False(t, ok)
		assert.Equal(t, Finding{}, finding)

		finding, ok = interfaceMethodFinding(pkg, typeSpec, types.NewFunc(token.NoPos, nil, "MultiArgs", testSignature(true, []types.Type{types.NewSlice(types.Typ[types.String])}, []types.Type{types.Universe.Lookup("error").Type()})), map[string]token.Pos{})
		require.True(t, ok)
		assert.Equal(t, "service.go", finding.Position.Filename)
		assert.Equal(t, "Service", finding.TypeName)
		assert.Equal(t, "MultiArgs", finding.MethodName)

		finding, ok = interfaceMethodFinding(pkg, typeSpec, types.NewFunc(token.NoPos, nil, "Reference", testSignature(true, []types.Type{types.NewSlice(types.Typ[types.String])}, []types.Type{types.Universe.Lookup("error").Type()})), map[string]token.Pos{})
		assert.False(t, ok)
		assert.Equal(t, Finding{}, finding)
	})

	t.Run("collectImplementationFinding rejects invalid declarations", func(t *testing.T) {
		pkg := &packages.Package{TypesInfo: &types.Info{Defs: map[*ast.Ident]types.Object{}}}

		_, ok := collectImplementationFinding(pkg, &ast.FuncDecl{Name: &ast.Ident{Name: "MultiArgs"}})
		assert.False(t, ok)

		_, ok = collectImplementationFinding(pkg, &ast.FuncDecl{
			Name: &ast.Ident{Name: "multiArgs"},
			Recv: &ast.FieldList{List: []*ast.Field{{Type: &ast.Ident{Name: "Service"}}}},
		})
		assert.False(t, ok)

		_, ok = collectImplementationFinding(pkg, &ast.FuncDecl{
			Name: &ast.Ident{Name: "Reference"},
			Recv: &ast.FieldList{List: []*ast.Field{{Type: &ast.Ident{Name: "Service"}}}},
		})
		assert.False(t, ok)

		_, ok = collectImplementationFinding(pkg, &ast.FuncDecl{
			Name: &ast.Ident{Name: "MultiArgs"},
			Recv: &ast.FieldList{List: []*ast.Field{{Type: &ast.ArrayType{}}}},
		})
		assert.False(t, ok)
	})

	t.Run("interfaceMethodPositions handles empty and embedded interfaces", func(t *testing.T) {
		assert.Empty(t, interfaceMethodPositions(nil, nil))

		fset := token.NewFileSet()
		file := fset.AddFile("service.go", -1, 64)
		methodPos := file.Pos(10)
		embedPos := file.Pos(20)
		embeddedExpr := &ast.Ident{Name: "Embedded", NamePos: embedPos}
		iface := types.NewInterfaceType([]*types.Func{
			types.NewFunc(token.NoPos, nil, "Broadcast", testSignature(true, []types.Type{types.NewSlice(types.Typ[types.String])}, []types.Type{types.Universe.Lookup("error").Type()})),
		}, nil)
		iface.Complete()

		pkg := &packages.Package{
			TypesInfo: &types.Info{
				Types: map[ast.Expr]types.TypeAndValue{
					embeddedExpr: {Type: iface},
				},
			},
		}

		positions := interfaceMethodPositions(pkg, &ast.InterfaceType{
			Methods: &ast.FieldList{List: []*ast.Field{
				{Names: []*ast.Ident{{Name: "Direct", NamePos: methodPos}}},
				{Type: embeddedExpr},
			}},
		})
		assert.Equal(t, methodPos, positions["Direct"])
		assert.Equal(t, embedPos, positions["Broadcast"])
	})

	t.Run("recordEmbeddedMethodPositions ignores unresolved embedded interfaces", func(t *testing.T) {
		positions := make(map[string]token.Pos)
		recordEmbeddedMethodPositions(&packages.Package{
			TypesInfo: &types.Info{Types: map[ast.Expr]types.TypeAndValue{}},
		}, positions, &ast.Ident{Name: "Missing"})
		assert.Empty(t, positions)
	})
}

func TestScanReportsPackageLoadErrors(t *testing.T) {
	oldPackagesLoad := packagesLoad
	packagesLoad = func(cfg *packages.Config, patterns ...string) ([]*packages.Package, error) {
		return nil, assert.AnError
	}
	t.Cleanup(func() {
		packagesLoad = oldPackagesLoad
	})

	findings, err := Scan(".", []string{"./..."})
	assert.Nil(t, findings)
	require.ErrorIs(t, err, assert.AnError)
}

func testSignature(variadic bool, paramTypes []types.Type, resultTypes []types.Type) *types.Signature {
	params := make([]*types.Var, 0, len(paramTypes))
	for i, paramType := range paramTypes {
		params = append(params, types.NewVar(token.NoPos, nil, string(rune('a'+i)), paramType))
	}

	results := make([]*types.Var, 0, len(resultTypes))
	for i, resultType := range resultTypes {
		results = append(results, types.NewVar(token.NoPos, nil, string(rune('r'+i)), resultType))
	}

	return types.NewSignatureType(nil, nil, nil, types.NewTuple(params...), types.NewTuple(results...), variadic)
}
