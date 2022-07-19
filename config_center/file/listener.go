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
	"io/ioutil"
	"sync"
)

import (
	"github.com/dubbogo/gost/log/logger"

	"github.com/fsnotify/fsnotify"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common/extension"
	"dubbo.apache.org/dubbo-go/v3/config_center"
	"dubbo.apache.org/dubbo-go/v3/remoting"
)

// CacheListener is file watcher
type CacheListener struct {
	watch        *fsnotify.Watcher
	keyListeners sync.Map
	rootPath     string
}

// NewCacheListener creates a new CacheListener
func NewCacheListener(rootPath string) *CacheListener {
	cl := &CacheListener{rootPath: rootPath}
	// start watcher
	watch, err := fsnotify.NewWatcher()
	if err != nil {
		logger.Errorf("file : listen config fail, error:%v ", err)
	}
	go func() {
		for {
			select {
			case event := <-watch.Events:
				key := event.Name
				logger.Debugf("watcher %s, event %v", cl.rootPath, event)
				if event.Op&fsnotify.Write == fsnotify.Write {
					if l, ok := cl.keyListeners.Load(key); ok {
						dataChangeCallback(l.(map[config_center.ConfigurationListener]struct{}), key,
							remoting.EventTypeUpdate)
					}
				}
				if event.Op&fsnotify.Create == fsnotify.Create {
					if l, ok := cl.keyListeners.Load(key); ok {
						dataChangeCallback(l.(map[config_center.ConfigurationListener]struct{}), key,
							remoting.EventTypeAdd)
					}
				}
				if event.Op&fsnotify.Remove == fsnotify.Remove {
					if l, ok := cl.keyListeners.Load(key); ok {
						removeCallback(l.(map[config_center.ConfigurationListener]struct{}), key, remoting.EventTypeDel)
					}
				}
			case err := <-watch.Errors:
				// err may be nil, ignore
				if err != nil {
					logger.Warnf("file : listen watch fail:%+v", err)
				}
			}
		}
	}()
	cl.watch = watch

	extension.AddCustomShutdownCallback(func() {
		cl.watch.Close()
	})

	return cl
}

func removeCallback(lmap map[config_center.ConfigurationListener]struct{}, key string, event remoting.EventType) {
	if len(lmap) == 0 {
		logger.Warnf("file watch callback but configuration listener is empty, key:%s, event:%v", key, event)
		return
	}
	for l := range lmap {
		callback(l, key, "", event)
	}
}

func dataChangeCallback(lmap map[config_center.ConfigurationListener]struct{}, key string, event remoting.EventType) {
	if len(lmap) == 0 {
		logger.Warnf("file watch callback but configuration listener is empty, key:%s, event:%v", key, event)
		return
	}
	c := getFileContent(key)
	for l := range lmap {
		callback(l, key, c, event)
	}
}

func callback(listener config_center.ConfigurationListener, path, data string, event remoting.EventType) {
	listener.Process(&config_center.ConfigChangeEvent{Key: path, Value: data, ConfigType: event})
}

// Close will remove key listener and close watcher
func (cl *CacheListener) Close() error {
	cl.keyListeners.Range(func(key, value interface{}) bool {
		cl.keyListeners.Delete(key)
		return true
	})
	return cl.watch.Close()
}

// AddListener will add a listener if loaded
// if you watcher a file or directory not exist, will error with no such file or directory
func (cl *CacheListener) AddListener(key string, listener config_center.ConfigurationListener) {
	// reference from https://stackoverflow.com/questions/34018908/golang-why-dont-we-have-a-set-datastructure
	// make a map[your type]struct{} like set in java
	listeners, loaded := cl.keyListeners.LoadOrStore(key, map[config_center.ConfigurationListener]struct{}{
		listener: {},
	})
	if loaded {
		listeners.(map[config_center.ConfigurationListener]struct{})[listener] = struct{}{}
		cl.keyListeners.Store(key, listeners)
		return
	}
	if err := cl.watch.Add(key); err != nil {
		logger.Errorf("watcher add path:%s err:%v", key, err)
	}
}

// RemoveListener will delete a listener if loaded
func (cl *CacheListener) RemoveListener(key string, listener config_center.ConfigurationListener) {
	listeners, loaded := cl.keyListeners.Load(key)
	if !loaded {
		return
	}
	delete(listeners.(map[config_center.ConfigurationListener]struct{}), listener)
	if err := cl.watch.Remove(key); err != nil {
		logger.Errorf("watcher remove path:%s err:%v", key, err)
	}
}

func getFileContent(path string) string {
	c, err := ioutil.ReadFile(path)
	if err != nil {
		logger.Errorf("read file path:%s err:%v", path, err)
		return ""
	}

	return string(c)
}
