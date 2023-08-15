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

package generator

import (
	"os"
	"path/filepath"
)

import (
	"github.com/spf13/cobra"
)

import (
	"dubbo.apache.org/dubbo-go/v3/protoc-gen-triple/util"
)

type Context struct {
	Src          string
	ProtocCmd    string
	GoOpts       []string
	GoOut        string
	GoModuleName string
	Pwd          string
}

func newContext(cmd *cobra.Command, args []string) (Context, error) {
	var ctx Context
	pwd, err := os.Getwd()
	if err != nil {
		return ctx, err
	}
	ctx.Pwd = pwd
	src, err := filepath.Abs(ProtocPath)
	if err != nil {
		return ctx, err
	}
	ctx.Src = src
	ctx.GoOut = filepath.Dir(src)
	module, err := util.GetModuleName()
	if err != nil {
		return ctx, err
	}
	ctx.GoModuleName = module
	ctx.GoOpts = GoOpts
	return ctx, nil
}

func Generate(cmd *cobra.Command, args []string) error {
	ctx, err := newContext(cmd, args)
	if err != nil {
		return err
	}
	return NewGenerator(ctx).gen()
}
