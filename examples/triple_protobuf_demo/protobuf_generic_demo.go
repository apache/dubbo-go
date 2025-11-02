/*
 * Triple泛化调用 Protobuf 支持演示
 */

package main

import (
	"context"
	"fmt"
	"time"
)

import (
	"dubbo.apache.org/dubbo-go/v3/protocol/triple"
)

func main() {
	fmt.Println("🚀 Triple泛化调用 Protobuf 支持演示")
	fmt.Println("====================================")

	// 演示1: 基础Protobuf类型调用
	fmt.Println("\n1. 📝 基础Protobuf类型调用")
	demonstrateBasicProtobufTypes()

	// 演示2: 复杂Protobuf消息调用
	fmt.Println("\n2. 🏗️ 复杂Protobuf消息调用")
	demonstrateComplexProtobufMessages()

	// 演示3: 电商订单服务 (Protobuf)
	fmt.Println("\n3. 🛒 电商订单服务演示")
	demonstrateEcommerceProtobuf()

	// 演示4: 用户认证服务 (Protobuf)
	fmt.Println("\n4. 🔐 用户认证服务演示")
	demonstrateUserAuthProtobuf()

	// 演示5: gRPC特性支持
	fmt.Println("\n5. ⚙️ gRPC特性支持演示")
	demonstrateGRPCFeatures()

	// 演示6: 类型转换演示
	fmt.Println("\n6. 🔄 类型转换机制演示")
	demonstrateTypeConversion()

	fmt.Println("\n🎉 Protobuf支持演示完成!")
}

// 基础Protobuf类型调用演示
func demonstrateBasicProtobufTypes() {
	fmt.Println("演示各种基础Protobuf类型的调用...")

	// 创建支持Protobuf的Triple泛化服务
	userService := triple.NewTripleGenericService(
		"tri://user-service:20000/com.example.UserService")

	ctx := context.Background()

	fmt.Println("\n📋 Protobuf基础类型映射:")
	fmt.Println("  string  → Go string")
	fmt.Println("  int32   → Go int32")
	fmt.Println("  int64   → Go int64")
	fmt.Println("  float   → Go float32")
	fmt.Println("  double  → Go float64")
	fmt.Println("  bool    → Go bool")
	fmt.Println("  bytes   → Go []byte")

	// 基础类型调用示例
	testCases := []struct {
		name       string
		method     string
		paramTypes []string
		args       []any
		desc       string
	}{
		{
			name:       "字符串参数",
			method:     "UpdateUserName",
			paramTypes: []string{"string"},
			args:       []any{"张三"},
			desc:       "protobuf string类型",
		},
		{
			name:       "整数参数",
			method:     "UpdateUserAge",
			paramTypes: []string{"int32"},
			args:       []any{int32(28)},
			desc:       "protobuf int32类型",
		},
		{
			name:       "长整数参数",
			method:     "UpdateUserId",
			paramTypes: []string{"int64"},
			args:       []any{int64(1234567890)},
			desc:       "protobuf int64类型",
		},
		{
			name:       "浮点数参数",
			method:     "UpdateUserScore",
			paramTypes: []string{"float64"},
			args:       []any{95.5},
			desc:       "protobuf double类型",
		},
		{
			name:       "布尔参数",
			method:     "SetUserActive",
			paramTypes: []string{"bool"},
			args:       []any{true},
			desc:       "protobuf bool类型",
		},
		{
			name:       "字节数组参数",
			method:     "UpdateUserAvatar",
			paramTypes: []string{"bytes"},
			args:       []any{[]byte("avatar_binary_data")},
			desc:       "protobuf bytes类型",
		},
	}

	for _, tc := range testCases {
		fmt.Printf("\n🔧 测试: %s\n", tc.name)
		fmt.Printf("  方法: %s\n", tc.method)
		fmt.Printf("  类型: %v\n", tc.paramTypes)
		fmt.Printf("  参数: %v\n", tc.args)
		fmt.Printf("  说明: %s\n", tc.desc)

		result, err := userService.Invoke(ctx, tc.method, tc.paramTypes, tc.args)
		if err != nil {
			fmt.Printf("  结果: ❌ %v (预期的网络错误)\n", err)
		} else {
			fmt.Printf("  结果: ✅ %v\n", result)
		}
	}
}

// 复杂Protobuf消息调用演示
func demonstrateComplexProtobufMessages() {
	fmt.Println("演示复杂Protobuf消息结构的调用...")

	userService := triple.NewTripleGenericService(
		"tri://user-service:20000/com.example.UserService")

	ctx := context.Background()

	fmt.Println("\n🏗️ 复杂Protobuf消息示例:")
	fmt.Println("假设有以下Protobuf定义:")
	fmt.Println("```protobuf")
	fmt.Println("message User {")
	fmt.Println("  int64 id = 1;")
	fmt.Println("  string name = 2;")
	fmt.Println("  UserProfile profile = 3;")
	fmt.Println("  repeated string hobbies = 4;")
	fmt.Println("  map<string, string> metadata = 5;")
	fmt.Println("}")
	fmt.Println("")
	fmt.Println("message UserProfile {")
	fmt.Println("  int32 age = 1;")
	fmt.Println("  string email = 2;")
	fmt.Println("  Address address = 3;")
	fmt.Println("}")
	fmt.Println("")
	fmt.Println("message Address {")
	fmt.Println("  string country = 1;")
	fmt.Println("  string city = 2;")
	fmt.Println("  string street = 3;")
	fmt.Println("}")
	fmt.Println("```")

	// 构造复杂的嵌套Protobuf消息
	complexUser := map[string]any{
		"id":   int64(12345),
		"name": "张三",
		"profile": map[string]any{
			"age":   int32(28),
			"email": "zhangsan@example.com",
			"address": map[string]any{
				"country": "中国",
				"city":    "北京",
				"street":  "长安街1号",
			},
		},
		"hobbies": []string{"阅读", "旅游", "编程", "摄影"},
		"metadata": map[string]any{
			"source":     "web_registration",
			"campaign":   "spring_2024",
			"referrer":   "google_ads",
			"user_agent": "Mozilla/5.0...",
		},
	}

	fmt.Println("\n📝 调用复杂消息:")
	fmt.Printf("  消息结构: %+v\n", complexUser)

	result, err := userService.Invoke(ctx, "CreateUser",
		[]string{"com.example.User"},
		[]any{complexUser})

	if err != nil {
		fmt.Printf("  结果: ❌ %v (预期的网络错误)\n", err)
	} else {
		fmt.Printf("  结果: ✅ %v\n", result)
	}

	// 数组和重复字段
	fmt.Println("\n📋 批量操作演示:")
	batchUsers := []any{
		map[string]any{
			"id":   int64(1001),
			"name": "用户1",
			"profile": map[string]any{
				"age":   int32(25),
				"email": "user1@example.com",
			},
		},
		map[string]any{
			"id":   int64(1002),
			"name": "用户2",
			"profile": map[string]any{
				"age":   int32(30),
				"email": "user2@example.com",
			},
		},
	}

	result, err = userService.Invoke(ctx, "BatchCreateUsers",
		[]string{"repeated:com.example.User"},
		[]any{batchUsers})

	if err != nil {
		fmt.Printf("  批量创建结果: ❌ %v (预期的网络错误)\n", err)
	} else {
		fmt.Printf("  批量创建结果: ✅ %v\n", result)
	}
}

// 电商订单服务演示
func demonstrateEcommerceProtobuf() {
	fmt.Println("演示电商订单服务的Protobuf调用...")

	orderService := triple.NewTripleGenericService(
		"tri://order-service:20000/ecommerce.OrderService")

	ctx := context.Background()

	fmt.Println("\n🛒 电商Protobuf服务定义:")
	fmt.Println("```protobuf")
	fmt.Println("message Product {")
	fmt.Println("  int64 id = 1;")
	fmt.Println("  string name = 2;")
	fmt.Println("  double price = 3;")
	fmt.Println("  int32 quantity = 4;")
	fmt.Println("}")
	fmt.Println("")
	fmt.Println("message CreateOrderRequest {")
	fmt.Println("  int64 user_id = 1;")
	fmt.Println("  repeated Product products = 2;")
	fmt.Println("  string shipping_address = 3;")
	fmt.Println("}")
	fmt.Println("```")

	// 构造订单请求
	createOrderReq := map[string]any{
		"user_id": int64(12345),
		"products": []any{
			map[string]any{
				"id":       int64(1001),
				"name":     "iPhone 15 Pro Max",
				"price":    1199.99,
				"quantity": int32(1),
			},
			map[string]any{
				"id":       int64(1002),
				"name":     "AirPods Pro (第二代)",
				"price":    249.99,
				"quantity": int32(1),
			},
			map[string]any{
				"id":       int64(1003),
				"name":     "MagSafe充电器",
				"price":    39.99,
				"quantity": int32(2),
			},
		},
		"shipping_address": "北京市朝阳区建国门外大街1号",
	}

	fmt.Println("\n📦 创建订单:")
	fmt.Printf("  用户ID: %v\n", createOrderReq["user_id"])
	fmt.Printf("  商品数量: %d\n", len(createOrderReq["products"].([]any)))
	fmt.Printf("  配送地址: %v\n", createOrderReq["shipping_address"])

	// 带gRPC metadata的调用
	metadata := map[string]any{
		"user-id":       "12345",
		"request-id":    fmt.Sprintf("req-%d", time.Now().Unix()),
		"client-type":   "mobile-app",
		"app-version":   "2.1.0",
		"device-id":     "device-abc123",
		"authorization": "Bearer eyJhbGciOiJIUzI1NiIs...",
	}

	result, err := orderService.InvokeWithAttachments(ctx, "CreateOrder",
		[]string{"ecommerce.CreateOrderRequest"},
		[]any{createOrderReq},
		metadata)

	if err != nil {
		fmt.Printf("  订单创建结果: ❌ %v (预期的网络错误)\n", err)
	} else {
		fmt.Printf("  订单创建结果: ✅ %v\n", result)
	}

	// 查询订单
	fmt.Println("\n🔍 查询订单:")
	queryResult, err := orderService.InvokeWithAttachments(ctx, "GetOrder",
		[]string{"int64"},
		[]any{int64(987654321)},
		map[string]any{
			"user-id":  "12345",
			"trace-id": "trace-query-001",
		})

	if err != nil {
		fmt.Printf("  订单查询结果: ❌ %v (预期的网络错误)\n", err)
	} else {
		fmt.Printf("  订单查询结果: ✅ %v\n", queryResult)
	}
}

// 用户认证服务演示
func demonstrateUserAuthProtobuf() {
	fmt.Println("演示用户认证服务的Protobuf异步调用...")

	authService := triple.NewTripleGenericService(
		"tri://auth-service:20000/auth.AuthService")

	ctx := context.Background()

	fmt.Println("\n🔐 认证服务Protobuf定义:")
	fmt.Println("```protobuf")
	fmt.Println("message LoginRequest {")
	fmt.Println("  string username = 1;")
	fmt.Println("  string password = 2;")
	fmt.Println("  DeviceInfo device_info = 3;")
	fmt.Println("}")
	fmt.Println("")
	fmt.Println("message DeviceInfo {")
	fmt.Println("  string device_id = 1;")
	fmt.Println("  DeviceType device_type = 2;")
	fmt.Println("  string app_version = 3;")
	fmt.Println("}")
	fmt.Println("```")

	// 构造登录请求
	loginReq := map[string]any{
		"username": "zhangsan",
		"password": "hashed_password_here",
		"device_info": map[string]any{
			"device_id":   "device-12345-abcde",
			"device_type": int32(1), // MOBILE = 1
			"app_version": "2.1.0",
		},
	}

	fmt.Println("\n🔑 异步登录请求:")
	fmt.Printf("  用户名: %v\n", loginReq["username"])
	fmt.Printf("  设备信息: %v\n", loginReq["device_info"])

	// 异步调用认证服务
	callID, err := authService.InvokeAsync(ctx, "Login",
		[]string{"auth.LoginRequest"},
		[]any{loginReq},
		map[string]any{
			"client-ip":      "192.168.1.100",
			"user-agent":     "MyApp/2.1.0 (iOS; iPhone13,2)",
			"request-time":   time.Now().Format(time.RFC3339),
			"session-id":     "session-abc123",
			"correlation-id": "corr-" + fmt.Sprintf("%d", time.Now().UnixNano()),
		},
		func(result any, err error) {
			if err != nil {
				fmt.Printf("\n  🔐 异步登录回调 - 失败: %v\n", err)
				return
			}

			fmt.Printf("\n  🔐 异步登录回调 - 成功!\n")

			// 解析Protobuf响应
			if response, ok := result.(map[string]any); ok {
				fmt.Printf("    响应结构: %+v\n", response)

				if success, ok := response["success"].(bool); ok && success {
					if token, ok := response["access_token"].(string); ok {
						fmt.Printf("    访问令牌: %s...\n", token[:min(20, len(token))])
					}
					if refreshToken, ok := response["refresh_token"].(string); ok {
						fmt.Printf("    刷新令牌: %s...\n", refreshToken[:min(20, len(refreshToken))])
					}
					if expiresIn, ok := response["expires_in"].(int64); ok {
						fmt.Printf("    过期时间: %d秒\n", expiresIn)
					}
				}
			}
		})

	if err != nil {
		fmt.Printf("  异步登录启动失败: %v\n", err)
		return
	}

	fmt.Printf("  异步登录已启动，调用ID: %s\n", callID)

	// 模拟等待异步调用完成
	time.Sleep(100 * time.Millisecond)

	// 其他认证相关操作
	fmt.Println("\n🔄 刷新令牌:")
	refreshReq := map[string]any{
		"refresh_token": "refresh_token_here",
		"device_id":     "device-12345-abcde",
	}

	refreshResult, err := authService.InvokeWithAttachments(ctx, "RefreshToken",
		[]string{"auth.RefreshTokenRequest"},
		[]any{refreshReq},
		map[string]any{
			"client-ip": "192.168.1.100",
			"device-id": "device-12345-abcde",
		})

	if err != nil {
		fmt.Printf("  刷新令牌结果: ❌ %v (预期的网络错误)\n", err)
	} else {
		fmt.Printf("  刷新令牌结果: ✅ %v\n", refreshResult)
	}
}

// gRPC特性支持演示
func demonstrateGRPCFeatures() {
	fmt.Println("演示Triple泛化调用对gRPC特性的支持...")

	notificationService := triple.NewTripleGenericService(
		"tri://notification-service:20000/notification.NotificationService")

	ctx := context.Background()

	fmt.Println("\n⚙️ 支持的gRPC特性:")
	fmt.Println("  ✅ Unary RPC (单次请求-响应)")
	fmt.Println("  ✅ gRPC Metadata (通过附件传递)")
	fmt.Println("  ✅ 错误状态码")
	fmt.Println("  ✅ 超时控制")
	fmt.Println("  ✅ 压缩支持")
	fmt.Println("  ⚠️ 流式RPC (部分支持)")

	// 1. gRPC Metadata演示
	fmt.Println("\n📡 gRPC Metadata演示:")

	grpcMetadata := map[string]any{
		// 标准gRPC headers
		"content-type":         "application/grpc+proto",
		"grpc-encoding":        "gzip",
		"grpc-accept-encoding": "gzip,deflate",
		"grpc-timeout":         "30S",
		"user-agent":           "dubbo-go/3.0 grpc-go/1.50.0",

		// 自定义headers
		"authorization":    "Bearer token123",
		"x-request-id":     "req-" + fmt.Sprintf("%d", time.Now().Unix()),
		"x-user-id":        "user-12345",
		"x-trace-id":       "trace-abcde-12345",
		"x-span-id":        "span-fghij-67890",
		"x-client-version": "2.1.0",

		// 业务相关metadata
		"x-tenant-id":   "tenant-corp-abc",
		"x-environment": "production",
		"x-region":      "us-west-2",
	}

	notificationReq := map[string]any{
		"user_id":  int64(12345),
		"type":     int32(1), // EMAIL = 1
		"title":    "欢迎使用我们的服务",
		"content":  "感谢您注册我们的平台，祝您使用愉快！",
		"channels": []string{"email", "sms", "push"},
	}

	fmt.Printf("  gRPC Metadata 数量: %d\n", len(grpcMetadata))
	fmt.Printf("  通知请求: %+v\n", notificationReq)

	result, err := notificationService.InvokeWithAttachments(ctx, "SendNotification",
		[]string{"notification.SendNotificationRequest"},
		[]any{notificationReq},
		grpcMetadata)

	if err != nil {
		fmt.Printf("  gRPC调用结果: ❌ %v (可能包含gRPC状态码)\n", err)
	} else {
		fmt.Printf("  gRPC调用结果: ✅ %v\n", result)
	}

	// 2. 超时控制演示
	fmt.Println("\n⏰ 超时控制演示:")

	// 创建带超时的上下文
	timeoutCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	_, err = notificationService.InvokeWithAttachments(timeoutCtx, "SendBulkNotification",
		[]string{"notification.BulkNotificationRequest"},
		[]any{
			map[string]any{
				"user_ids": []int64{1001, 1002, 1003, 1004, 1005},
				"message":  "系统维护通知",
			},
		},
		map[string]any{
			"grpc-timeout": "5S",
		})

	if err != nil {
		fmt.Printf("  超时控制结果: ❌ %v (可能是超时或网络错误)\n", err)
	} else {
		fmt.Printf("  超时控制结果: ✅ 调用成功\n")
	}

	// 3. 批量调用演示 (利用Triple的增强功能)
	fmt.Println("\n📦 批量gRPC调用演示:")

	bulkRequests := []triple.TripleInvocationRequest{
		{
			MethodName: "SendNotification",
			Types:      []string{"notification.SendNotificationRequest"},
			Args: []any{
				map[string]any{
					"user_id": int64(1001),
					"type":    int32(1),
					"title":   "消息1",
					"content": "内容1",
				},
			},
			Attachments: map[string]any{
				"grpc-timeout": "10S",
				"x-batch-id":   "batch-001",
				"x-item-id":    "item-1",
			},
		},
		{
			MethodName: "SendNotification",
			Types:      []string{"notification.SendNotificationRequest"},
			Args: []any{
				map[string]any{
					"user_id": int64(1002),
					"type":    int32(2),
					"title":   "消息2",
					"content": "内容2",
				},
			},
			Attachments: map[string]any{
				"grpc-timeout": "10S",
				"x-batch-id":   "batch-001",
				"x-item-id":    "item-2",
			},
		},
	}

	batchResults, err := notificationService.BatchInvoke(ctx, bulkRequests)
	if err != nil {
		fmt.Printf("  批量gRPC调用失败: %v\n", err)
	} else {
		fmt.Printf("  批量gRPC调用完成: %d个请求\n", len(batchResults))
		for i, result := range batchResults {
			if result.Error != nil {
				fmt.Printf("    请求%d: ❌ %v\n", i+1, result.Error)
			} else {
				fmt.Printf("    请求%d: ✅ 成功\n", i+1)
			}
		}
	}
}

// 类型转换机制演示
func demonstrateTypeConversion() {
	fmt.Println("演示Triple泛化调用的Protobuf类型转换机制...")

	conversionService := triple.NewTripleGenericService(
		"tri://conversion-service:20000/conversion.ConversionService")

	ctx := context.Background()

	fmt.Println("\n🔄 类型转换测试:")

	// 测试各种类型转换场景
	conversionTests := []struct {
		name         string
		method       string
		inputType    string
		inputValue   any
		expectedType string
		description  string
	}{
		{
			name:         "Go int 到 protobuf int32",
			method:       "ProcessInt32",
			inputType:    "int32",
			inputValue:   123,
			expectedType: "int32",
			description:  "自动将Go int转换为protobuf int32",
		},
		{
			name:         "Go int 到 protobuf int64",
			method:       "ProcessInt64",
			inputType:    "int64",
			inputValue:   1234567890,
			expectedType: "int64",
			description:  "自动将Go int转换为protobuf int64",
		},
		{
			name:         "Go float64 到 protobuf float",
			method:       "ProcessFloat",
			inputType:    "float32",
			inputValue:   3.14159,
			expectedType: "float32",
			description:  "自动将Go float64转换为protobuf float",
		},
		{
			name:         "Go string 到 protobuf string",
			method:       "ProcessString",
			inputType:    "string",
			inputValue:   "Hello Protobuf 你好",
			expectedType: "string",
			description:  "直接传递string类型",
		},
		{
			name:         "Go []byte 到 protobuf bytes",
			method:       "ProcessBytes",
			inputType:    "bytes",
			inputValue:   []byte("binary data 二进制数据"),
			expectedType: "bytes",
			description:  "直接传递bytes类型",
		},
		{
			name:      "Go map 到 protobuf message",
			method:    "ProcessMessage",
			inputType: "conversion.MessageType",
			inputValue: map[string]any{
				"id":     int64(123),
				"name":   "测试消息",
				"active": true,
				"score":  95.5,
				"tags":   []string{"test", "protobuf"},
			},
			expectedType: "message",
			description:  "将Go map转换为protobuf message",
		},
	}

	for _, test := range conversionTests {
		fmt.Printf("\n🧪 测试: %s\n", test.name)
		fmt.Printf("  输入类型: %T\n", test.inputValue)
		fmt.Printf("  输入值: %v\n", test.inputValue)
		fmt.Printf("  期望转换: %s → %s\n", test.inputType, test.expectedType)
		fmt.Printf("  说明: %s\n", test.description)

		_, err := conversionService.Invoke(ctx, test.method,
			[]string{test.inputType},
			[]any{test.inputValue})

		if err != nil {
			fmt.Printf("  转换结果: ❌ %v (预期的网络错误)\n", err)
		} else {
			fmt.Printf("  转换结果: ✅ 成功转换\n")
		}
	}

	// 复杂嵌套结构转换
	fmt.Println("\n🏗️ 复杂嵌套结构转换:")

	complexStruct := map[string]any{
		"header": map[string]any{
			"request_id": "req-12345",
			"timestamp":  int64(time.Now().Unix()),
			"version":    "v1.0",
		},
		"payload": map[string]any{
			"users": []any{
				map[string]any{
					"id":    int64(1001),
					"name":  "用户1",
					"age":   int32(25),
					"email": "user1@example.com",
				},
				map[string]any{
					"id":    int64(1002),
					"name":  "用户2",
					"age":   int32(30),
					"email": "user2@example.com",
				},
			},
			"metadata": map[string]any{
				"total_count": int32(2),
				"page_size":   int32(10),
				"has_more":    false,
			},
		},
		"footer": map[string]any{
			"processing_time": float64(1.23),
			"server_id":       "server-abc123",
		},
	}

	fmt.Printf("  复杂结构层级: 3层嵌套\n")
	fmt.Printf("  包含类型: map, []interface{}, int64, int32, string, bool, float64\n")
	fmt.Printf("  结构大小: %d个顶级字段\n", len(complexStruct))

	_, err := conversionService.Invoke(ctx, "ProcessComplexStructure",
		[]string{"conversion.ComplexRequest"},
		[]any{complexStruct})

	if err != nil {
		fmt.Printf("  复杂结构转换: ❌ %v (预期的网络错误)\n", err)
	} else {
		fmt.Printf("  复杂结构转换: ✅ 成功处理复杂嵌套结构\n")
	}

	fmt.Println("\n💡 类型转换最佳实践:")
	fmt.Println("  ✅ 使用正确的Go类型 (int32, int64, float32, float64)")
	fmt.Println("  ✅ 明确指定protobuf消息类型名")
	fmt.Println("  ✅ 使用map[string]interface{}构造复杂消息")
	fmt.Println("  ✅ 数组使用[]interface{}或具体类型切片")
	fmt.Println("  ⚠️ 避免使用Go的默认int类型 (可能导致类型不匹配)")
	fmt.Println("  ⚠️ 注意浮点数精度 (float32 vs float64)")
}

// 辅助函数
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
