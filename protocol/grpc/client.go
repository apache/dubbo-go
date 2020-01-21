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
)

import (
	"google.golang.org/grpc"
)

import (
	"github.com/apache/dubbo-go/common"
	"github.com/apache/dubbo-go/common/constant"
	"github.com/apache/dubbo-go/config"
)

// Client ...
type Client struct {
	*grpc.ClientConn
	invoker reflect.Value
}

// NewClient ...
func NewClient(url common.URL) *Client {
	conn, err := grpc.Dial(url.Location, grpc.WithInsecure(), grpc.WithBlock())
	if err != nil {
		panic(err)
	}

	key := url.GetParam(constant.BEAN_NAME_KEY, "")
	impl := config.GetConsumerService(key)
	invoker := getInvoker(impl, conn)

	return &Client{
		ClientConn: conn,
		invoker:    reflect.ValueOf(invoker),
	}
}

func getInvoker(impl interface{}, conn *grpc.ClientConn) interface{} {
	in := []reflect.Value{}
	in = append(in, reflect.ValueOf(conn))
	method := reflect.ValueOf(impl).MethodByName("GetDubboStub")
	res := method.Call(in)
	return res[0].Interface()
}
