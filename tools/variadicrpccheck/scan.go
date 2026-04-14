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

// Scan uses syntax plus type information because interface declarations cannot
// be discovered from runtime reflection, and exported implementations need type
// checks to distinguish RPC-style variadics from option-style helper APIs.
func Scan(dir string, patterns []string) ([]Finding, error) {
	if len(patterns) == 0 {
		patterns = []string{"./..."}
	}

	cfg := &packages.Config{
		Mode: packages.NeedName |
			packages.NeedFiles |
			packages.NeedCompiledGoFiles |
			packages.NeedSyntax |
			packages.NeedTypes |
			packages.NeedTypesInfo,
		Dir: dir,
	}

	pkgs, err := packages.Load(cfg, patterns...)
	if err != nil {
		return nil, err
	}

	findings := make([]Finding, 0)
	loadErrs := make([]string, 0)
	for _, pkg := range pkgs {
		for _, pkgErr := range pkg.Errors {
			loadErrs = append(loadErrs, pkgErr.Error())
		}
		findings = append(findings, collectPackageFindings(pkg)...)
	}

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

	if len(loadErrs) > 0 {
		return findings, errors.New(strings.Join(loadErrs, "; "))
	}

	return findings, nil
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
		typeSpec, ok := spec.(*ast.TypeSpec)
		if !ok || !typeSpec.Name.IsExported() {
			continue
		}

		iface, ok := typeSpec.Type.(*ast.InterfaceType)
		if !ok {
			continue
		}

		obj := pkg.TypesInfo.Defs[typeSpec.Name]
		if obj == nil {
			continue
		}

		ifaceType, ok := obj.Type().Underlying().(*types.Interface)
		if !ok {
			continue
		}

		methodPositions := interfaceMethodPositions(pkg, iface)
		ifaceType.Complete()
		for i := 0; i < ifaceType.NumMethods(); i++ {
			method := ifaceType.Method(i)
			if !method.Exported() {
				continue
			}
			if !isCandidateRPCMethodName(method.Name()) {
				continue
			}

			sig, ok := method.Type().(*types.Signature)
			if !ok || !isVariadicRPCSignature(sig) {
				continue
			}

			pos := methodPositions[method.Name()]
			if !pos.IsValid() {
				pos = typeSpec.Name.Pos()
			}

			findings = append(findings, Finding{
				Position:   pkg.Fset.Position(pos),
				Kind:       "interface",
				TypeName:   typeSpec.Name.Name,
				MethodName: method.Name(),
			})
		}
	}

	return findings
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

func interfaceMethodPositions(pkg *packages.Package, iface *ast.InterfaceType) map[string]token.Pos {
	positions := make(map[string]token.Pos)
	if iface == nil || iface.Methods == nil {
		return positions
	}

	for _, field := range iface.Methods.List {
		if len(field.Names) == 1 {
			methodName := field.Names[0]
			if methodName.IsExported() {
				positions[methodName.Name] = methodName.Pos()
			}
			continue
		}

		embeddedIface := embeddedInterfaceType(pkg, field.Type)
		if embeddedIface == nil {
			continue
		}

		embeddedIface.Complete()
		for i := 0; i < embeddedIface.NumMethods(); i++ {
			method := embeddedIface.Method(i)
			if method.Exported() {
				if _, exists := positions[method.Name()]; !exists {
					positions[method.Name()] = field.Type.Pos()
				}
			}
		}
	}

	return positions
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
