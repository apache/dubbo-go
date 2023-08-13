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
	"bytes"
	"github.com/emicklei/proto"
	"os"
	"testing"
)

func init() {
	protoFile := bytes.NewBufferString(testProto)
	parser := proto.NewParser(protoFile)
	p, _ := parser.Parse()

	g := NewGenerator(Context{
		GoModuleName: "protoc-gen-triple",
		Src:          "../../.",
	})
	data, _ = g.parseProtoToTriple(p)
}

var data TripleGo

func TestPreamble(t *testing.T) {
	err := TplPreamble.Execute(os.Stdout, data)
	if err != nil {
		panic(err)
	}
}

func TestPackage(t *testing.T) {
	err := TplPackage.Execute(os.Stdout, data)
	if err != nil {
		panic(err)
	}
}

func TestImport(t *testing.T) {
	err := TplImport.Execute(os.Stdout, data)
	if err != nil {
		panic(err)
	}
}

func TestTotalTpl(t *testing.T) {
	err := TplTotal.Execute(os.Stdout, data)
	if err != nil {
		panic(err)
	}
}

func TestClientInterfaceTemplate(t *testing.T) {
	err := TplClientInterface.Execute(os.Stdout, data)
	if err != nil {
		t.Fatalf("Failed to execute template: %v", err)
	}
}

func TestClientInterfaceImplTpl(t *testing.T) {
	err := TplClientInterfaceImpl.Execute(os.Stdout, data)
	if err != nil {
		t.Fatalf("Failed to execute template: %v", err)
	}
}

func TestClientImplTpl(t *testing.T) {
	err := TplClientImpl.Execute(os.Stdout, data)
	if err != nil {
		t.Fatalf("Failed to execute template: %v", err)
	}
}

func TestMethodInfoTpl(t *testing.T) {
	err := TplMethodInfo.Execute(os.Stdout, data)
	if err != nil {
		t.Fatalf("Failed to execute template: %v", err)
	}
}

func TestHandlerTpl(t *testing.T) {
	err := TplHandler.Execute(os.Stdout, data)
	if err != nil {
		t.Fatalf("Failed to execute template: %v", err)
	}
}

func TestServerImplTpl(t *testing.T) {
	err := TplServerImpl.Execute(os.Stdout, data)
	if err != nil {
		t.Fatalf("Failed to execute template: %v", err)
	}
}

func TestServiceInfoImplTpl(t *testing.T) {
	err := TplServerInfo.Execute(os.Stdout, data)
	if err != nil {
		t.Fatalf("Failed to execute template: %v", err)
	}
}

func TestUnImplTpl(t *testing.T) {
	err := TplUnImpl.Execute(os.Stdout, data)
	if err != nil {
		t.Fatalf("Failed to execute template: %v", err)
	}
}

func TestAll(t *testing.T) {
	for _, tpl := range Tpls {
		err := tpl.Execute(os.Stdout, data)
		if err != nil {
			t.Fatalf("Failed to execute template: %v", err)
		}
	}
}

const testProto = `syntax = "proto3";

package greet;

option go_package = "/proto;proto";

message GreetRequest {
  string name = 1;
}

message GreetResponse {
  string greeting = 1;
}

message GreetStreamRequest {
  string name = 1;
}

message GreetStreamResponse {
  string greeting = 1;
}

message GreetClientStreamRequest {
  string name = 1;
}

message GreetClientStreamResponse {
  string greeting = 1;
}

message GreetServerStreamRequest {
  string name = 1;
}

message GreetServerStreamResponse {
  string greeting = 1;
}

service GreetService {
  rpc Greet(GreetRequest) returns (GreetResponse) {}
  rpc GreetStream(stream GreetStreamRequest) returns (stream GreetStreamResponse) {}
  rpc GreetClientStream(stream GreetClientStreamRequest) returns (GreetClientStreamResponse) {}
  rpc GreetServerStream(GreetServerStreamRequest) returns (stream GreetServerStreamResponse) {}
}`
