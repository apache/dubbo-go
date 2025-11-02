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

package dubbo

import (
	"os"
	"path/filepath"
	"testing"
)

func writeFile(t *testing.T, dir, name, content string) string {
	t.Helper()
	p := filepath.Join(dir, name)
	if err := os.WriteFile(p, []byte(content), 0o600); err != nil {
		t.Fatalf("write file: %v", err)
	}
	return p
}

func TestHotUpdateConfig_AllowsLoggerLevelChange(t *testing.T) {
	// snapshot globals we touch and restore afterwards
	prevIns := instanceOptions
	defer func() { instanceOptions = prevIns }()

	tmp := t.TempDir()
	base := "dubbo:\n  logger:\n    level: info\n"
	updated := "dubbo:\n  logger:\n    level: debug\n"

	path := writeFile(t, tmp, "conf.yaml", base)
	conf := NewLoaderConf(WithPath(path))

	// overwrite the file with the updated content
	if err := os.WriteFile(path, []byte(updated), 0o600); err != nil {
		t.Fatalf("overwrite file: %v", err)
	}

	if err := hotUpdateConfig(conf); err != nil {
		t.Fatalf("hotUpdateConfig unexpected error: %v", err)
	}
	if got := instanceOptions.Logger.Level; got != "debug" {
		t.Fatalf("logger level not updated, want=debug got=%s", got)
	}
}

func TestHotUpdateConfig_DeniesDisallowedChange(t *testing.T) {
	// snapshot globals we touch and restore afterwards
	prevIns := instanceOptions
	defer func() { instanceOptions = prevIns }()

	tmp := t.TempDir()
	base := "dubbo:\n  application:\n    name: app1\n"
	updated := "dubbo:\n  application:\n    name: app2\n"

	path := writeFile(t, tmp, "conf.yaml", base)
	conf := NewLoaderConf(WithPath(path))

	// ensure a known baseline value in globals
	instanceOptions.Application.Name = "baseline"

	if err := os.WriteFile(path, []byte(updated), 0o600); err != nil {
		t.Fatalf("overwrite file: %v", err)
	}

	if err := hotUpdateConfig(conf); err == nil {
		t.Fatalf("expected error for disallowed change, got nil")
	}
	if got := instanceOptions.Application.Name; got != "baseline" {
		t.Fatalf("instanceOptions changed unexpectedly, want=baseline got=%s", got)
	}
}

func TestHotUpdateConfig_AllowsWithCustomPrefix(t *testing.T) {
	// snapshot globals and hot-reload predicates
	prevIns := instanceOptions
	prevPreds := hotReloadAllowedPredicates
	defer func() { instanceOptions = prevIns; hotReloadAllowedPredicates = prevPreds }()

	tmp := t.TempDir()
	base := "dubbo:\n  application:\n    name: app1\n"
	updated := "dubbo:\n  application:\n    name: app2\n"

	path := writeFile(t, tmp, "conf.yaml", base)
	conf := NewLoaderConf(WithPath(path))

	// allow changing any key under dubbo.application.*
	AllowHotReloadPrefix("dubbo.application.")

	if err := os.WriteFile(path, []byte(updated), 0o600); err != nil {
		t.Fatalf("overwrite file: %v", err)
	}

	if err := hotUpdateConfig(conf); err != nil {
		t.Fatalf("hotUpdateConfig unexpected error with allowed prefix: %v", err)
	}
}
