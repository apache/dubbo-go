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
	"errors"
	"fmt"
	"go/ast"
	"go/token"
	"go/types"
	"path/filepath"
	"sort"
	"strings"
)

import (
	"golang.org/x/tools/go/packages"
)

type Finding struct {
	Position   token.Position
	Kind       string
	TypeName   string
	MethodName string
}

// String renders warning-only output so callers can pipe the tool into CI logs
// without any post-processing.
func (f Finding) String() string {
	return fmt.Sprintf(
		"%s:%d:%d: warning: %s %s exports variadic RPC method %s; prefer []T, request structs, or Triple + Protobuf IDL",
		f.Position.Filename,
		f.Position.Line,
		f.Position.Column,
		f.Kind,
		f.TypeName,
		f.MethodName,
	)
}

// Scan loads the requested packages, finds exported variadic RPC contracts,
// keeps the output order stable, and returns any package-load errors together
// with the findings collected before the error was seen.
func Scan(dir string, patterns []string) ([]Finding, error) {
	pkgs, err := loadPackages(dir, normalizePatterns(patterns))
	if err != nil {
		return nil, err
	}

	findings, loadErrs := collectFindingsAndErrors(pkgs)
	sortFindings(findings)
	if len(loadErrs) > 0 {
		return findings, errors.New(strings.Join(loadErrs, "; "))
	}

	return findings, nil
}

// normalizePatterns falls back to the usual recursive package scan when the
// caller does not pass an explicit pattern list.
func normalizePatterns(patterns []string) []string {
	if len(patterns) == 0 {
		return []string{"./..."}
	}
	return patterns
}

// loadPackages asks go/packages for the syntax and type information needed to
// inspect both interface declarations and concrete methods.
func loadPackages(dir string, patterns []string) ([]*packages.Package, error) {
	cfg := &packages.Config{
		Mode: packages.NeedName |
			packages.NeedFiles |
			packages.NeedCompiledGoFiles |
			packages.NeedSyntax |
			packages.NeedTypes |
			packages.NeedTypesInfo,
		Dir: dir,
	}
	return packages.Load(cfg, patterns...)
}

// collectFindingsAndErrors scans every loaded package and keeps loader errors
// separate so we can still emit useful warnings before returning the error.
func collectFindingsAndErrors(pkgs []*packages.Package) ([]Finding, []string) {
	findings := make([]Finding, 0)
	loadErrs := make([]string, 0)
	for _, pkg := range pkgs {
		loadErrs = append(loadErrs, packageErrors(pkg)...)
		findings = append(findings, collectPackageFindings(pkg)...)
	}
	return findings, loadErrs
}

// packageErrors flattens go/packages errors into strings so Scan can combine
// them into one returned error value.
func packageErrors(pkg *packages.Package) []string {
	if pkg == nil || len(pkg.Errors) == 0 {
		return nil
	}

	loadErrs := make([]string, 0, len(pkg.Errors))
	for _, pkgErr := range pkg.Errors {
		loadErrs = append(loadErrs, pkgErr.Error())
	}
	return loadErrs
}

// sortFindings keeps warning output stable across runs.
func sortFindings(findings []Finding) {
	sort.Slice(findings, func(i, j int) bool {
		if findings[i].Position.Filename != findings[j].Position.Filename {
			return findings[i].Position.Filename < findings[j].Position.Filename
		}
		if findings[i].Position.Line != findings[j].Position.Line {
			return findings[i].Position.Line < findings[j].Position.Line
		}
		if findings[i].Position.Column != findings[j].Position.Column {
			return findings[i].Position.Column < findings[j].Position.Column
		}
		if findings[i].Kind != findings[j].Kind {
			return findings[i].Kind < findings[j].Kind
		}
		if findings[i].TypeName != findings[j].TypeName {
			return findings[i].TypeName < findings[j].TypeName
		}
		return findings[i].MethodName < findings[j].MethodName
	})
}

// collectPackageFindings walks compiled files so build tags and generated-file
// selection stay aligned with the package loader.
func collectPackageFindings(pkg *packages.Package) []Finding {
	findings := make([]Finding, 0)
	for fileIdx, file := range pkg.Syntax {
		if fileIdx >= len(pkg.CompiledGoFiles) {
			continue
		}

		filename := pkg.CompiledGoFiles[fileIdx]
		if shouldSkipFile(filename) {
			continue
		}

		for _, decl := range file.Decls {
			switch typedDecl := decl.(type) {
			case *ast.GenDecl:
				findings = append(findings, collectInterfaceFindings(pkg, typedDecl)...)
			case *ast.FuncDecl:
				if finding, ok := collectImplementationFinding(pkg, typedDecl); ok {
					findings = append(findings, finding)
				}
			}
		}
	}
	return findings
}

// collectInterfaceFindings catches contract definitions before any concrete
// implementation exists, including methods inherited through embedded
// interfaces, which is where new cross-language APIs are often introduced.
func collectInterfaceFindings(pkg *packages.Package, decl *ast.GenDecl) []Finding {
	if decl.Tok != token.TYPE {
		return nil
	}

	findings := make([]Finding, 0)
	for _, spec := range decl.Specs {
		typeSpec, iface, ok := exportedInterfaceSpec(spec)
		if !ok {
			continue
		}

		findings = append(findings, typeSpecInterfaceFindings(pkg, typeSpec, iface)...)
	}

	return findings
}

// exportedInterfaceSpec keeps only exported interface declarations and skips
// unexported or non-interface type specs early.
func exportedInterfaceSpec(spec ast.Spec) (*ast.TypeSpec, *ast.InterfaceType, bool) {
	typeSpec, ok := spec.(*ast.TypeSpec)
	if !ok || !typeSpec.Name.IsExported() {
		return nil, nil, false
	}

	iface, ok := typeSpec.Type.(*ast.InterfaceType)
	if !ok {
		return nil, nil, false
	}

	return typeSpec, iface, true
}

// typeSpecInterfaceFindings runs the RPC-style variadic check for one exported
// interface and preserves positions for both direct and embedded methods.
func typeSpecInterfaceFindings(pkg *packages.Package, typeSpec *ast.TypeSpec, iface *ast.InterfaceType) []Finding {
	ifaceType, ok := interfaceTypeForSpec(pkg, typeSpec)
	if !ok {
		return nil
	}

	methodPositions := interfaceMethodPositions(pkg, iface)
	ifaceType.Complete()

	findings := make([]Finding, 0, ifaceType.NumMethods())
	for i := 0; i < ifaceType.NumMethods(); i++ {
		if finding, ok := interfaceMethodFinding(pkg, typeSpec, ifaceType.Method(i), methodPositions); ok {
			findings = append(findings, finding)
		}
	}
	return findings
}

// interfaceTypeForSpec resolves the go/types interface backing one AST type
// declaration so later checks can use normalized method signatures.
func interfaceTypeForSpec(pkg *packages.Package, typeSpec *ast.TypeSpec) (*types.Interface, bool) {
	obj := pkg.TypesInfo.Defs[typeSpec.Name]
	if obj == nil {
		return nil, false
	}

	ifaceType, ok := obj.Type().Underlying().(*types.Interface)
	return ifaceType, ok
}

// interfaceMethodFinding returns one finding for an exported variadic RPC-style
// interface method and falls back to the interface declaration position when a
// more precise method position is unavailable.
func interfaceMethodFinding(pkg *packages.Package, typeSpec *ast.TypeSpec, method *types.Func, methodPositions map[string]token.Pos) (Finding, bool) {
	if !method.Exported() || !isCandidateRPCMethodName(method.Name()) {
		return Finding{}, false
	}

	sig, ok := method.Type().(*types.Signature)
	if !ok || !isVariadicRPCSignature(sig) {
		return Finding{}, false
	}

	pos := methodPositions[method.Name()]
	if !pos.IsValid() {
		pos = typeSpec.Name.Pos()
	}

	return Finding{
		Position:   pkg.Fset.Position(pos),
		Kind:       "interface",
		TypeName:   typeSpec.Name.Name,
		MethodName: method.Name(),
	}, true
}

// collectImplementationFinding covers struct receiver methods used as direct
// service implementations in non-interface registration flows.
func collectImplementationFinding(pkg *packages.Package, decl *ast.FuncDecl) (Finding, bool) {
	if decl.Recv == nil || len(decl.Recv.List) == 0 || !decl.Name.IsExported() {
		return Finding{}, false
	}
	if !isCandidateRPCMethodName(decl.Name.Name) {
		return Finding{}, false
	}

	receiverName := receiverTypeName(decl.Recv.List[0].Type)
	if receiverName == "" || !ast.IsExported(receiverName) {
		return Finding{}, false
	}

	sig, ok := signatureForIdent(pkg, decl.Name)
	if !ok || !isVariadicRPCSignature(sig) {
		return Finding{}, false
	}

	return Finding{
		Position:   pkg.Fset.Position(decl.Name.Pos()),
		Kind:       "implementation",
		TypeName:   receiverName,
		MethodName: decl.Name.Name,
	}, true
}

func signatureForIdent(pkg *packages.Package, ident *ast.Ident) (*types.Signature, bool) {
	obj := pkg.TypesInfo.Defs[ident]
	if obj == nil {
		return nil, false
	}

	sig, ok := obj.Type().(*types.Signature)
	return sig, ok
}

// interfaceMethodPositions maps exported method names to the source line we
// want to report. Direct methods use their own line; embedded methods use the
// local embedding line.
func interfaceMethodPositions(pkg *packages.Package, iface *ast.InterfaceType) map[string]token.Pos {
	positions := make(map[string]token.Pos)
	if iface == nil || iface.Methods == nil {
		return positions
	}

	for _, field := range iface.Methods.List {
		recordInterfaceMethodPositions(pkg, positions, field)
	}

	return positions
}

// recordInterfaceMethodPositions handles one interface field entry, whether it
// is a named method or an embedded interface.
func recordInterfaceMethodPositions(pkg *packages.Package, positions map[string]token.Pos, field *ast.Field) {
	if len(field.Names) == 1 {
		recordNamedMethodPosition(positions, field.Names[0])
		return
	}

	recordEmbeddedMethodPositions(pkg, positions, field.Type)
}

// recordNamedMethodPosition stores the declared position for one exported
// method listed directly in the interface body.
func recordNamedMethodPosition(positions map[string]token.Pos, methodName *ast.Ident) {
	if methodName != nil && methodName.IsExported() {
		positions[methodName.Name] = methodName.Pos()
	}
}

// recordEmbeddedMethodPositions records exported methods contributed by an
// embedded interface, but reports them at the local embedding site so the
// warning points to the contract introduced in this file.
func recordEmbeddedMethodPositions(pkg *packages.Package, positions map[string]token.Pos, expr ast.Expr) {
	embeddedIface := embeddedInterfaceType(pkg, expr)
	if embeddedIface == nil {
		return
	}

	embeddedIface.Complete()
	for i := 0; i < embeddedIface.NumMethods(); i++ {
		method := embeddedIface.Method(i)
		if method.Exported() {
			recordEmbeddedMethodPosition(positions, method.Name(), expr.Pos())
		}
	}
}

// recordEmbeddedMethodPosition keeps the first local embedding position seen
// for a method name when multiple embedded interfaces contribute it.
func recordEmbeddedMethodPosition(positions map[string]token.Pos, methodName string, pos token.Pos) {
	if _, exists := positions[methodName]; !exists {
		positions[methodName] = pos
	}
}

func embeddedInterfaceType(pkg *packages.Package, expr ast.Expr) *types.Interface {
	if pkg == nil || pkg.TypesInfo == nil {
		return nil
	}

	typedExpr, ok := pkg.TypesInfo.Types[expr]
	if !ok || typedExpr.Type == nil {
		return nil
	}

	iface, _ := typedExpr.Type.Underlying().(*types.Interface)
	return iface
}

func receiverTypeName(expr ast.Expr) string {
	switch typedExpr := expr.(type) {
	case *ast.Ident:
		return typedExpr.Name
	case *ast.StarExpr:
		return receiverTypeName(typedExpr.X)
	case *ast.IndexExpr:
		return receiverTypeName(typedExpr.X)
	case *ast.IndexListExpr:
		return receiverTypeName(typedExpr.X)
	default:
		return ""
	}
}

// shouldSkipFile filters generated and test sources so the tool focuses on
// user-authored service contracts instead of stubs and test scaffolding.
func shouldSkipFile(filename string) bool {
	base := filepath.Base(filename)
	return strings.HasSuffix(base, "_test.go") ||
		strings.HasSuffix(base, ".pb.go") ||
		strings.HasSuffix(base, ".triple.go")
}

func isCandidateRPCMethodName(methodName string) bool {
	return methodName != "Reference" &&
		methodName != "SetGRPCServer" &&
		!strings.HasPrefix(methodName, "XXX")
}

// isVariadicRPCSignature mirrors the service-method shape closely enough to be
// useful in CI, while excluding option-style variadics such as opts ...Option.
func isVariadicRPCSignature(sig *types.Signature) bool {
	if sig == nil || !sig.Variadic() {
		return false
	}

	results := sig.Results()
	if results == nil || (results.Len() != 1 && results.Len() != 2) {
		return false
	}
	if types.TypeString(results.At(results.Len()-1).Type(), nil) != "error" {
		return false
	}
	if results.Len() == 2 && !isExportedOrBuiltinGoType(results.At(0).Type()) {
		return false
	}

	params := sig.Params()
	if params.Len() == 0 {
		return false
	}
	for i := 0; i < params.Len(); i++ {
		if !isExportedOrBuiltinGoType(params.At(i).Type()) {
			return false
		}
	}

	return !isOptionLikeVariadic(params.At(params.Len() - 1).Type())
}

// isExportedOrBuiltinGoType mirrors common.isExportedOrBuiltinType closely
// enough for scanner parity: pointer wrappers are unwrapped, named types must
// be exported unless they are builtins, and unnamed composite types are
// accepted like reflect.Type.Name/PkgPath would be.
func isExportedOrBuiltinGoType(typ types.Type) bool {
	for {
		ptr, ok := typ.(*types.Pointer)
		if !ok {
			break
		}
		typ = ptr.Elem()
	}

	switch typedType := typ.(type) {
	case *types.Named:
		return isExportedOrBuiltinTypeName(typedType.Obj())
	case *types.Alias:
		return isExportedOrBuiltinTypeName(typedType.Obj())
	default:
		return true
	}
}

func isExportedOrBuiltinTypeName(obj *types.TypeName) bool {
	if obj == nil {
		return true
	}
	return token.IsExported(obj.Name()) || obj.Pkg() == nil
}

func isOptionLikeVariadic(paramType types.Type) bool {
	sliceType, ok := paramType.(*types.Slice)
	if !ok {
		return false
	}

	return isOptionLikeType(sliceType.Elem())
}

func isOptionLikeType(typ types.Type) bool {
	switch typedType := typ.(type) {
	case *types.Pointer:
		return isOptionLikeType(typedType.Elem())
	case *types.Named:
		return hasOptionLikeName(typedType.Obj().Name()) || hasOptionLikeSuffix(types.TypeString(typedType, typeQualifier))
	case *types.Alias:
		return hasOptionLikeName(typedType.Obj().Name()) || hasOptionLikeSuffix(types.TypeString(typedType, typeQualifier))
	default:
		return hasOptionLikeSuffix(types.TypeString(typ, typeQualifier))
	}
}

func hasOptionLikeName(name string) bool {
	return strings.HasSuffix(name, "Option") || strings.HasSuffix(name, "CallOption")
}

func hasOptionLikeSuffix(typeName string) bool {
	return strings.HasSuffix(typeName, ".Option") ||
		strings.HasSuffix(typeName, ".CallOption") ||
		strings.HasSuffix(typeName, " Option") ||
		strings.HasSuffix(typeName, " CallOption") ||
		hasOptionLikeName(typeName)
}

func typeQualifier(pkg *types.Package) string {
	if pkg == nil {
		return ""
	}
	return pkg.Path()
}
