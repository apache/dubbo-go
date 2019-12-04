/*
Licensed to the Apache Software Foundation (ASF) under one or more
contributor license agreements.  See the NOTICE file distributed with
this work for additional information regarding copyright ownership.
The ASF licenses this file to You under the Apache License, Version 2.0
(the "License"); you may not use this file except in compliance with
the License.  You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package grpc

import (
	"reflect"

	"google.golang.org/grpc"
)

func init() {

}

type dubboGrpcCli interface {
	getDubboStub(conn grpc.ClientConn)
}

var (
	cliType = reflect.TypeOf((*dubboGrpcCli)(nil))
)

type Client struct {
	*grpc.ClientConn
	invoker reflect.Value
}

func NewClient(impl interface{}) *Client {

	conn, err := grpc.Dial("addresss", grpc.WithInsecure(), grpc.WithBlock())
	if err != nil {
		panic(err)
	}

	if reflect.TypeOf(impl).Implements(cliType.Elem()) {
		panic(err)
	}

	in := []reflect.Value{}
	in = append(in, reflect.ValueOf(conn))
	method, _ := cliType.MethodByName("getDubboStub")
	res := method.Func.Call(in)
	invoker := res[0].Interface()

	return &Client{
		ClientConn: conn,
		invoker:    reflect.ValueOf(invoker),
	}
}
