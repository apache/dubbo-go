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

package dubbo3

import (
	"context"
	"net"
	"reflect"
)

import (
	"github.com/golang/protobuf/proto"
	"google.golang.org/grpc"
)

import (
	"github.com/apache/dubbo-go/common"
	"github.com/apache/dubbo-go/common/constant"
	"github.com/apache/dubbo-go/common/logger"
	"github.com/apache/dubbo-go/config"
)

// TripleClient client endpoint for client end
type TripleClient struct {
	conn         net.Conn
	h2Controller *H2Controller
	addr         string
	Invoker      reflect.Value
	url          *common.URL
}

// TripleConn is the sturuct that called in pb.go file, it's client field contains all net logic of dubbo3
type TripleConn struct {
	client *TripleClient
}

// Invoke called when unary rpc 's pb.go file
func (t *TripleConn) Invoke(ctx context.Context, method string, args, reply interface{}, opts ...grpc.CallOption) error {
	t.client.Request(ctx, method, args, reply)
	return nil
}

// NewStream called when streaming rpc 's pb.go file
func (t *TripleConn) NewStream(ctx context.Context, method string, opts ...grpc.CallOption) (grpc.ClientStream, error) {
	return t.client.StreamRequest(ctx, method)
}

// newTripleConn new a triple conn with given @tripleclient, which contains all net logic
func newTripleConn(client *TripleClient) *TripleConn {
	return &TripleConn{
		client: client,
	}
}

// getInvoker return invoker that have service method
func getInvoker(impl interface{}, conn *TripleConn) interface{} {
	in := make([]reflect.Value, 0, 16)
	in = append(in, reflect.ValueOf(conn))

	method := reflect.ValueOf(impl).MethodByName("GetDubboStub")
	res := method.Call(in)
	// res[0] is a struct that contains SayHello method, res[0] is greeter Client in example
	// it's SayHello methodwill call specific of conn's invoker.
	return res[0].Interface()
}

// NewTripleClient create triple client with given @url,
// it's return tripleClient , contains invoker, and contain triple conn
func NewTripleClient(url *common.URL) (*TripleClient, error) {
	key := url.GetParam(constant.BEAN_NAME_KEY, "")
	impl := config.GetConsumerService(key)
	tripleClient := &TripleClient{
		url: url,
	}
	if err := tripleClient.connect(url); err != nil {
		return nil, err
	}
	invoker := getInvoker(impl, newTripleConn(tripleClient)) // put dubbo3 network logic to tripleConn
	tripleClient.Invoker = reflect.ValueOf(invoker)
	return tripleClient, nil
}

// Connect called when new TripleClient, which start a tcp conn with target addr
func (t *TripleClient) connect(url *common.URL) error {
	logger.Warn("want to connect to url = ", url.Location)
	conn, err := net.Dial("tcp", url.Location)
	if err != nil {
		return err
	}
	t.addr = url.Location
	t.conn = conn
	t.h2Controller, err = NewH2Controller(t.conn, false, nil, url)
	if err != nil {
		return err
	}
	return t.h2Controller.H2ShakeHand()
}

// Request call h2Controller to send unary rpc req to server
func (t *TripleClient) Request(ctx context.Context, method string, arg, reply interface{}) error {
	reqData, err := proto.Marshal(arg.(proto.Message))
	if err != nil {
		panic("client request marshal not ok ")
	}
	if err := t.h2Controller.UnaryInvoke(ctx, method, t.conn.RemoteAddr().String(), reqData, reply, t.url); err != nil {
		return err
	}
	return nil
}

// StreamRequest call h2Controller to send streaming request to sever, to start link.
func (t *TripleClient) StreamRequest(ctx context.Context, method string) (grpc.ClientStream, error) {
	return t.h2Controller.StreamInvoke(ctx, method)
}

// Close
func (t *TripleClient) Close() {

}

// IsAvailable
func (t *TripleClient) IsAvailable() bool {
	return true
}
