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

package filesystem

import (
	perrors "github.com/pkg/errors"
	"os"
)

type File struct {
	Path         string   //路径
	fileWriter   *os.File //写入文件
}

// 创建文件锁，配合 defer f.Release() 来使用
func (file *File)Create() (f *Filelock, e error) {
	if file == nil || file.Path == "" {
		return nil, perrors.Errorf("cannot create flock on empty path")
	}
	lock, e := os.Create(file.Path)
	if e != nil {
		return
	}
	return &Filelock{
		LockFile: file,
		lock:     lock,
	}, nil
}
