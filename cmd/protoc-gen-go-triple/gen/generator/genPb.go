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
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

import (
	"dubbo.apache.org/dubbo-go/v3/cmd/protoc-gen-go-triple/util"
)

func (g *Generator) GenPb() error {
	pwd := g.ctx.ModuleDir
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

	// Check if we need to add the --go_opt=module parameter
	if g.isGoPackageFull() {
		g.ctx.ProtocCmd += fmt.Sprintf(" --go_opt=module=%s", g.ctx.GoModuleName)
	}

	if len(g.ctx.GoOpts) > 0 {
		g.ctx.ProtocCmd += " --go_opt=" + strings.Join(g.ctx.GoOpts, ",")
	}
}

// Check if go_package is a full package path
func (g *Generator) isGoPackageFull() bool {
	// Parse go_package from the proto file
	goPackage := g.parseGoPackageFromProto()

	// Check if goPackage is a full package path, e.g., starts with "github.com/"
	return strings.HasPrefix(goPackage, g.ctx.GoModuleName)
}

// Parse go_package from the proto file
func (g *Generator) parseGoPackageFromProto() string {
	protoFileContent := g.loadProtoFileContent() // Load the content of the proto file

	// Use regular expression to match go_package in the proto file
	goPackageRegex := regexp.MustCompile(`go_package\s*=\s*"([^"]+)"`)
	match := goPackageRegex.FindStringSubmatch(protoFileContent)

	if len(match) > 1 {
		return match[1] // Return the matched go_package
	}

	return "" // Return an empty string if go_package is not matched
}

// Load the content of the proto file
func (g *Generator) loadProtoFileContent() string {
	protoFilePath := g.ctx.Src
	content, err := os.ReadFile(protoFilePath)
	if err != nil {
		log.Fatalf("Failed to read proto file: %v", err)
	}
	return string(content)
}
