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

var packagesLoad = packages.Load

const (
	dubboRootPkgPath   = "dubbo.apache.org/dubbo-go/v3"
	dubboCommonPkgPath = dubboRootPkgPath + "/common"
	dubboConfigPkgPath = dubboRootPkgPath + "/config"
	dubboServerPkgPath = dubboRootPkgPath + "/server"

	serverServiceOptionsType = "*" + dubboServerPkgPath + ".ServiceOptions"
	configServiceConfigType  = "*" + dubboConfigPkgPath + ".ServiceConfig"
)

type registeredTypeKey struct {
	pkgPath  string
	typeName string
}

type functionKey struct {
	pkgPath string
	name    string
}

// functionDeclRef keeps the owning package with the helper body so wrapper
// tracing can inspect forwarded parameters.
type functionDeclRef struct {
	pkg  *packages.Package
	decl *ast.FuncDecl
}

// assignmentRef records one initializer or reassignment with its source
// position for call-site-sensitive lookup.
type assignmentRef struct {
	pos  token.Pos
	expr ast.Expr
}

type wrapperResolution struct {
	argIndex  int
	resolving bool
	resolved  bool
}

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
	return packagesLoad(cfg, patterns...)
}

// collectFindingsAndErrors scans every loaded package and keeps loader errors
// separate so we can still emit useful warnings before returning the error.
func collectFindingsAndErrors(pkgs []*packages.Package) ([]Finding, []string) {
	findings := make([]Finding, 0)
	loadErrs := make([]string, 0)
	registeredTypes := registeredImplementationTypes(pkgs)
	for _, pkg := range pkgs {
		loadErrs = append(loadErrs, packageErrors(pkg)...)
		findings = append(findings, collectPackageFindings(pkg, registeredTypes)...)
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
func collectPackageFindings(pkg *packages.Package, registeredTypes map[registeredTypeKey]struct{}) []Finding {
	findings := make([]Finding, 0)
	for fileIdx, file := range pkg.Syntax {
		if fileIdx >= len(pkg.CompiledGoFiles) {
			continue
		}

		filename := pkg.CompiledGoFiles[fileIdx]
		if shouldSkipFindingFile(filename) {
			continue
		}

		for _, decl := range file.Decls {
			switch typedDecl := decl.(type) {
			case *ast.GenDecl:
				findings = append(findings, collectInterfaceFindings(pkg, typedDecl)...)
			case *ast.FuncDecl:
				if finding, ok := collectImplementationFinding(pkg, typedDecl, registeredTypes); ok {
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
func collectImplementationFinding(pkg *packages.Package, decl *ast.FuncDecl, registeredTypes map[registeredTypeKey]struct{}) (Finding, bool) {
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
	if _, ok := registeredTypes[registeredTypeKey{pkgPath: packagePath(pkg), typeName: receiverName}]; !ok {
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

// registeredImplementationTypes collects exported concrete types that appear as
// handlers in known registration or export calls across the loaded packages.
func registeredImplementationTypes(pkgs []*packages.Package) map[registeredTypeKey]struct{} {
	registeredTypes := make(map[registeredTypeKey]struct{})
	analyzer := newRegistrationAnalyzer(pkgs)
	for _, pkg := range pkgs {
		analyzer.collectPackageRegistrations(pkg, registeredTypes)
	}
	return registeredTypes
}

// registrationAnalyzer holds the shared indexes used to trace concrete
// implementation types through registration helpers and wrappers.
type registrationAnalyzer struct {
	functionDecls      map[functionKey]functionDeclRef
	wrapperResolutions map[functionKey]*wrapperResolution
	varInitializers    map[string]map[types.Object][]assignmentRef
}

// newRegistrationAnalyzer precomputes the indexes reused across registration
// tracing.
func newRegistrationAnalyzer(pkgs []*packages.Package) *registrationAnalyzer {
	return &registrationAnalyzer{
		functionDecls:      functionDecls(pkgs),
		wrapperResolutions: make(map[functionKey]*wrapperResolution),
		varInitializers:    variableInitializers(pkgs),
	}
}

// collectPackageRegistrations walks one package and records concrete handler
// types used in known registration calls.
func (a *registrationAnalyzer) collectPackageRegistrations(pkg *packages.Package, registeredTypes map[registeredTypeKey]struct{}) {
	for fileIdx, file := range pkg.Syntax {
		if shouldSkipPackageFile(pkg, fileIdx, shouldSkipRegistrationFile) {
			continue
		}
		a.collectFileRegistrations(pkg, file, registeredTypes)
	}
}

// collectFileRegistrations scans a single file for registration calls.
func (a *registrationAnalyzer) collectFileRegistrations(pkg *packages.Package, file *ast.File, registeredTypes map[registeredTypeKey]struct{}) {
	ast.Inspect(file, func(node ast.Node) bool {
		call, ok := node.(*ast.CallExpr)
		if !ok {
			return true
		}

		handlerArg, ok := a.registrationHandlerArg(pkg, call)
		if !ok {
			return true
		}

		a.recordRegisteredImplementationType(pkg, registeredTypes, handlerArg, call.Pos())
		return true
	})
}

// registrationHandlerArg returns the AST expression that carries the service
// implementation for a recognized registration call.
func (a *registrationAnalyzer) registrationHandlerArg(pkg *packages.Package, call *ast.CallExpr) (ast.Expr, bool) {
	argIndex, ok := a.handlerArgumentIndex(pkg, call)
	if !ok || argIndex >= len(call.Args) {
		return nil, false
	}
	return call.Args[argIndex], true
}

// handlerArgumentIndex returns the argument slot that carries the service
// implementation for one known registration call shape.
func (a *registrationAnalyzer) handlerArgumentIndex(pkg *packages.Package, call *ast.CallExpr) (int, bool) {
	switch typedFun := call.Fun.(type) {
	case *ast.SelectorExpr:
		if selection := pkg.TypesInfo.Selections[typedFun]; selection != nil {
			return selectedMethodHandlerArgumentIndex(selection)
		}

		obj := pkg.TypesInfo.Uses[typedFun.Sel]
		return a.calledObjectHandlerArgumentIndex(obj)
	case *ast.Ident:
		obj := pkg.TypesInfo.Uses[typedFun]
		return a.calledObjectHandlerArgumentIndex(obj)
	default:
		return 0, false
	}
}

// selectedMethodHandlerArgumentIndex matches method-style registration paths
// such as srv.RegisterService(...) and cfg.Implement(...).
func selectedMethodHandlerArgumentIndex(selection *types.Selection) (int, bool) {
	obj := selection.Obj()
	if obj == nil || obj.Pkg() == nil {
		return 0, false
	}

	path := obj.Pkg().Path()
	name := obj.Name()
	switch {
	case path == dubboServerPkgPath && name == "Register":
		return 0, true
	case path == dubboServerPkgPath && name == "RegisterService":
		return 0, true
	case path == dubboServerPkgPath && name == "Implement":
		return 0, types.TypeString(selection.Recv(), nil) == serverServiceOptionsType
	case path == dubboConfigPkgPath && name == "Implement":
		return 0, types.TypeString(selection.Recv(), nil) == configServiceConfigType
	case path == dubboCommonPkgPath && name == "Register":
		return 4, strings.HasSuffix(types.TypeString(selection.Recv(), nil), ".serviceMap")
	default:
		return 0, false
	}
}

// calledObjectHandlerArgumentIndex matches package-level registration helpers
// and generated Register*Handler entry points.
func (a *registrationAnalyzer) calledObjectHandlerArgumentIndex(obj types.Object) (int, bool) {
	if obj == nil {
		return 0, false
	}
	if obj.Pkg() != nil {
		switch obj.Pkg().Path() {
		case dubboConfigPkgPath, dubboRootPkgPath:
			switch obj.Name() {
			case "SetProviderService":
				return 0, true
			case "SetProviderServiceWithInfo":
				return 0, true
			}
		}
	}

	if strings.HasPrefix(obj.Name(), "Register") && strings.HasSuffix(obj.Name(), "Handler") {
		return 1, true
	}

	if key, ok := functionKeyForObject(obj); ok {
		if argIndex, ok := a.wrapperHandlerArgumentIndex(key); ok {
			return argIndex, true
		}
	}

	return 0, false
}

// recordRegisteredImplementationType stores the named concrete type behind one
// registration argument when it resolves to an exported implementation.
func (a *registrationAnalyzer) recordRegisteredImplementationType(pkg *packages.Package, registeredTypes map[registeredTypeKey]struct{}, expr ast.Expr, callPos token.Pos) {
	if pkg == nil || pkg.TypesInfo == nil {
		return
	}

	if key, ok := a.registeredTypeKeyForExpr(pkg, expr, callPos, nil); ok {
		registeredTypes[key] = struct{}{}
	}
}

// registeredTypeKeyForExpr resolves one registration argument to the concrete
// exported type visible at that call site.
func (a *registrationAnalyzer) registeredTypeKeyForExpr(pkg *packages.Package, expr ast.Expr, callPos token.Pos, seen map[types.Object]struct{}) (registeredTypeKey, bool) {
	if key, ok := registeredTypeKeyForType(pkg.TypesInfo.TypeOf(expr)); ok {
		return key, true
	}

	ident, ok := expr.(*ast.Ident)
	if !ok {
		return registeredTypeKey{}, false
	}

	obj := pkg.TypesInfo.ObjectOf(ident)
	if obj == nil {
		return registeredTypeKey{}, false
	}

	if seen == nil {
		seen = make(map[types.Object]struct{})
	}
	if _, seenBefore := seen[obj]; seenBefore {
		return registeredTypeKey{}, false
	}
	seen[obj] = struct{}{}

	initExprs := a.varInitializers[packagePath(pkg)]
	if initExprs == nil {
		return registeredTypeKey{}, false
	}

	assignment, ok := latestAssignmentBefore(initExprs[obj], callPos)
	if !ok {
		return registeredTypeKey{}, false
	}

	return a.registeredTypeKeyForExpr(pkg, assignment.expr, assignment.pos, seen)
}

// wrapperHandlerArgumentIndex recognizes simple passthrough wrappers that
// forward one parameter into a known registration helper.
func (a *registrationAnalyzer) wrapperHandlerArgumentIndex(key functionKey) (int, bool) {
	state, ok, done := a.cachedWrapperResolution(key)
	if done {
		return state.argIndex, ok
	}

	ref, ok := a.functionDecls[key]
	if !ok || ref.decl == nil || ref.decl.Body == nil {
		return 0, false
	}

	params := parameterIndexes(ref.pkg, ref.decl)
	if len(params) == 0 {
		return 0, false
	}

	state.resolving = true
	defer func() {
		state.resolving = false
	}()

	return a.resolveWrapperHandlerArgument(ref, params, state)
}

// cachedWrapperResolution avoids re-walking wrapper bodies once a helper has
// been resolved or rejected.
func (a *registrationAnalyzer) cachedWrapperResolution(key functionKey) (*wrapperResolution, bool, bool) {
	state := a.wrapperResolutions[key]
	if state == nil {
		state = &wrapperResolution{}
		a.wrapperResolutions[key] = state
		return state, false, false
	}
	if state.resolved {
		return state, true, true
	}
	if state.resolving {
		return state, false, true
	}
	return state, false, false
}

// resolveWrapperHandlerArgument looks for the forwarded parameter that reaches
// a known registration helper inside one wrapper body.
func (a *registrationAnalyzer) resolveWrapperHandlerArgument(ref functionDeclRef, params map[types.Object]int, state *wrapperResolution) (int, bool) {
	for _, stmt := range ref.decl.Body.List {
		if paramIndex, ok := a.forwardedWrapperParamIndex(ref.pkg, params, stmt); ok {
			state.argIndex = paramIndex
			state.resolved = true
			return paramIndex, true
		}
	}
	return 0, false
}

// forwardedWrapperParamIndex searches one statement subtree for a forwarded
// registration parameter.
func (a *registrationAnalyzer) forwardedWrapperParamIndex(pkg *packages.Package, params map[types.Object]int, stmt ast.Node) (int, bool) {
	paramIndex := 0
	matched := false
	ast.Inspect(stmt, func(node ast.Node) bool {
		call, ok := node.(*ast.CallExpr)
		if !ok || matched {
			return !matched
		}

		idx, ok := a.wrapperParamIndexFromCall(pkg, params, call)
		if !ok {
			return true
		}

		paramIndex = idx
		matched = true
		return false
	})
	return paramIndex, matched
}

// wrapperParamIndexFromCall maps one nested registration call back to the
// wrapper parameter it forwards.
func (a *registrationAnalyzer) wrapperParamIndexFromCall(pkg *packages.Package, params map[types.Object]int, call *ast.CallExpr) (int, bool) {
	handlerArg, ok := a.registrationHandlerArg(pkg, call)
	if !ok {
		return 0, false
	}

	ident, ok := handlerArg.(*ast.Ident)
	if !ok {
		return 0, false
	}

	obj := pkg.TypesInfo.ObjectOf(ident)
	paramIndex, ok := params[obj]
	if !ok {
		return 0, false
	}

	return paramIndex, true
}

// functionDecls indexes package-level helpers that may wrap a known
// registration path.
func functionDecls(pkgs []*packages.Package) map[functionKey]functionDeclRef {
	decls := make(map[functionKey]functionDeclRef)
	for _, pkg := range pkgs {
		collectPackageFunctionDecls(pkg, decls)
	}
	return decls
}

// collectPackageFunctionDecls indexes package-level helpers from one package.
func collectPackageFunctionDecls(pkg *packages.Package, decls map[functionKey]functionDeclRef) {
	for fileIdx, file := range pkg.Syntax {
		if shouldSkipPackageFile(pkg, fileIdx, shouldSkipRegistrationFile) {
			continue
		}
		collectFileFunctionDecls(pkg, file, decls)
	}
}

// collectFileFunctionDecls records package-level helper declarations from one file.
func collectFileFunctionDecls(pkg *packages.Package, file *ast.File, decls map[functionKey]functionDeclRef) {
	for _, decl := range file.Decls {
		recordFunctionDecl(pkg, decls, decl)
	}
}

// recordFunctionDecl stores one package-level function under the stable key
// used by wrapper tracing.
func recordFunctionDecl(pkg *packages.Package, decls map[functionKey]functionDeclRef, decl ast.Decl) {
	funcDecl, ok := decl.(*ast.FuncDecl)
	if !ok || funcDecl.Recv != nil {
		return
	}

	obj := pkg.TypesInfo.Defs[funcDecl.Name]
	key, ok := functionKeyForObject(obj)
	if !ok {
		return
	}

	decls[key] = functionDeclRef{pkg: pkg, decl: funcDecl}
}

// variableInitializers stores ordered initializers and reassignments for
// interface-typed handler variables.
func variableInitializers(pkgs []*packages.Package) map[string]map[types.Object][]assignmentRef {
	initializers := make(map[string]map[types.Object][]assignmentRef)
	for _, pkg := range pkgs {
		for fileIdx, file := range pkg.Syntax {
			if shouldSkipPackageFile(pkg, fileIdx, shouldSkipRegistrationFile) {
				continue
			}

			pkgInits := initializers[packagePath(pkg)]
			if pkgInits == nil {
				pkgInits = make(map[types.Object][]assignmentRef)
				initializers[packagePath(pkg)] = pkgInits
			}

			recordValueSpecInitializers(pkg, pkgInits, file)
			recordAssignInitializers(pkg, pkgInits, file)
		}
	}
	return initializers
}

// recordValueSpecInitializers captures var declarations with explicit values
// for later call-site lookup.
func recordValueSpecInitializers(pkg *packages.Package, initializers map[types.Object][]assignmentRef, file *ast.File) {
	ast.Inspect(file, func(node ast.Node) bool {
		genDecl, ok := varGenDecl(node)
		if !ok {
			return true
		}
		recordVarDeclInitializers(pkg, initializers, genDecl)
		return true
	})
}

// varGenDecl keeps only var declarations for initializer indexing.
func varGenDecl(node ast.Node) (*ast.GenDecl, bool) {
	genDecl, ok := node.(*ast.GenDecl)
	if !ok || genDecl.Tok != token.VAR {
		return nil, false
	}
	return genDecl, true
}

// recordVarDeclInitializers records every explicit initializer from one var
// declaration block.
func recordVarDeclInitializers(pkg *packages.Package, initializers map[types.Object][]assignmentRef, genDecl *ast.GenDecl) {
	for _, spec := range genDecl.Specs {
		recordValueSpecInitializer(pkg, initializers, spec)
	}
}

// recordValueSpecInitializer stores the initializer expressions for one value
// spec when names and values line up.
func recordValueSpecInitializer(pkg *packages.Package, initializers map[types.Object][]assignmentRef, spec ast.Spec) {
	valueSpec, ok := spec.(*ast.ValueSpec)
	if !ok || len(valueSpec.Values) != len(valueSpec.Names) {
		return
	}

	for i, name := range valueSpec.Names {
		appendInitializer(pkg, initializers, name, name.Pos(), valueSpec.Values[i])
	}
}

// recordAssignInitializers appends later reassignments so call-site lookup can
// choose the visible value.
func recordAssignInitializers(pkg *packages.Package, initializers map[types.Object][]assignmentRef, file *ast.File) {
	ast.Inspect(file, func(node ast.Node) bool {
		assign, ok := node.(*ast.AssignStmt)
		if !ok || len(assign.Lhs) != len(assign.Rhs) {
			return true
		}
		for i, lhs := range assign.Lhs {
			ident, ok := lhs.(*ast.Ident)
			if !ok {
				continue
			}
			appendInitializer(pkg, initializers, ident, ident.Pos(), assign.Rhs[i])
		}
		return true
	})
}

// appendInitializer records one initializer or reassignment for later
// call-site lookup.
func appendInitializer(pkg *packages.Package, initializers map[types.Object][]assignmentRef, ident *ast.Ident, pos token.Pos, expr ast.Expr) {
	if ident == nil {
		return
	}

	if obj := pkg.TypesInfo.ObjectOf(ident); obj != nil {
		initializers[obj] = append(initializers[obj], assignmentRef{pos: pos, expr: expr})
	}
}

// shouldSkipPackageFile centralizes the compiled-file bounds check used by the
// registration and helper indexes.
func shouldSkipPackageFile(pkg *packages.Package, fileIdx int, skip func(string) bool) bool {
	if fileIdx >= len(pkg.CompiledGoFiles) {
		return true
	}
	return skip(pkg.CompiledGoFiles[fileIdx])
}

// latestAssignmentBefore returns the nearest initializer or reassignment
// visible before the registration call site.
func latestAssignmentBefore(assignments []assignmentRef, pos token.Pos) (assignmentRef, bool) {
	for i := len(assignments) - 1; i >= 0; i-- {
		if pos == token.NoPos || assignments[i].pos < pos {
			return assignments[i], true
		}
	}
	return assignmentRef{}, false
}

// parameterIndexes maps named parameters to their ordinal positions for
// wrapper forwarding checks.
func parameterIndexes(pkg *packages.Package, decl *ast.FuncDecl) map[types.Object]int {
	params := make(map[types.Object]int)
	if decl.Type == nil || decl.Type.Params == nil {
		return params
	}

	index := 0
	for _, field := range decl.Type.Params.List {
		for _, name := range field.Names {
			if obj := pkg.TypesInfo.Defs[name]; obj != nil {
				params[obj] = index
			}
			index++
		}
	}
	return params
}

// functionKeyForObject turns a package-level function object into the stable
// lookup key used by wrapper tracing.
func functionKeyForObject(obj types.Object) (functionKey, bool) {
	if obj == nil || obj.Pkg() == nil {
		return functionKey{}, false
	}
	_, ok := obj.(*types.Func)
	if !ok {
		return functionKey{}, false
	}
	return functionKey{pkgPath: obj.Pkg().Path(), name: obj.Name()}, true
}

// registeredTypeKeyForType unwraps aliases and pointers until it reaches the
// exported named type that should be tracked for implementation findings.
func registeredTypeKeyForType(typ types.Type) (registeredTypeKey, bool) {
	for typ != nil {
		typ = types.Unalias(typ)
		switch typedType := typ.(type) {
		case *types.Pointer:
			typ = typedType.Elem()
		case *types.Named:
			obj := typedType.Obj()
			if obj == nil || obj.Pkg() == nil {
				return registeredTypeKey{}, false
			}
			if !token.IsExported(obj.Name()) {
				return registeredTypeKey{}, false
			}
			return registeredTypeKey{pkgPath: obj.Pkg().Path(), typeName: obj.Name()}, true
		default:
			return registeredTypeKey{}, false
		}
	}
	return registeredTypeKey{}, false
}

func packagePath(pkg *packages.Package) string {
	if pkg != nil && pkg.Types != nil {
		return pkg.Types.Path()
	}
	if pkg != nil {
		return pkg.PkgPath
	}
	return ""
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

// shouldSkipFindingFile filters generated and test sources so the tool focuses
// reported findings on user-authored service contracts instead of stubs.
func shouldSkipFindingFile(filename string) bool {
	base := filepath.Base(filename)
	return strings.HasSuffix(base, "_test.go") ||
		strings.HasSuffix(base, ".pb.go") ||
		strings.HasSuffix(base, ".triple.go")
}

// shouldSkipRegistrationFile keeps helper discovery out of tests and protobuf
// stubs, but still allows generated Triple wrappers to participate in
// registration tracing.
func shouldSkipRegistrationFile(filename string) bool {
	base := filepath.Base(filename)
	return strings.HasSuffix(base, "_test.go") ||
		strings.HasSuffix(base, ".pb.go")
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
