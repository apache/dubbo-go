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
	"time"
)

import (
	hessian "github.com/apache/dubbo-go-hessian2"

	"github.com/dubbogo/gost/log/logger"
)

import (
	"dubbo.apache.org/dubbo-go/v3/client"
	_ "dubbo.apache.org/dubbo-go/v3/imports"
	"dubbo.apache.org/dubbo-go/v3/protocol"
)

func main() {
	fmt.Println("🚀 启动 Triple 协议 Consumer (泛化调用测试)")
	fmt.Println("====================================")

	// 等待服务器启动
	fmt.Println("⏳ 等待服务器启动...")
	time.Sleep(3 * time.Second)

	// 创建客户端
	cli, err := client.NewClient(
		client.WithClientProtocol(
			protocol.WithTriple(),
		),
	)
	if err != nil {
		panic(fmt.Sprintf("创建客户端失败: %v", err))
	}

	fmt.Println("✅ Triple 客户端创建成功")

	// 测试1: 使用传统 GenericService (非IDL模式)
	fmt.Println("\n1️⃣ 测试传统 GenericService (非IDL模式)")
	fmt.Println("=====================================")

	genericService, err := cli.NewGenericService("com.example.UserService",
		client.WithURL("127.0.0.1:20001"),
		client.WithSerialization("hessian2"),
	)
	if err != nil {
		logger.Errorf("创建GenericService失败: %v", err)
	} else {
		ctx := context.Background()

		// 调用 GetUser
		result, err := genericService.Invoke(ctx, "GetUser", []string{"int64"}, []hessian.Object{hessian.Object(int64(123))})
		if err != nil {
			logger.Errorf("GenericService.GetUser 调用失败: %v", err)
		} else {
			logger.Infof("✅ GenericService.GetUser 调用成功: %+v", result)
		}
	}

	// 测试2: 使用新的 TripleGenericService
	fmt.Println("\n2️⃣ 测试新的 TripleGenericService")
	fmt.Println("================================")

	tripleGS, err := cli.NewTripleGenericService("tri://127.0.0.1:20001/com.example.UserService?serialization=hessian2")
	if err != nil {
		logger.Errorf("创建TripleGenericService失败: %v", err)
		return
	}

	fmt.Println("✅ TripleGenericService 创建成功")

	ctx := context.Background()

	// 测试 GetUser
	fmt.Println("\n🔍 测试 GetUser 方法:")
	result, err := tripleGS.Invoke(ctx, "GetUser", []string{"int64"}, []interface{}{int64(456)})
	if err != nil {
		logger.Errorf("❌ TripleGenericService.GetUser 调用失败: %v", err)
	} else {
		logger.Infof("✅ TripleGenericService.GetUser 调用成功: %+v", result)
	}

	// 测试 CreateUser
	fmt.Println("\n✨ 测试 CreateUser 方法:")
	newUser := map[string]interface{}{
		"name":  "张三",
		"email": "zhangsan@example.com",
		"age":   28,
	}
	result, err = tripleGS.Invoke(ctx, "CreateUser", []string{"map"}, []interface{}{newUser})
	if err != nil {
		logger.Errorf("❌ TripleGenericService.CreateUser 调用失败: %v", err)
	} else {
		logger.Infof("✅ TripleGenericService.CreateUser 调用成功: %+v", result)
	}

	// 测试 UpdateUser
	fmt.Println("\n📝 测试 UpdateUser 方法:")
	updates := map[string]interface{}{
		"name":  "张三更新",
		"age":   30,
		"email": "zhangsan_updated@example.com",
	}
	result, err = tripleGS.Invoke(ctx, "UpdateUser", []string{"int64", "map"}, []interface{}{int64(456), updates})
	if err != nil {
		logger.Errorf("❌ TripleGenericService.UpdateUser 调用失败: %v", err)
	} else {
		logger.Infof("✅ TripleGenericService.UpdateUser 调用成功: %+v", result)
	}

	// 测试 BatchGetUsers
	fmt.Println("\n📦 测试 BatchGetUsers 方法:")
	userIDs := []int64{100, 200, 300}
	result, err = tripleGS.Invoke(ctx, "BatchGetUsers", []string{"[]int64"}, []interface{}{userIDs})
	if err != nil {
		logger.Errorf("❌ TripleGenericService.BatchGetUsers 调用失败: %v", err)
	} else {
		logger.Infof("✅ TripleGenericService.BatchGetUsers 调用成功: %+v", result)
	}

	// 测试3: 带附件的调用
	fmt.Println("\n3️⃣ 测试带附件的泛化调用")
	fmt.Println("========================")

	attachments := map[string]interface{}{
		"traceId":   "trace-123-456",
		"userId":    "current-user-789",
		"timeout":   "5000",
		"requestId": "req-" + fmt.Sprint(time.Now().Unix()),
	}

	result, err = tripleGS.InvokeWithAttachments(ctx, "GetUser", []string{"int64"}, []interface{}{int64(789)}, attachments)
	if err != nil {
		logger.Errorf("❌ TripleGenericService.InvokeWithAttachments 调用失败: %v", err)
	} else {
		logger.Infof("✅ TripleGenericService.InvokeWithAttachments 调用成功: %+v", result)
	}

	// 测试4: 使用流式构建器
	fmt.Println("\n4️⃣ 测试流式附件构建器")
	fmt.Println("====================")

	builderAttachments := tripleGS.CreateAttachmentBuilder().
		SetString("service", "user-service").
		SetString("version", "v1.0.0").
		SetString("retries", "3").
		SetString("enableCache", "true").
		Build()

	result, err = tripleGS.InvokeWithAttachments(ctx, "GetUser", []string{"int64"}, []interface{}{int64(999)}, builderAttachments)
	if err != nil {
		logger.Errorf("❌ 流式构建器调用失败: %v", err)
	} else {
		logger.Infof("✅ 流式构建器调用成功: %+v", result)
	}

	fmt.Println("\n🎉 Triple 协议泛化调用测试完成!")
	fmt.Println("💡 测试结果总结:")
	fmt.Println("  📋 传统 GenericService: 已测试")
	fmt.Println("  🆕 TripleGenericService: 已测试")
	fmt.Println("  📎 附件透传: 已测试")
	fmt.Println("  🔧 流式构建器: 已测试")
	fmt.Println("  🔄 多种参数类型: 已测试")
}
