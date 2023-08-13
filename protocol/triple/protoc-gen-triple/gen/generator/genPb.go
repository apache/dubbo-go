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
	"fmt"
	"os"
	"path/filepath"
	"protoc-gen-triple/util"
)

func (g *Generator) GenPb() error {
	pwd, err := os.Getwd()
	if err != nil {
		return err
	}
	g.genPbCmd(pwd)
	output, err := util.Exec(g.ctx.ProtocCmd, pwd)
	if len(output) > 0 {
		fmt.Println(output)
	}
	return err
}

func (g *Generator) genPbCmd(goout string) {
	src := g.ctx.Src
	g.ctx.ProtocCmd = fmt.Sprintf("protoc %s -I=%s --go_out=%s", filepath.Base(src), filepath.Dir(src), goout)
}
