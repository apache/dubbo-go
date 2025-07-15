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
	plugin "github.com/golang/protobuf/protoc-gen-go/plugin"

	"google.golang.org/protobuf/proto"
)

import (
	"dubbo.apache.org/dubbo-go/v3/tools/protoc-gen-triple-openapi/internal/converter"
)

func main() {
	showVersion := flag.Bool("version", false, "print the version and exit")

	flag.Parse()

	if *showVersion {
		fmt.Printf("protoc-gen-triple-openapi %s\n", version)
		return
	}

	resp, err := converter.ConvertFrom(os.Stdin)
	if err != nil {
		panic(err)
	}

	renderResponse(resp)
}

func renderResponse(resp *plugin.CodeGeneratorResponse) {
	data, err := proto.Marshal(resp)
	if err != nil {
		panic(err)
	}

	_, err = os.Stdout.Write(data)
	if err != nil {
		panic(err)
	}
}
