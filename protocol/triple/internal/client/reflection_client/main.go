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
	"context"
)

import (
	"github.com/dubbogo/gost/log/logger"

	"github.com/golang/protobuf/proto"

	"google.golang.org/protobuf/types/descriptorpb"
)

import (
	"dubbo.apache.org/dubbo-go/v3/client"
	_ "dubbo.apache.org/dubbo-go/v3/imports"
	reflection "dubbo.apache.org/dubbo-go/v3/protocol/triple/reflection/triple_reflection"
)

func main() {
	cli, err := client.NewClient(
		client.WithClientURL("tri://127.0.0.1:20000"),
	)
	if err != nil {
		panic(err)
	}
	svc, err := reflection.NewServerReflection(cli)
	if err != nil {
		panic(err)
	}
	stream, err := svc.ServerReflectionInfo(context.Background())
	if err != nil {
		panic(err)
	}
	testReflection(stream)
}

func testReflection(stream reflection.ServerReflection_ServerReflectionInfoClient) {
	if err := testFileByFilename(stream); err != nil {
		logger.Error(err)
	}
	if err := testFileContainingSymbol(stream); err != nil {
		logger.Error(err)
	}
	if err := testListServices(stream); err != nil {
		logger.Error(err)
	}
	if err := stream.CloseRequest(); err != nil {
		logger.Error(err)
	}
	if err := stream.CloseResponse(); err != nil {
		logger.Error(err)
	}
}

func testFileByFilename(stream reflection.ServerReflection_ServerReflectionInfoClient) error {
	logger.Info("start to test call FileByFilename")
	if err := stream.Send(&reflection.ServerReflectionRequest{
		MessageRequest: &reflection.ServerReflectionRequest_FileByFilename{FileByFilename: "reflection.proto"},
	}); err != nil {
		return err
	}
	recv, err := stream.Recv()
	if err != nil {
		return err
	}
	m := new(descriptorpb.FileDescriptorProto)
	if err = proto.Unmarshal(recv.GetFileDescriptorResponse().GetFileDescriptorProto()[0], m); err != nil {
		return err
	}
	logger.Infof("call FileByFilename 's resp : %s", m)
	return nil
}

func testFileContainingSymbol(stream reflection.ServerReflection_ServerReflectionInfoClient) error {
	logger.Info("start to test call FileContainingSymbol")
	if err := stream.Send(&reflection.ServerReflectionRequest{
		MessageRequest: &reflection.ServerReflectionRequest_FileContainingSymbol{FileContainingSymbol: "dubbo.reflection.v1alpha.ServerReflection"},
	}); err != nil {
		return err
	}
	recv, err := stream.Recv()
	if err != nil {
		return err
	}
	m := new(descriptorpb.FileDescriptorProto)
	if err = proto.Unmarshal(recv.GetFileDescriptorResponse().GetFileDescriptorProto()[0], m); err != nil {
		return err
	}
	logger.Infof("call FileContainingSymbol 's resp : %s", m)
	return nil
}

func testListServices(stream reflection.ServerReflection_ServerReflectionInfoClient) error {
	logger.Info("start to test call ListServices")
	if err := stream.Send(&reflection.ServerReflectionRequest{
		MessageRequest: &reflection.ServerReflectionRequest_ListServices{},
	}); err != nil {
		return err
	}
	recv, err := stream.Recv()
	if err != nil {
		return err
	}
	logger.Infof("call ListServices 's resp : %s", recv.GetListServicesResponse().GetService())
	return nil
}
