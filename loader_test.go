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
	"sync"
	"testing"
	"time"
)

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common/extension"
)

type loaderYAMLConfig struct {
	prefix      string
	Value       int `yaml:"value"`
	initialized extension.Scope
	onInit      func(*loaderYAMLConfig)
}

func (c *loaderYAMLConfig) Prefix() string {
	return c.prefix
}

func (c *loaderYAMLConfig) New() extension.Config {
	return &loaderYAMLConfig{
		prefix: c.prefix,
		onInit: c.onInit,
	}
}

func (c *loaderYAMLConfig) Init(scope extension.Scope) error {
	c.initialized = scope
	if c.onInit != nil {
		c.onInit(c)
	}
	return nil
}

func (c *loaderYAMLConfig) FilterNames(extension.Scope) []string {
	return nil
}

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

func TestHotUpdateConfig_DeniesExtensionChangeEvenWhenBroadlyAllowed(t *testing.T) {
	prevIns := instanceOptions
	prevPreds := hotReloadAllowedPredicates
	t.Cleanup(func() {
		instanceOptions = prevIns
		hotReloadAllowedPredicates = prevPreds
	})

	tmp := t.TempDir()
	base := "dubbo:\n  extensions:\n    demo:\n      value: 1\n"
	updated := "dubbo:\n  extensions:\n    demo:\n      value: 2\n"

	path := writeFile(t, tmp, "conf.yaml", base)
	conf := NewLoaderConf(WithPath(path))
	AllowHotReloadPrefix("dubbo.")

	if err := os.WriteFile(path, []byte(updated), 0o600); err != nil {
		t.Fatalf("overwrite file: %v", err)
	}

	err := hotUpdateConfig(conf)
	require.EqualError(t, err, "hot reload denied: extension configuration changes require restart")
	require.Same(t, prevIns, instanceOptions)
}

func TestLoadConfigInitializesClientExtensionFromYAML(t *testing.T) {
	const prefix = "loader-yaml-extension"
	extension.UnregisterConfig(prefix)
	t.Cleanup(func() { extension.UnregisterConfig(prefix) })

	var initialized *loaderYAMLConfig
	require.NoError(t, extension.RegisterConfig(&loaderYAMLConfig{
		prefix: prefix,
		onInit: func(config *loaderYAMLConfig) {
			initialized = config
		},
	}))

	conf := NewLoaderConf(WithBytes([]byte(`dubbo:
  extensions:
    loader-yaml-extension:
      consumer:
        value: 7
`)))
	loadedOptions, err := loadInstanceOptions(conf)
	require.NoError(t, err)
	require.NoError(t, loadedOptions.init())

	instance := &Instance{insOpts: loadedOptions}
	_, err = instance.NewClient()
	require.NoError(t, err)
	require.NotNil(t, initialized)
	assert.Equal(t, 7, initialized.Value)
	assert.Equal(t, extension.ClientScope, initialized.initialized)
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

func TestExtensionConfigsFromKoanfPreservesDottedKeys(t *testing.T) {
	conf := NewLoaderConf(WithBytes([]byte(`dubbo:
  extensions:
    dotted:
      consumer:
        greet.GreetService:::Greet:
          timeout: 1000
`)))
	configs := extensionConfigsFromKoanf(GetConfigResolver(conf))
	require.NotNil(t, configs)
	dotted, ok := configs["dotted"].(map[string]any)
	require.True(t, ok)
	consumer, ok := dotted["consumer"].(map[string]any)
	require.True(t, ok)
	require.Contains(t, consumer, "greet.GreetService:::Greet")
}

// TestGoSafely_RunsAndRecoversPanic is a regression test for the inline
// replacement of gost/runtime.GoSafely in loader.go (ported from the gost
// TestGoSafe semantics): fn must run, and a panic raised inside fn must be
// recovered so that wg.Go still completes and the process is not crashed.
func TestGoSafely_RunsAndRecoversPanic(t *testing.T) {
	t.Run("normal handler runs and completes", func(t *testing.T) {
		ran := make(chan struct{})
		var wg sync.WaitGroup
		goSafely(&wg, func() { close(ran) })
		select {
		case <-ran:
		case <-time.After(time.Second):
			t.Fatal("goSafely did not run the handler")
		}
		wg.Wait()
	})

	t.Run("panicking handler is contained and completes", func(t *testing.T) {
		started := make(chan struct{})
		var wg sync.WaitGroup
		goSafely(&wg, func() {
			close(started)
			panic("boom")
		})
		select {
		case <-started:
		case <-time.After(time.Second):
			t.Fatal("goSafely did not start the panicking handler")
		}
		// Must not hang: the recover inside goSafely ensures the goroutine
		// finishes and wg reaches zero even though the handler panicked.
		wg.Wait()
	})
}
