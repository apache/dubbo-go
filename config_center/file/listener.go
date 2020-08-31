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
	"bytes"
	"errors"
	"os"
	"os/exec"
	"os/user"
	"runtime"
	"strings"
	"sync"
)

import (
	"github.com/fsnotify/fsnotify"
)

import (
	"github.com/apache/dubbo-go/common/logger"
	"github.com/apache/dubbo-go/config_center"
)

var (
	watchStartOnce = sync.Once{}
	watch          *fsnotify.Watcher
	callback       map[string]config_center.ConfigurationListener
	//path           = strings.Join([]string{h, ".dubbo", "registry"}, string(filepath.Separator))
)

func init() {
	watch, err := fsnotify.NewWatcher()
	if err != nil {
		logger.Errorf("file : listen config fail, error:%v ", err)
		return
	}

	watchStartOnce.Do(func() {
		go func() {
			for {
				select {
				case event := <-watch.Events:
					if event.Op&fsnotify.Write == fsnotify.Write {

					}
					if event.Op&fsnotify.Create == fsnotify.Create {

					}
					if event.Op&fsnotify.Remove == fsnotify.Remove {

					}
				case err := <-watch.Errors:
					logger.Errorf("file : listen watch fail:", err)
				}
			}
		}()
	})
}

func (fsdc *fileSystemDynamicConfiguration) addListener(key string, listener config_center.ConfigurationListener) {
	path := fsdc.buildPath(key, config_center.DEFAULT_GROUP)
	_, loaded := fsdc.keyListeners.Load(path)
	if !loaded {
		if err := watch.Add(path); err != nil {
			logger.Errorf("file : listen watch: %s add fail:", key, err)
		} else {
			fsdc.keyListeners.Store(key, listener)
		}
	} else {
		logger.Infof("profile:%s. this profile is already listening", key)
	}
}

func (fsdc *fileSystemDynamicConfiguration) removeListener(key string, listener config_center.ConfigurationListener) {
	path := fsdc.buildPath(key, config_center.DEFAULT_GROUP)
	_, loaded := fsdc.keyListeners.Load(path)
	if !loaded {
		if err := watch.Remove(path); err != nil {
			logger.Errorf("file : listen watch: %s remove fail:", key, err)
		} else {
			fsdc.keyListeners.Delete(path)
		}
	} else {
		logger.Infof("profile:%s. this profile is not exist", key)
	}
}

func (fsdc *fileSystemDynamicConfiguration) close() {
	watch.Close()
}

// Home returns the home directory for the executing user.
//
// This uses an OS-specific method for discovering the home directory.
// An error is returned if a home directory cannot be detected.
func Home() (string, error) {
	user, err := user.Current()
	if nil == err {
		return user.HomeDir, nil
	}

	// cross compile support
	if "windows" == runtime.GOOS {
		return homeWindows()
	}

	// Unix-like system, so just assume Unix
	return homeUnix()
}

func homeUnix() (string, error) {
	// First prefer the HOME environmental variable
	if home := os.Getenv("HOME"); home != "" {
		return home, nil
	}

	// If that fails, try the shell
	var stdout bytes.Buffer
	cmd := exec.Command("sh", "-c", "eval echo ~$USER")
	cmd.Stdout = &stdout
	if err := cmd.Run(); err != nil {
		return "", err
	}

	result := strings.TrimSpace(stdout.String())
	if result == "" {
		return "", errors.New("blank output when reading home directory")
	}

	return result, nil
}

func homeWindows() (string, error) {
	drive := os.Getenv("HOMEDRIVE")
	path := os.Getenv("HOMEPATH")
	home := drive + path
	if drive == "" || path == "" {
		home = os.Getenv("USERPROFILE")
	}
	if home == "" {
		return "", errors.New("HOMEDRIVE, HOMEPATH, and USERPROFILE are blank")
	}

	return home, nil
}
