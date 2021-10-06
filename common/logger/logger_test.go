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

package logger

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
)

import (
	"github.com/stretchr/testify/assert"
)

func TestInitLog(t *testing.T) {
	var (
		err  error
		path string
	)

	err = InitLog("")
	assert.EqualError(t, err, "log configure file name is nil")

	path, err = filepath.Abs("./log.xml")
	assert.NoError(t, err)
	err = InitLog(path)
	assert.EqualError(t, err, "log configure file name{"+path+"} suffix must be .yml")

	path, err = filepath.Abs("./logger.yml")
	assert.NoError(t, err)
	err = InitLog(path)
	var errMsg string
	if runtime.GOOS == "windows" {
		errMsg = fmt.Sprintf("open %s: The system cannot find the file specified.", path)
	} else {
		errMsg = fmt.Sprintf("open %s: no such file or directory", path)
	}
	assert.EqualError(t, err, fmt.Sprintf("ioutil.ReadFile(file:%s) = error:%s", path, errMsg))

	err = InitLog("./log.yml")
	assert.NoError(t, err)

	Debug("debug")
	Info("info")
	Warn("warn")
	Error("error")
	Debugf("%s", "debug")
	Infof("%s", "info")
	Warnf("%s", "warn")
	Errorf("%s", "error")
}

func TestSetLevel(t *testing.T) {
	err := InitLog("./log.yml")
	assert.NoError(t, err)
	Debug("debug")
	Info("info")

	assert.True(t, SetLoggerLevel("info"))
	Debug("debug")
	Info("info")

	SetLogger(GetLogger().(*DubboLogger).Logger)
	assert.False(t, SetLoggerLevel("debug"))
	Debug("debug")
	Info("info")
}

func TestFatal(t *testing.T) {
	err := InitLog("./log.yml")
	assert.NoError(t, err)
	if os.Getenv("BE_Fatal") == "1" {
		Fatal("fool")
		return
	}
	cmd := exec.Command(os.Args[0], "-test.run=TestFatal")
	cmd.Env = append(os.Environ(), "BE_Fatal=1")
	err = cmd.Run()
	if e, ok := err.(*exec.ExitError); ok && !e.Success() {
		return
	}
	t.Fatalf("process ran with err %v, want exit status 1", err)
}

func TestFatalf(t *testing.T) {
	err := InitLog("./log.yml")
	assert.NoError(t, err)
	if os.Getenv("BE_Fatalf") == "1" {
		Fatalf("%s", "foolf")
		return
	}
	cmd := exec.Command(os.Args[0], "-test.run=TestFatalf")
	cmd.Env = append(os.Environ(), "BE_Fatalf=1")
	err = cmd.Run()
	if e, ok := err.(*exec.ExitError); ok && !e.Success() {
		return
	}
	t.Fatalf("process ran with err %v, want exit status 1", err)
}
