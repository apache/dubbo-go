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
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
)

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestScanFindsNonFormattedWithFormatArgs(t *testing.T) {
	dir := t.TempDir()
	writeTempFile(t, dir, "go.mod", goModuleContentWithReplace("example.com/loggercheck", repoRoot(t)))
	writeTempFile(t, dir, "main.go", `package main

import "github.com/dubbogo/gost/log/logger"

func main() {
	err := error(nil)
	logger.Error("something failed, err=%v", err)
}
`)
	tidyModule(t, dir)
	findings, err := Scan(dir, []string{"./..."})
	require.NoError(t, err)
	require.Len(t, findings, 1)
	assert.Contains(t, findings[0].Message, "logger.Error called with format verb")
	assert.Contains(t, findings[0].Message, "use logger.Errorf")
}

func TestScanFindsFormattedWithoutArgs(t *testing.T) {
	dir := t.TempDir()
	writeTempFile(t, dir, "go.mod", goModuleContentWithReplace("example.com/loggercheck", repoRoot(t)))
	writeTempFile(t, dir, "main.go", `package main

import "github.com/dubbogo/gost/log/logger"

func main() {
	logger.Infof("no format args here")
}
`)
	tidyModule(t, dir)
	findings, err := Scan(dir, []string{"./..."})
	require.NoError(t, err)
	require.Len(t, findings, 1)
	assert.Contains(t, findings[0].Message, "use logger.Info")
}

func TestScanFindsFmtSprintfWrap(t *testing.T) {
	dir := t.TempDir()
	writeTempFile(t, dir, "go.mod", goModuleContentWithReplace("example.com/loggercheck", repoRoot(t)))
	writeTempFile(t, dir, "main.go", `package main

import (
	"fmt"
	"github.com/dubbogo/gost/log/logger"
)

func main() {
	logger.Infof(fmt.Sprintf("value is %v", 42))
}
`)
	tidyModule(t, dir)
	findings, err := Scan(dir, []string{"./..."})
	require.NoError(t, err)
	require.Len(t, findings, 1)
	assert.Contains(t, findings[0].Message, "logger.Infof(fmt.Sprintf(...)) is redundant")
}

func TestScanFindsTrailingNewline(t *testing.T) {
	dir := t.TempDir()
	writeTempFile(t, dir, "go.mod", goModuleContentWithReplace("example.com/loggercheck", repoRoot(t)))
	writeTempFile(t, dir, "main.go", `package main

import "github.com/dubbogo/gost/log/logger"

func main() {
	logger.Info("service exported\n")
}
`)
	tidyModule(t, dir)
	findings, err := Scan(dir, []string{"./..."})
	require.NoError(t, err)
	require.Len(t, findings, 1)
	assert.Contains(t, findings[0].Message, "ends with \\n")
}

func TestScanFindsTrailingDots(t *testing.T) {
	dir := t.TempDir()
	writeTempFile(t, dir, "go.mod", goModuleContentWithReplace("example.com/loggercheck", repoRoot(t)))
	writeTempFile(t, dir, "main.go", `package main

import "github.com/dubbogo/gost/log/logger"

func main() {
	logger.Info("file watcher is stopping...")
}
`)
	tidyModule(t, dir)
	findings, err := Scan(dir, []string{"./..."})
	require.NoError(t, err)
	require.Len(t, findings, 1)
	assert.Contains(t, findings[0].Message, "ends with '...'")
}

func TestScanFindsTrailingExclamation(t *testing.T) {
	dir := t.TempDir()
	writeTempFile(t, dir, "go.mod", goModuleContentWithReplace("example.com/loggercheck", repoRoot(t)))
	writeTempFile(t, dir, "main.go", `package main

import "github.com/dubbogo/gost/log/logger"

func main() {
	logger.Info("hot reload completed successfully!")
}
`)
	tidyModule(t, dir)
	findings, err := Scan(dir, []string{"./..."})
	require.NoError(t, err)
	require.Len(t, findings, 1)
	assert.Contains(t, findings[0].Message, "ends with '!'")
}

func TestScanAllowsValidFormattedCall(t *testing.T) {
	dir := t.TempDir()
	writeTempFile(t, dir, "go.mod", goModuleContentWithReplace("example.com/loggercheck", repoRoot(t)))
	writeTempFile(t, dir, "main.go", `package main

import "github.com/dubbogo/gost/log/logger"

func main() {
	logger.Infof("[Server] registering service=%s", "demo")
}
`)
	tidyModule(t, dir)
	findings, err := Scan(dir, []string{"./..."})
	require.NoError(t, err)
	require.Empty(t, findings)
}

func TestScanAllowsValidNonFormattedCall(t *testing.T) {
	dir := t.TempDir()
	writeTempFile(t, dir, "go.mod", goModuleContentWithReplace("example.com/loggercheck", repoRoot(t)))
	writeTempFile(t, dir, "main.go", `package main

import "github.com/dubbogo/gost/log/logger"

func main() {
	logger.Info("[Server] service exported")
}
`)
	tidyModule(t, dir)
	findings, err := Scan(dir, []string{"./..."})
	require.NoError(t, err)
	require.Empty(t, findings)
}

func TestScanSkipsNonLoggerCalls(t *testing.T) {
	dir := t.TempDir()
	writeTempFile(t, dir, "go.mod", goModuleContentWithReplace("example.com/loggercheck", repoRoot(t)))
	writeTempFile(t, dir, "main.go", `package main

import "fmt"

func main() {
	fmt.Printf("hello %s\n", "world")
}
`)
	tidyModule(t, dir)
	findings, err := Scan(dir, []string{"./..."})
	require.NoError(t, err)
	require.Empty(t, findings)
}

func TestScanFindsMultipleViolations(t *testing.T) {
	dir := t.TempDir()
	writeTempFile(t, dir, "go.mod", goModuleContentWithReplace("example.com/loggercheck", repoRoot(t)))
	writeTempFile(t, dir, "main.go", `package main

import "github.com/dubbogo/gost/log/logger"

func main() {
	err := error(nil)
	logger.Error("failed, err=%v", err)
	logger.Infof("no args")
}
`)
	tidyModule(t, dir)
	findings, err := Scan(dir, []string{"./..."})
	require.NoError(t, err)
	assert.Len(t, findings, 2)
}

func TestRunPrintsWarningsButReturnsZero(t *testing.T) {
	dir := t.TempDir()
	writeTempFile(t, dir, "go.mod", goModuleContentWithReplace("example.com/loggercheck", repoRoot(t)))
	writeTempFile(t, dir, "main.go", `package main

import "github.com/dubbogo/gost/log/logger"

func main() {
	logger.Infof("no args here")
}
`)
	tidyModule(t, dir)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := run(&stdout, &stderr, dir, []string{"./..."})
	assert.Equal(t, 0, code)
	assert.Contains(t, stdout.String(), "warning: logger.Infof called with no format args")
	assert.Empty(t, stderr.String())
}

func goModuleContentWithReplace(modulePath, dubboGoPath string) string {
	return fmt.Sprintf(`module %s

go 1.25.0

require github.com/dubbogo/gost v1.14.3
`, modulePath)
}

func writeTempFile(t *testing.T, dir, name, content string) {
	t.Helper()
	path := filepath.Join(dir, name)
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
	require.NoError(t, os.WriteFile(path, []byte(content), 0o644))
}

func tidyModule(t *testing.T, dir string) {
	t.Helper()
	cmd := exec.Command("go", "mod", "tidy")
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	require.NoError(t, err, "go mod tidy failed: %s", string(out))
}

func repoRoot(t *testing.T) string {
	t.Helper()
	_, filename, _, ok := runtime.Caller(0)
	require.True(t, ok)
	return filepath.Clean(filepath.Join(filepath.Dir(filename), "..", ".."))
}
