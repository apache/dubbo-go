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

package accesslog

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"
)

func TestOpenLogFileClosesRotatedHandle(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("requires Linux /proc fd accounting")
	}

	accessLog := filepath.Join(t.TempDir(), "access.log")
	if err := os.WriteFile(accessLog, []byte("old log"), LogFileMode); err != nil {
		t.Fatal(err)
	}
	filter := &Filter{}

	before := countOpenFiles(t)
	for range 128 {
		staleTime := time.Now().Add(-48 * time.Hour)
		if err := os.Chtimes(accessLog, staleTime, staleTime); err != nil {
			t.Fatal(err)
		}
		logFile, err := filter.openLogFile(accessLog)
		if err != nil {
			t.Fatal(err)
		}
		if err := logFile.Close(); err != nil {
			t.Fatal(err)
		}
	}
	after := countOpenFiles(t)
	if after > before+2 {
		t.Fatalf("open file descriptors grew from %d to %d", before, after)
	}
}

func TestOpenLogFileClosesHandleOnRenameError(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("requires Linux file permission behavior")
	}

	dir := t.TempDir()
	accessLog := filepath.Join(dir, "access.log")
	if err := os.WriteFile(accessLog, []byte("old log"), LogFileMode); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(dir, 0o500); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(dir, 0o700) })

	filter := &Filter{}
	before := countOpenFiles(t)
	for range 128 {
		staleTime := time.Now().Add(-48 * time.Hour)
		if err := os.Chtimes(accessLog, staleTime, staleTime); err != nil {
			t.Fatal(err)
		}
		logFile, err := filter.openLogFile(accessLog)
		if err == nil {
			_ = logFile.Close()
			t.Skip("environment permits rename in a read-only directory")
		}
	}
	after := countOpenFiles(t)
	if after > before+2 {
		t.Fatalf("open file descriptors grew from %d to %d", before, after)
	}
}

func TestGetOrOpenLogFileClosesCachedFileOnOpenError(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("requires Linux file descriptor behavior")
	}

	dir := t.TempDir()
	accessLog := filepath.Join(dir, "access.log")
	oldFile, err := os.OpenFile(accessLog, os.O_CREATE|os.O_APPEND|os.O_RDWR, LogFileMode)
	if err != nil {
		t.Fatal(err)
	}
	staleTime := time.Now().Add(-48 * time.Hour)
	err = os.Chtimes(accessLog, staleTime, staleTime)
	if err != nil {
		_ = oldFile.Close()
		t.Fatal(err)
	}
	err = os.Remove(accessLog)
	if err != nil {
		_ = oldFile.Close()
		t.Fatal(err)
	}
	err = os.Mkdir(accessLog, 0o700)
	if err != nil {
		_ = oldFile.Close()
		t.Fatal(err)
	}

	filter := &Filter{fileCache: map[string]*os.File{accessLog: oldFile}}
	_, err = filter.getOrOpenLogFile(accessLog)
	if err == nil {
		t.Fatal("getOrOpenLogFile should return an error when OpenFile targets a directory")
	}
	if closeErr := oldFile.Close(); closeErr == nil {
		t.Fatal("cached file was not closed before OpenFile failed")
	}
}

func countOpenFiles(t *testing.T) int {
	t.Helper()
	entries, err := os.ReadDir("/proc/self/fd")
	if err != nil {
		t.Fatal(err)
	}
	return len(entries)
}
