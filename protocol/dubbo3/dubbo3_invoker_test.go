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

// removed unused mock URL constant

//func TestInvoke(t *testing.T) {
//	go internal.InitDubboServer()
//	time.Sleep(time.Second * 3)
//
//	url, err := common.NewURL(mockDubbo3CommonUrl2)
//	assert.Nil(t, err)
//
//	invoker, err := NewDubboInvoker(url)
//
//	assert.Nil(t, err)
//
//	args := []reflect.Value{}
//	args = append(args, reflect.ValueOf(&internal.HelloRequest{Name: "request name"}))
//	bizReply := &internal.HelloReply{}
//	invo := invocation.NewRPCInvocationWithOptions(invocation.WithMethodName("SayHello"),
//		invocation.WithParameterValues(args), invocation.WithReply(bizReply))
//	res := invoker.Invoke(context.Background(), invo)
//	assert.Nil(t, res.Error())
//	assert.NotNil(t, res.Result())
//	assert.Equal(t, "Hello request name", bizReply.Message)
//}
//
//func TestInvokeTimoutConfig(t *testing.T) {
//	go internal.InitDubboServer()
//	time.Sleep(time.Second * 3)
//
//	// test for millisecond
//	tmpMockUrl := mockDubbo3CommonUrl2 + "&timeout=300ms"
//	url, err := common.NewURL(tmpMockUrl)
//	assert.Nil(t, err)
//
//	invoker, err := NewDubboInvoker(url)
//	assert.Nil(t, err)
//
//	assert.Equal(t, invoker.timeout, time.Duration(time.Millisecond*300))
//
//	// test for second
//	tmpMockUrl = mockDubbo3CommonUrl2 + "&timeout=1s"
//	url, err = common.NewURL(tmpMockUrl)
//	assert.Nil(t, err)
//
//	invoker, err = NewDubboInvoker(url)
//	assert.Nil(t, err)
//	assert.Equal(t, invoker.timeout, time.Duration(time.Second))
//
//	// test for timeout default config
//	url, err = common.NewURL(mockDubbo3CommonUrl2)
//	assert.Nil(t, err)
//
//	invoker, err = NewDubboInvoker(url)
//	assert.Nil(t, err)
//	assert.Equal(t, invoker.timeout, time.Duration(time.Second*3))
//}
