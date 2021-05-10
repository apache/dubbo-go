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
	"io/ioutil"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"runtime"
	"strings"
)

import (
	gxset "github.com/dubbogo/gost/container/set"
	perrors "github.com/pkg/errors"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/config_center"
	"dubbo.apache.org/dubbo-go/v3/config_center/parser"
)

var osType = runtime.GOOS

const (
	windowsOS = "windows"
)

const (
	ParamNamePrefix               = "dubbo.config-center."
	ConfigCenterDirParamName      = ParamNamePrefix + "dir"
	ConfigCenterEncodingParamName = ParamNamePrefix + "encoding"
	defaultConfigCenterEncoding   = "UTF-8"
)

// FileSystemDynamicConfiguration
type FileSystemDynamicConfiguration struct {
	config_center.BaseDynamicConfiguration
	url           *common.URL
	rootPath      string
	encoding      string
	cacheListener *CacheListener
	parser        parser.ConfigurationParser
}

func newFileSystemDynamicConfiguration(url *common.URL) (*FileSystemDynamicConfiguration, error) {
	encode := url.GetParam(ConfigCenterEncodingParamName, defaultConfigCenterEncoding)

	root := url.GetParam(ConfigCenterDirParamName, "")
	var c *FileSystemDynamicConfiguration

	root, err := mkdirIfNecessary(root)
	if err != nil {
		return nil, err
	}

	c = &FileSystemDynamicConfiguration{
		url:      url,
		rootPath: root,
		encoding: encode,
	}

	c.cacheListener = NewCacheListener(c.rootPath)

	return c, nil
}

// RootPath get root path
func (fsdc *FileSystemDynamicConfiguration) RootPath() string {
	return fsdc.rootPath
}

// Parser Get Parser
func (fsdc *FileSystemDynamicConfiguration) Parser() parser.ConfigurationParser {
	return fsdc.parser
}

// SetParser Set Parser
func (fsdc *FileSystemDynamicConfiguration) SetParser(p parser.ConfigurationParser) {
	fsdc.parser = p
}

// AddListener Add listener
func (fsdc *FileSystemDynamicConfiguration) AddListener(key string, listener config_center.ConfigurationListener,
	opts ...config_center.Option) {
	tmpOpts := &config_center.Options{}
	for _, opt := range opts {
		opt(tmpOpts)
	}

	tmpPath := fsdc.GetPath(key, tmpOpts.Group)
	fsdc.cacheListener.AddListener(tmpPath, listener)
}

// RemoveListener Remove listener
func (fsdc *FileSystemDynamicConfiguration) RemoveListener(key string, listener config_center.ConfigurationListener,
	opts ...config_center.Option) {
	tmpOpts := &config_center.Options{}
	for _, opt := range opts {
		opt(tmpOpts)
	}

	tmpPath := fsdc.GetPath(key, tmpOpts.Group)
	fsdc.cacheListener.RemoveListener(tmpPath, listener)
}

// GetProperties get properties file
func (fsdc *FileSystemDynamicConfiguration) GetProperties(key string, opts ...config_center.Option) (string, error) {
	tmpOpts := &config_center.Options{}
	for _, opt := range opts {
		opt(tmpOpts)
	}

	tmpPath := fsdc.GetPath(key, tmpOpts.Group)
	file, err := ioutil.ReadFile(tmpPath)
	if err != nil {
		return "", perrors.WithStack(err)
	}
	return string(file), nil
}

// GetRule get Router rule properties file
func (fsdc *FileSystemDynamicConfiguration) GetRule(key string, opts ...config_center.Option) (string, error) {
	return fsdc.GetProperties(key, opts...)
}

// GetInternalProperty get value by key in Default properties file(dubbo.properties)
func (fsdc *FileSystemDynamicConfiguration) GetInternalProperty(key string, opts ...config_center.Option) (string,
	error) {
	return fsdc.GetProperties(key, opts...)
}

// PublishConfig will publish the config with the (key, group, value) pair
func (fsdc *FileSystemDynamicConfiguration) PublishConfig(key string, group string, value string) error {
	tmpPath := fsdc.GetPath(key, group)
	return fsdc.write2File(tmpPath, value)
}

// GetConfigKeysByGroup will return all keys with the group
func (fsdc *FileSystemDynamicConfiguration) GetConfigKeysByGroup(group string) (*gxset.HashSet, error) {
	tmpPath := fsdc.GetPath("", group)
	r := gxset.NewSet()

	fileInfo, _ := ioutil.ReadDir(tmpPath)

	for _, file := range fileInfo {
		// list file
		if file.IsDir() {
			continue
		}

		r.Add(file.Name())
	}

	return r, nil
}

// RemoveConfig will remove tconfig_center/nacos/impl_testhe config whit hte (key, group)
func (fsdc *FileSystemDynamicConfiguration) RemoveConfig(key string, group string) error {
	tmpPath := fsdc.GetPath(key, group)
	_, err := fsdc.deleteDelay(tmpPath)
	return err
}

// Close close file watcher
func (fsdc *FileSystemDynamicConfiguration) Close() error {
	return fsdc.cacheListener.Close()
}

// GetPath get path
func (fsdc *FileSystemDynamicConfiguration) GetPath(key string, group string) string {
	if len(key) == 0 {
		return filepath.Join(fsdc.rootPath, group)
	}

	if len(group) == 0 {
		group = config_center.DEFAULT_GROUP
	}

	return filepath.Join(fsdc.rootPath, group, adapterKey(key))
}

func (fsdc *FileSystemDynamicConfiguration) deleteDelay(path string) (bool, error) {
	if len(path) == 0 {
		return false, nil
	}

	if err := os.RemoveAll(path); err != nil {
		return false, err
	}

	return true, nil
}

func (fsdc *FileSystemDynamicConfiguration) write2File(fp string, value string) error {
	if err := forceMkdirParent(fp); err != nil {
		return perrors.WithStack(err)
	}

	return ioutil.WriteFile(fp, []byte(value), os.ModePerm)
}

func forceMkdirParent(fp string) error {
	return createDir(getParentDirectory(fp))
}

func createDir(path string) error {
	// create dir, chmod is drwxrwxrwx(0777)
	if err := os.MkdirAll(path, os.ModePerm); err != nil {
		return err
	}

	return nil
}

func getParentDirectory(fp string) string {
	return substr(fp, 0, strings.LastIndex(fp, string(filepath.Separator)))
}

func substr(s string, pos, length int) string {
	runes := []rune(s)
	l := pos + length
	if l > len(runes) {
		l = len(runes)
	}

	return string(runes[pos:l])
}

// Home returns the home directory for the executing user.
//
// This uses an OS-specific method for discovering the home directory.
// An error is returned if a home directory cannot be detected.
func Home() (string, error) {
	currentUser, err := user.Current()
	if nil == err {
		return currentUser.HomeDir, nil
	}

	// cross compile support
	if windowsOS == osType {
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
	if len(result) == 0 {
		return "", errors.New("blank output when reading home directory")
	}

	return result, nil
}

func homeWindows() (string, error) {
	drive := os.Getenv("HOMEDRIVE")
	homePath := os.Getenv("HOMEPATH")
	home := drive + homePath
	if len(drive) == 0 || len(homePath) == 0 {
		home = os.Getenv("USERPROFILE")
	}
	if len(home) == 0 {
		return "", errors.New("HOMEDRIVE, HOMEPATH, and USERPROFILE are blank")
	}

	return home, nil
}

func mkdirIfNecessary(urlRoot string) (string, error) {
	if !legalPath(urlRoot) {
		// not exist, use default, mac is: /XXX/xx/.dubbo/config-center
		rp, err := Home()
		if err != nil {
			return "", perrors.WithStack(err)
		}

		urlRoot = adapterUrl(rp)
	}

	if _, err := os.Stat(urlRoot); err != nil {
		// it must be dir, if not exist, will create
		if err = createDir(urlRoot); err != nil {
			return "", perrors.WithStack(err)
		}
	}

	return urlRoot, nil
}

func legalPath(path string) bool {
	if len(path) == 0 {
		return false
	}
	if _, err := os.Stat(path); err != nil {
		return false
	}

	return true
}

func adapterUrl(rp string) string {
	if osType == windowsOS {
		return filepath.Join(rp, "_dubbo", "config-center")
	}

	return filepath.Join(rp, ".dubbo", "config-center")
}

// used for GetPath. param key default is instance's id.
// e.g: (ip:port) 127.0.0.1:20081, in windows env, will change to 127_0_0_1_20081
func adapterKey(key string) string {
	if len(key) == 0 {
		return ""
	}

	if osType == windowsOS {
		return strings.ReplaceAll(strings.ReplaceAll(key, ".", "_"), ":", "_")
	}

	return key
}
