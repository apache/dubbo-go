//go:build linux
// +build linux

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

package file

import (
	"os"
	"os/exec"
	"syscall"
	"testing"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/extension"
)

const watcherInitFailureChildEnv = "DUBBO_GO_FILE_WATCHER_INIT_FAILURE_CHILD"

func TestCacheListenerWatcherInitFailure(t *testing.T) {
	if os.Getenv(watcherInitFailureChildEnv) == "1" {
		testCacheListenerWatcherInitFailure(t)
		return
	}

	cmd := exec.Command(os.Args[0], "-test.run", "^TestCacheListenerWatcherInitFailure$")
	cmd.Env = append(os.Environ(), watcherInitFailureChildEnv+"=1")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("watcher initialization child failed: %v\n%s", err, output)
	}
}

func testCacheListenerWatcherInitFailure(t *testing.T) {
	rootPath := t.TempDir()
	url, err := common.NewURL("registry://127.0.0.1:2181")
	if err != nil {
		t.Fatalf("create URL error %v", err)
	}
	url.SetParam(ConfigCenterDirParamName, rootPath)

	var limit syscall.Rlimit
	if err := syscall.Getrlimit(syscall.RLIMIT_NOFILE, &limit); err != nil {
		t.Fatalf("get file descriptor limit error %v", err)
	}
	if limit.Max < 3 {
		t.Skipf("file descriptor hard limit is too low: %d", limit.Max)
	}
	originalLimit := limit
	limit.Cur = 3
	if err := syscall.Setrlimit(syscall.RLIMIT_NOFILE, &limit); err != nil {
		t.Fatalf("set file descriptor limit error %v", err)
	}
	defer func() {
		if err := syscall.Setrlimit(syscall.RLIMIT_NOFILE, &originalLimit); err != nil {
			t.Errorf("restore file descriptor limit error %v", err)
		}
	}()

	callbacksBefore := extension.GetAllCustomShutdownCallbacks().Len()
	listener, err := NewCacheListener(rootPath)
	if err == nil {
		t.Fatal("expected watcher initialization to fail")
	}
	if listener != nil {
		t.Fatal("expected nil listener when watcher initialization fails")
	}

	factory, err := extension.GetConfigCenterFactory("file")
	if err != nil {
		t.Fatalf("get file config center factory error %v", err)
	}
	dynamicConfig, err := factory.GetDynamicConfiguration(url)
	if err == nil {
		t.Fatal("expected file config center initialization to fail")
	}
	if dynamicConfig != nil {
		t.Fatal("expected nil dynamic configuration when watcher initialization fails")
	}

	callbacksAfter := extension.GetAllCustomShutdownCallbacks().Len()
	if callbacksAfter != callbacksBefore {
		t.Fatalf("watcher initialization failure registered shutdown callback: before=%d after=%d", callbacksBefore, callbacksAfter)
	}
}
