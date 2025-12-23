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

// protoc-gen-go-triple is a plugin for the Google protocol buffer compiler to
// generate Go code.
//
// Check readme for how to use it.
package main

import (
	"flag"
	"fmt"
	"os"
	"strings"
)

import (
	"google.golang.org/protobuf/compiler/protogen"

	"google.golang.org/protobuf/types/pluginpb"
)

import (
	"dubbo.apache.org/dubbo-go/v3/tools/protoc-gen-go-triple/gen/generator"
	"dubbo.apache.org/dubbo-go/v3/tools/protoc-gen-go-triple/internal/old_triple"
	"dubbo.apache.org/dubbo-go/v3/tools/protoc-gen-go-triple/internal/version"
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
	var errors []error

	for _, file := range plugin.Files {
		if !file.Generate {
			continue
		}

		// Skip proto files without services.
		if len(file.Proto.GetService()) == 0 {
			continue
		}

		filename := file.GeneratedFilenamePrefix + ".triple.go"
		// Use the same import path as the pb.go file to ensure they're in the same package
		// Extract the package name from the go_package option
		// Use the import path as parsed by protogen to avoid edge cases.
		g := plugin.NewGeneratedFile(filename, file.GoImportPath)
		// import Dubbo's libraries
		g.QualifiedGoIdent(protogen.GoImportPath("dubbo.apache.org/dubbo-go/v3/client").Ident("client"))
		g.QualifiedGoIdent(protogen.GoImportPath("dubbo.apache.org/dubbo-go/v3/common").Ident("common"))
		g.QualifiedGoIdent(protogen.GoImportPath("dubbo.apache.org/dubbo-go/v3/common/constant").Ident("constant"))
		g.QualifiedGoIdent(protogen.GoImportPath("dubbo.apache.org/dubbo-go/v3/protocol/triple/triple_protocol").Ident("triple_protocol"))
		g.QualifiedGoIdent(protogen.GoImportPath("dubbo.apache.org/dubbo-go/v3/server").Ident("server"))
		tripleGo, err := generator.ProcessProtoFile(g, file)
		if err != nil {
			errors = append(errors, fmt.Errorf("processing %s: %w", file.Desc.Path(), err))
			continue
		}
		// Ensure the generated file uses the exact Go package name computed by protoc-gen-go.
		tripleGo.Package = string(file.GoPackageName)

		err = generator.GenTripleFile(g, tripleGo)
		if err != nil {
			errors = append(errors, fmt.Errorf("generating %s: %w", filename, err))
		}
	}
	if len(errors) > 0 {
		var errorMessages []string
		for _, err := range errors {
			errorMessages = append(errorMessages, err.Error())
		}
		return fmt.Errorf("multiple errors occurred:\n%s", strings.Join(errorMessages, "\n"))
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
