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
	"flag"
	"fmt"
	"os"
)

import (
	"google.golang.org/protobuf/compiler/protogen"

	"google.golang.org/protobuf/types/pluginpb"
)

import (
	"dubbo.apache.org/dubbo-go/v3/triple-tool/gen/generator"
	"dubbo.apache.org/dubbo-go/v3/triple-tool/internal/old_triple"
	"dubbo.apache.org/dubbo-go/v3/triple-tool/internal/version"
)

const (
	usage = "See https://connect.build/docs/go/getting-started to learn how to use this plugin.\n\nFlags:\n  -h, --help\tPrint this help and exit.\n      --version\tPrint the version and exit."
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

	var flags flag.FlagSet
	useOld := flags.Bool("useOldVersion", false, "print the version and exit")
	old_triple.RequireUnimplemented = flags.Bool("require_unimplemented_servers", true, "set to false to match legacy behavior")

	protogen.Options{
		ParamFunc: flags.Set,
	}.Run(
		func(plugin *protogen.Plugin) error {
			plugin.SupportedFeatures = uint64(pluginpb.CodeGeneratorResponse_FEATURE_PROTO3_OPTIONAL)
			if *useOld {
				return genOldTriple(plugin)
			}
			return genTriple(plugin)
		},
	)
}

func genTriple(plugin *protogen.Plugin) error {
	for _, file := range plugin.Files {
		if file.Generate {
			tripleGo, err := generator.ProcessProtoFile(file.Proto)
			if err != nil {
				return err
			}
			return generator.GenTripleFile(tripleGo)
		}
	}
	return nil
}

func genOldTriple(plugin *protogen.Plugin) error {
	for _, file := range plugin.Files {
		if file.Generate {
			old_triple.GenerateFile(plugin, file)
		}
	}
	return nil
}
