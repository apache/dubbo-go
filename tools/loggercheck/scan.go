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
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

import (
	"golang.org/x/tools/go/packages"
)

var packagesLoad = packages.Load

const (
	gostLoggerPkgPath = "github.com/dubbogo/gost/log/logger"
)

// loggerMethods without the "f" suffix.
var nonFormattedMethods = map[string]bool{
	"Info":  true,
	"Warn":  true,
	"Error": true,
	"Debug": true,
}

// loggerMethodPairs maps non-f to f version.
var loggerMethodPairs = map[string]string{
	"Info":  "Infof",
	"Warn":  "Warnf",
	"Error": "Errorf",
	"Debug": "Debugf",
}

// formattedMethods are the *f variants.
var formattedMethods = map[string]bool{
	"Infof":  true,
	"Warnf":  true,
	"Errorf": true,
	"Debugf": true,
}

var fmtSprintfSelectorRegex = regexp.MustCompile(`^fmt\.Sprintf$`)

type Finding struct {
	Position token.Position
	Message  string
}

func (f Finding) String() string {
	return fmt.Sprintf("%s:%d:%d: warning: %s",
		f.Position.Filename, f.Position.Line, f.Position.Column, f.Message)
}

// Scan loads packages and collects logger format findings.
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

func normalizePatterns(patterns []string) []string {
	if len(patterns) == 0 {
		return []string{"./..."}
	}
	return patterns
}

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

func collectFindingsAndErrors(pkgs []*packages.Package) ([]Finding, []string) {
	findings := make([]Finding, 0)
	loadErrs := make([]string, 0)
	for _, pkg := range pkgs {
		loadErrs = append(loadErrs, packageErrors(pkg)...)
		findings = append(findings, collectPackageFindings(pkg)...)
	}
	return findings, loadErrs
}

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

func sortFindings(findings []Finding) {
	sort.Slice(findings, func(i, j int) bool {
		if findings[i].Position.Filename != findings[j].Position.Filename {
			return findings[i].Position.Filename < findings[j].Position.Filename
		}
		return findings[i].Position.Line < findings[j].Position.Line
	})
}

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
		findings = append(findings, collectFileFindings(pkg, file)...)
	}
	return findings
}

func shouldSkipFile(filename string) bool {
	base := filepath.Base(filename)
	return strings.HasSuffix(base, "_test.go") ||
		strings.HasSuffix(base, ".pb.go") ||
		strings.HasSuffix(base, ".triple.go")
}

func collectFileFindings(pkg *packages.Package, file *ast.File) []Finding {
	findings := make([]Finding, 0)
	ast.Inspect(file, func(node ast.Node) bool {
		call, ok := node.(*ast.CallExpr)
		if !ok {
			return true
		}
		sel, ok := call.Fun.(*ast.SelectorExpr)
		if !ok {
			return true
		}
		if !isLoggerCall(pkg, sel) {
			return true
		}

		methodName := sel.Sel.Name

		// Rule 1: non-f method with format args
		if nonFormattedMethods[methodName] {
			if f := checkNonFormattedWithArgs(pkg, call, methodName); f != nil {
				findings = append(findings, *f)
			}
		}

		// Rule 2: *f method without format args
		if formattedMethods[methodName] {
			if f := checkFormattedWithoutArgs(pkg, call, methodName); f != nil {
				findings = append(findings, *f)
			}
		}

		// Rule 3: *f(fmt.Sprintf(...))
		if formattedMethods[methodName] {
			if f := checkFmtSprintfWrap(pkg, call, methodName); f != nil {
				findings = append(findings, *f)
			}
		}

		// Rule 4: check message decorations (\n, ..., ! at end)
		if f := checkMessageDecorations(pkg, call, methodName); f != nil {
			findings = append(findings, *f)
		}

		return true
	})
	return findings
}

func isLoggerCall(pkg *packages.Package, sel *ast.SelectorExpr) bool {
	obj := pkg.TypesInfo.Uses[sel.Sel]
	if obj == nil || obj.Pkg() == nil {
		return false
	}
	return obj.Pkg().Path() == gostLoggerPkgPath
}

// Rule 1: logger.Info("...%s...", arg) should be logger.Infof
func checkNonFormattedWithArgs(pkg *packages.Package, call *ast.CallExpr, methodName string) *Finding {
	if len(call.Args) == 0 {
		return nil
	}
	lit, ok := call.Args[0].(*ast.BasicLit)
	if !ok || lit.Kind != token.STRING {
		return nil
	}
	formatStr := lit.Value
	// Check if the first arg contains format verbs like %s, %v, %d, %+v, %#v
	if !hasFormatVerbs(formatStr) {
		return nil
	}
	// Only flag if there are extra args that would be consumed by format verbs
	if len(call.Args) <= 1 {
		return nil
	}

	correct := loggerMethodPairs[methodName]
	return &Finding{
		Position: pkg.Fset.Position(call.Pos()),
		Message:  fmt.Sprintf("logger.%s called with format verb and %d args, use logger.%s instead", methodName, len(call.Args), correct),
	}
}

// Rule 2: logger.Infof("no args") should be logger.Info
func checkFormattedWithoutArgs(pkg *packages.Package, call *ast.CallExpr, methodName string) *Finding {
	if len(call.Args) != 1 {
		return nil
	}
	lit, ok := call.Args[0].(*ast.BasicLit)
	if !ok || lit.Kind != token.STRING {
		return nil
	}
	// If the string has format verbs it's fine
	if hasFormatVerbs(lit.Value) {
		return nil
	}

	nonF := strings.TrimSuffix(methodName, "f")
	return &Finding{
		Position: pkg.Fset.Position(call.Pos()),
		Message:  fmt.Sprintf("logger.%s called with no format args, use logger.%s instead", methodName, nonF),
	}
}

// Rule 3: logger.Infof(fmt.Sprintf(...)) is an anti-pattern
func checkFmtSprintfWrap(pkg *packages.Package, call *ast.CallExpr, methodName string) *Finding {
	if len(call.Args) == 0 {
		return nil
	}
	inner, ok := call.Args[0].(*ast.CallExpr)
	if !ok {
		return nil
	}
	innerSel, ok := inner.Fun.(*ast.SelectorExpr)
	if !ok {
		return nil
	}
	// Check if it's fmt.Sprintf
	if ident, ok := innerSel.X.(*ast.Ident); ok {
		if ident.Name == "fmt" && innerSel.Sel.Name == "Sprintf" {
			return &Finding{
				Position: pkg.Fset.Position(call.Pos()),
				Message:  fmt.Sprintf("logger.%s(fmt.Sprintf(...)) is redundant, use logger.%s directly with format args", methodName, methodName),
			}
		}
	}
	return nil
}

// Rule 4: check for trailing \n, ..., ! in the message (not in key=value area)
func checkMessageDecorations(pkg *packages.Package, call *ast.CallExpr, methodName string) *Finding {
	if len(call.Args) == 0 {
		return nil
	}
	lit, ok := call.Args[0].(*ast.BasicLit)
	if !ok || lit.Kind != token.STRING {
		return nil
	}
	msg := lit.Value

	// Check trailing newline
	if strings.HasSuffix(msg, `\n"`) {
		return &Finding{
			Position: pkg.Fset.Position(call.Pos()),
			Message:  fmt.Sprintf("logger.%s message ends with \\n, remove trailing newline", methodName),
		}
	}

	// Unquote to check message content
	unquoted, err := strUnquote(msg)
	if err != nil {
		return nil
	}

	// Check trailing "..." in message
	if strings.HasSuffix(unquoted, "...") {
		return &Finding{
			Position: pkg.Fset.Position(call.Pos()),
			Message:  fmt.Sprintf("logger.%s message ends with '...', remove trailing dots", methodName),
		}
	}

	// Check trailing "!" in message
	if strings.HasSuffix(unquoted, "!") {
		return &Finding{
			Position: pkg.Fset.Position(call.Pos()),
			Message:  fmt.Sprintf("logger.%s message ends with '!', remove trailing exclamation", methodName),
		}
	}

	return nil
}

var formatVerbRegex = regexp.MustCompile(`%[+#\- 0]*(?:\d+)?(?:\.\d+)?[a-zA-Z]`)

func hasFormatVerbs(s string) bool {
	return formatVerbRegex.MatchString(s)
}

// strUnquote removes surrounding backticks or quotes from a Go string literal.
func strUnquote(s string) (string, error) {
	if len(s) < 2 {
		return "", fmt.Errorf("string too short: %s", s)
	}
	// Raw string literal: `...`
	if s[0] == '`' {
		return s[1 : len(s)-1], nil
	}
	// Interpreted string literal: "..."
	if s[0] == '"' {
		// Use simple unescaping for the check purposes
		return s[1 : len(s)-1], nil
	}
	return s, nil
}
