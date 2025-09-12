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
	"fmt"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/common/extension"
	"dubbo.apache.org/dubbo-go/v3/protocol/invocation"
	"dubbo.apache.org/dubbo-go/v3/protocol/protocolwrapper"
	"dubbo.apache.org/dubbo-go/v3/protocol/triple"
)

func main() {
	fmt.Println("🧪 Triple 泛化调用单元测试")
	fmt.Println("==========================")

	// 测试1: 验证 TripleGenericService 的 Invoke 方法
	fmt.Println("\n1️⃣ 测试 TripleGenericService.Invoke")
	tripleGS := triple.NewTripleGenericService("tri://127.0.0.1:20001/com.example.UserService")

	ctx := context.Background()
	result, err := tripleGS.Invoke(ctx, "GetUser", []string{"int64"}, []interface{}{int64(123)})

	if err != nil {
		fmt.Printf("❌ 调用失败: %v\n", err)
	} else {
		fmt.Printf("✅ 调用成功: %+v\n", result)
	}

	// 测试2: 验证直接构造 Invocation 的方式
	fmt.Println("\n2️⃣ 测试直接构造 Invocation")

	// 构造 URL
	url, _ := common.NewURL("tri://127.0.0.1:20001/com.example.UserService")
	url.SetParam(constant.GenericKey, constant.GenericSerializationDefault)
	url.SetParam(constant.IDLMode, constant.NONIDL)
	url.Methods = []string{constant.Generic}

	// 构造 Invocation
	req := []interface{}{"GetUser", []string{"int64"}, []interface{}{int64(456)}}
	resp := &[]interface{}{}

	inv := invocation.NewRPCInvocationWithOptions(
		invocation.WithMethodName(constant.Generic),
		invocation.WithArguments(req),
		invocation.WithParameterRawValues([]interface{}{req, resp}),
	)
	inv.SetAttachment(constant.GenericKey, constant.GenericSerializationDefault)
	inv.SetAttribute(constant.CallTypeKey, constant.CallUnary)

	// 获取 Invoker 并调用
	invoker := extension.GetProtocol(protocolwrapper.FILTER).Refer(url)
	if invoker == nil {
		fmt.Println("❌ 创建 Invoker 失败")
		return
	}

	result2 := invoker.Invoke(ctx, inv)
	if err := result2.Error(); err != nil {
		fmt.Printf("❌ Invocation 调用失败: %v\n", err)
	} else {
		fmt.Printf("✅ Invocation 调用成功: %+v\n", result2.Result())
	}

	fmt.Println("\n🎉 测试完成!")
}





