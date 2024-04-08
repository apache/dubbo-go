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

package main

import (
	"fmt"
	"os"
)

import (
	"google.golang.org/protobuf/compiler/protogen"
	"google.golang.org/protobuf/types/pluginpb"
)

import (
	"dubbo.apache.org/dubbo-go/v3/cmd/protoc-gen-go-dubbo/gen/generator"
	"dubbo.apache.org/dubbo-go/v3/cmd/protoc-gen-go-dubbo/internal/version"
)

const (
	usage = ""
)

func main() {
	if len(os.Args) == 2 && os.Args[1] == "--version" {
		fmt.Fprintln(os.Stdout, version.Version)
		os.Exit(0)
	}
	if len(os.Args) == 2 && (os.Args[1] == "-h" || os.Args[1] == "--help") {
		fmt.Fprintln(os.Stdout, usage)
		os.Exit(0)
	}
	if len(os.Args) != 1 {
		fmt.Fprintln(os.Stderr, usage)
		os.Exit(1)
	}

	protogen.Options{}.Run(func(plugin *protogen.Plugin) error {
		plugin.SupportedFeatures = uint64(pluginpb.CodeGeneratorResponse_FEATURE_PROTO3_OPTIONAL)
		return genDubbo(plugin)
	})
}

func genDubbo(plugin *protogen.Plugin) error {
	for _, file := range plugin.Files {
		if file.Generate {
			dubboGo, err := generator.ProcessProtoFile(file.Proto)
			if err != nil {
				return err
			}
			return generator.GenDubboFile(dubboGo)
		}
	}
	return nil
}
