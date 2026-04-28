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
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	goModFileName              = "go.mod"
	serviceFileName            = "service.go"
	variadicCheckModulePath    = "example.com/variadicrpccheck"
	variadicCheckModuleContent = "module example.com/variadicrpccheck\n\ngo 1.25.0\n"
)

func TestScanFindsVariadicRPCContracts(t *testing.T) {
	dir := t.TempDir()
	writeTempFile(t, dir, goModFileName, goModuleContentWithDubboGoReplace("example.com/variadicrpccheck", repoRoot(t)))
	writeTempFile(t, dir, serviceFileName, `package sample

import (
	"context"

	"dubbo.apache.org/dubbo-go/v3/server"
)

type Option struct{}
type hidden struct{}

type MultiArgsService interface {
	MultiArgs(ctx context.Context, args ...string) error
	Configure(opts ...Option) error
}

type VariadicService struct{}

func (s *VariadicService) MultiArgs(ctx context.Context, args ...string) error {
	return nil
}

func (s *VariadicService) Configure(opts ...Option) error {
	return nil
}

type HiddenReplyService struct{}

func (s *HiddenReplyService) MultiArgs(ctx context.Context, args ...string) (hidden, error) {
	return hidden{}, nil
}

type HiddenArgService struct{}

func (s *HiddenArgService) MultiArgs(ctx context.Context, arg hidden, args ...string) error {
	return nil
}

type PlainService struct{}

func (s *PlainService) Echo(ctx context.Context, arg string) error {
	return nil
}

type Helper struct{}

func (h *Helper) Merge(ctx context.Context, values ...string) error {
	return nil
}

func register() {
	var srv *server.Server
	_ = srv.RegisterService(&VariadicService{})
}
`)
	writeTempFile(t, dir, "generated.pb.go", `package sample

import "context"

type GeneratedService struct{}

func (s *GeneratedService) MultiArgs(ctx context.Context, args ...string) error {
	return nil
}
`)
	writeTempFile(t, dir, "transport.triple.go", `package sample

import "context"

type TripleGeneratedService struct{}

func (s *TripleGeneratedService) MultiArgs(ctx context.Context, args ...string) error {
	return nil
}
`)
	writeTempFile(t, dir, "ignored_test.go", `package sample

import "context"

type TestOnlyService struct{}

func (s *TestOnlyService) MultiArgs(ctx context.Context, args ...string) error {
	return nil
}
`)

	findings, err := Scan(dir, []string{"./..."})
	require.NoError(t, err)

	got := make([]string, 0, len(findings))
	for _, finding := range findings {
		got = append(got, fmt.Sprintf("%s:%s:%s", finding.Kind, finding.TypeName, finding.MethodName))
	}

	assert.ElementsMatch(t, []string{
		"implementation:VariadicService:MultiArgs",
		"interface:MultiArgsService:MultiArgs",
	}, got)
}

func TestScanFindsEmbeddedImportedVariadicInterface(t *testing.T) {
	baseDir := t.TempDir()
	writeTempFile(t, baseDir, goModFileName, goModuleContent("example.com/base"))
	writeTempFile(t, baseDir, "base.go", `package base

import "context"

type BaseService interface {
	MultiArgs(ctx context.Context, args ...string) error
}
	`)

	dir := t.TempDir()
	writeTempFile(t, dir, goModFileName, fmt.Sprintf(`%s

	require example.com/base v0.0.0

	replace example.com/base => %s
`, goModuleContent("example.com/local"), baseDir))
	writeTempFile(t, dir, serviceFileName, `package local

import "example.com/base"

type WrappedService interface {
	base.BaseService
}
`)

	findings, err := Scan(dir, []string{"./..."})
	require.NoError(t, err)
	require.Len(t, findings, 1)
	assert.Equal(t, "interface", findings[0].Kind)
	assert.Equal(t, "WrappedService", findings[0].TypeName)
	assert.Equal(t, "MultiArgs", findings[0].MethodName)
	assert.Equal(t, filepath.Join(dir, "service.go"), findings[0].Position.Filename)
	assert.Equal(t, 6, findings[0].Position.Line)
}

func TestScanFindsRegisteredVariadicImplementationAcrossPackages(t *testing.T) {
	dir := t.TempDir()
	writeTempFile(t, dir, goModFileName, goModuleContentWithDubboGoReplace("example.com/crosspkg", repoRoot(t)))
	writeTempFile(t, filepath.Join(dir, "service"), serviceFileName, `package service

import "context"

type VariadicService struct{}

func (s *VariadicService) MultiArgs(ctx context.Context, args ...string) error {
	return nil
}

type Helper struct{}

func (h *Helper) Merge(ctx context.Context, values ...string) error {
	return nil
}
`)
	writeTempFile(t, filepath.Join(dir, "provider"), "provider.go", `package provider

import (
	"dubbo.apache.org/dubbo-go/v3/server"
	"example.com/crosspkg/service"
)

func register() {
	var srv *server.Server
	_ = srv.RegisterService(&service.VariadicService{})
}
`)

	findings, err := Scan(dir, []string{"./..."})
	require.NoError(t, err)
	require.Len(t, findings, 1)
	assert.Equal(t, "implementation", findings[0].Kind)
	assert.Equal(t, "VariadicService", findings[0].TypeName)
	assert.Equal(t, "MultiArgs", findings[0].MethodName)
	assert.Equal(t, filepath.Join(dir, "service", "service.go"), findings[0].Position.Filename)
}

func TestScanFindsVariadicImplementationRegisteredViaSetProviderServiceWithInfo(t *testing.T) {
	dir := t.TempDir()
	writeTempFile(t, dir, goModFileName, goModuleContentWithDubboGoReplace("example.com/providerinfo", repoRoot(t)))
	writeTempFile(t, filepath.Join(dir, "service"), serviceFileName, `package service

import "context"

type VariadicService struct{}

func (s *VariadicService) MultiArgs(ctx context.Context, args ...string) error {
	return nil
}

type Helper struct{}

func (h *Helper) Merge(ctx context.Context, values ...string) error {
	return nil
}
`)
	writeTempFile(t, filepath.Join(dir, "provider"), "provider.go", `package provider

import (
	"dubbo.apache.org/dubbo-go/v3/config"
	"example.com/providerinfo/service"
)

func init() {
	config.SetProviderServiceWithInfo(&service.VariadicService{}, nil)
}
`)

	findings, err := Scan(dir, []string{"./..."})
	require.NoError(t, err)
	require.Len(t, findings, 1)
	assert.Equal(t, "implementation", findings[0].Kind)
	assert.Equal(t, "VariadicService", findings[0].TypeName)
	assert.Equal(t, "MultiArgs", findings[0].MethodName)
	assert.Equal(t, filepath.Join(dir, "service", "service.go"), findings[0].Position.Filename)
}

func TestScanFindsVariadicImplementationRegisteredViaRootSetProviderService(t *testing.T) {
	dir := t.TempDir()
	writeTempFile(t, dir, goModFileName, goModuleContentWithDubboGoReplace("example.com/rootprovider", repoRoot(t)))
	writeTempFile(t, filepath.Join(dir, "service"), serviceFileName, `package service

import "context"

type VariadicService struct{}

func (s *VariadicService) MultiArgs(ctx context.Context, args ...string) error {
	return nil
}
`)
	writeTempFile(t, filepath.Join(dir, "provider"), "provider.go", `package provider

import (
	dubbo "dubbo.apache.org/dubbo-go/v3"
	"example.com/rootprovider/service"
)

func init() {
	dubbo.SetProviderService(&service.VariadicService{})
}
`)

	findings, err := Scan(dir, []string{"./..."})
	require.NoError(t, err)
	require.Len(t, findings, 1)
	assert.Equal(t, "implementation", findings[0].Kind)
	assert.Equal(t, "VariadicService", findings[0].TypeName)
	assert.Equal(t, "MultiArgs", findings[0].MethodName)
	assert.Equal(t, filepath.Join(dir, "service", "service.go"), findings[0].Position.Filename)
}

func TestScanFindsVariadicImplementationRegisteredViaGeneratedWrapper(t *testing.T) {
	dir := t.TempDir()
	writeTempFile(t, dir, goModFileName, goModuleContentWithDubboGoReplace("example.com/generatedwrapper", repoRoot(t)))
	writeTempFile(t, filepath.Join(dir, "service"), serviceFileName, `package service

import "context"

type VariadicService struct{}

func (s *VariadicService) MultiArgs(ctx context.Context, args ...string) error {
	return nil
}
`)
	writeTempFile(t, filepath.Join(dir, "generated"), "generated.go", `package generated

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/config"
)

func SetProviderService(srv common.RPCService) {
	config.SetProviderServiceWithInfo(srv, nil)
}
`)
	writeTempFile(t, filepath.Join(dir, "provider"), "provider.go", `package provider

import (
	"example.com/generatedwrapper/generated"
	"example.com/generatedwrapper/service"
)

func init() {
	generated.SetProviderService(&service.VariadicService{})
}
`)

	findings, err := Scan(dir, []string{"./..."})
	require.NoError(t, err)
	require.Len(t, findings, 1)
	assert.Equal(t, "implementation", findings[0].Kind)
	assert.Equal(t, "VariadicService", findings[0].TypeName)
	assert.Equal(t, "MultiArgs", findings[0].MethodName)
	assert.Equal(t, filepath.Join(dir, "service", "service.go"), findings[0].Position.Filename)
}

func TestScanFindsVariadicImplementationRegisteredViaTripleWrapper(t *testing.T) {
	dir := t.TempDir()
	writeTempFile(t, dir, goModFileName, goModuleContentWithDubboGoReplace("example.com/triplewrapper", repoRoot(t)))
	writeTempFile(t, filepath.Join(dir, "service"), serviceFileName, `package service

import "context"

type VariadicService struct{}

func (s *VariadicService) MultiArgs(ctx context.Context, args ...string) error {
	return nil
}
`)
	writeTempFile(t, filepath.Join(dir, "generated"), "generated.triple.go", `package generated

import (
	dubbo "dubbo.apache.org/dubbo-go/v3"
	"dubbo.apache.org/dubbo-go/v3/common"
)

func SetProviderService(srv common.RPCService) {
	dubbo.SetProviderServiceWithInfo(srv, nil)
}
`)
	writeTempFile(t, filepath.Join(dir, "provider"), "provider.go", `package provider

import (
	"example.com/triplewrapper/generated"
	"example.com/triplewrapper/service"
)

func init() {
	generated.SetProviderService(&service.VariadicService{})
}
`)

	findings, err := Scan(dir, []string{"./..."})
	require.NoError(t, err)
	require.Len(t, findings, 1)
	assert.Equal(t, "implementation", findings[0].Kind)
	assert.Equal(t, "VariadicService", findings[0].TypeName)
	assert.Equal(t, "MultiArgs", findings[0].MethodName)
	assert.Equal(t, filepath.Join(dir, "service", "service.go"), findings[0].Position.Filename)
}

func TestScanFindsVariadicImplementationRegisteredViaInterfaceVariable(t *testing.T) {
	dir := t.TempDir()
	writeTempFile(t, dir, goModFileName, goModuleContentWithDubboGoReplace("example.com/interfacevar", repoRoot(t)))
	writeTempFile(t, filepath.Join(dir, "service"), serviceFileName, `package service

import "context"

type VariadicService struct{}

func (s *VariadicService) MultiArgs(ctx context.Context, args ...string) error {
	return nil
}
`)
	writeTempFile(t, filepath.Join(dir, "provider"), "provider.go", `package provider

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/config"
	"example.com/interfacevar/service"
)

func init() {
	var svc common.RPCService = &service.VariadicService{}
	config.SetProviderServiceWithInfo(svc, nil)
}
`)

	findings, err := Scan(dir, []string{"./..."})
	require.NoError(t, err)
	require.Len(t, findings, 1)
	assert.Equal(t, "implementation", findings[0].Kind)
	assert.Equal(t, "VariadicService", findings[0].TypeName)
	assert.Equal(t, "MultiArgs", findings[0].MethodName)
	assert.Equal(t, filepath.Join(dir, "service", "service.go"), findings[0].Position.Filename)
}

func TestScanUsesInterfaceVariableTypeAtRegistrationCallSite(t *testing.T) {
	dir := t.TempDir()
	writeTempFile(t, dir, goModFileName, goModuleContentWithDubboGoReplace("example.com/interfacevarreassign", repoRoot(t)))
	writeTempFile(t, filepath.Join(dir, "service"), serviceFileName, `package service

import "context"

type FirstService struct{}

func (s *FirstService) MultiArgs(ctx context.Context, args ...string) error {
	return nil
}

type SecondService struct{}

func (s *SecondService) MultiArgs(ctx context.Context, args ...string) error {
	return nil
}
`)
	writeTempFile(t, filepath.Join(dir, "provider"), "provider.go", `package provider

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/config"
	"example.com/interfacevarreassign/service"
)

func init() {
	var svc common.RPCService = &service.FirstService{}
	config.SetProviderServiceWithInfo(svc, nil)
	svc = &service.SecondService{}
}
`)

	findings, err := Scan(dir, []string{"./..."})
	require.NoError(t, err)
	require.Len(t, findings, 1)
	assert.Equal(t, "implementation", findings[0].Kind)
	assert.Equal(t, "FirstService", findings[0].TypeName)
	assert.Equal(t, "MultiArgs", findings[0].MethodName)
	assert.Equal(t, filepath.Join(dir, "service", "service.go"), findings[0].Position.Filename)
}

func TestRunPrintsWarningsButReturnsZero(t *testing.T) {
	dir := t.TempDir()
	writeTempFile(t, dir, goModFileName, goModuleContentWithDubboGoReplace("example.com/variadicrpccheck", repoRoot(t)))
	writeTempFile(t, dir, serviceFileName, `package sample

import (
	"context"

	"dubbo.apache.org/dubbo-go/v3/server"
)

type VariadicService struct{}

func (s *VariadicService) MultiArgs(ctx context.Context, args ...string) error {
	return nil
}

func register() {
	var srv *server.Server
	_ = srv.RegisterService(&VariadicService{})
}
`)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := run(&stdout, &stderr, dir, []string{"./..."})

	assert.Equal(t, 0, code)
	assert.Contains(t, stdout.String(), "warning: implementation VariadicService exports variadic RPC method MultiArgs")
	assert.Empty(t, stderr.String())
}

func TestRunReportsScanErrorButReturnsZero(t *testing.T) {
	dir := t.TempDir()
	writeTempFile(t, dir, goModFileName, variadicCheckModuleContent)
	writeTempFile(t, dir, "broken.go", "package sample\n\nfunc broken(\n")

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := run(&stdout, &stderr, dir, []string{"./..."})

	assert.Equal(t, 0, code)
	assert.Empty(t, stdout.String())
	assert.Contains(t, stderr.String(), "variadicrpccheck:")
}

func goModuleContent(modulePath string) string {
	if modulePath == variadicCheckModulePath {
		return variadicCheckModuleContent
	}
	return fmt.Sprintf("module %s\n\ngo 1.25.0\n", modulePath)
}

func goModuleContentWithDubboGoReplace(modulePath, dubboGoPath string) string {
	return fmt.Sprintf(`module %s

go 1.25.0

require dubbo.apache.org/dubbo-go/v3 v3.0.0

replace dubbo.apache.org/dubbo-go/v3 => %s
`, modulePath, dubboGoPath)
}

func writeTempFile(t *testing.T, dir, name, content string) {
	t.Helper()
	path := filepath.Join(dir, name)
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
	require.NoError(t, os.WriteFile(path, []byte(content), 0o644))
}

func repoRoot(t *testing.T) string {
	t.Helper()
	_, filename, _, ok := runtime.Caller(0)
	require.True(t, ok)
	return filepath.Clean(filepath.Join(filepath.Dir(filename), "..", ".."))
}
