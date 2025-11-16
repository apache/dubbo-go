
// /*
//  * Triple 泛化调用基础使用示例
//  */

// package main

// import (
// 	"context"
// 	"fmt"
// 	"time"

// 	"dubbo.apache.org/dubbo-go/v3/protocol/triple"
// )

// func main() {
// 	fmt.Println("🚀 Triple 泛化调用基础使用示例")
// 	fmt.Println("==============================")

// 	// 创建泛化服务客户端
// 	tripleGS := triple.NewTripleGenericService("tri://127.0.0.1:20000/com.example.UserService?serialization=hessian2")
// 	ctx := context.Background()

// 	// 示例1: 简单字符串方法调用
// 	fmt.Println("\n1. 📝 简单字符串方法调用")
// 	result, err := tripleGS.Invoke(ctx, "sayHello", []string{"string"}, []interface{}{"World"})
// 	if err != nil {
// 		fmt.Printf("❌ 调用失败: %v\n", err)
// 	} else {
// 		fmt.Printf("✅ 调用成功: %v\n", result)
// 	}

// 	// 示例2: 数值计算方法
// 	fmt.Println("\n2. 🧮 数值计算方法")
// 	result, err = tripleGS.Invoke(ctx, "add", []string{"int32", "int32"}, []interface{}{int32(10), int32(20)})
// 	if err != nil {
// 		fmt.Printf("❌ 调用失败: %v\n", err)
// 	} else {
// 		fmt.Printf("✅ 调用成功: %v\n", result)
// 	}

// 	// 示例3: 复杂对象方法
// 	fmt.Println("\n3. 👤 用户对象创建")
// 	user := map[string]interface{}{
// 		"name":  "张三",
// 		"age":   28,
// 		"email": "zhangsan@example.com",
// 		"address": map[string]interface{}{
// 			"city":    "北京",
// 			"street":  "长安街1号",
// 			"zipcode": "100000",
// 		},
// 		"hobbies": []string{"阅读", "旅游", "编程"},
// 	}

// 	result, err = tripleGS.Invoke(ctx, "createUser", []string{"map"}, []interface{}{user})
// 	if err != nil {
// 		fmt.Printf("❌ 调用失败: %v\n", err)
// 	} else {
// 		fmt.Printf("✅ 调用成功: %v\n", result)
// 	}

// 	// 示例4: 数组参数方法
// 	fmt.Println("\n4. 📊 批量查询用户")
// 	userIDs := []int64{1001, 1002, 1003, 1004, 1005}
// 	result, err = tripleGS.Invoke(ctx, "batchGetUsers", []string{"[]int64"}, []interface{}{userIDs})
// 	if err != nil {
// 		fmt.Printf("❌ 调用失败: %v\n", err)
// 	} else {
// 		fmt.Printf("✅ 调用成功: %v\n", result)
// 	}

// 	// 示例5: 多参数类型组合
// 	fmt.Println("\n5. 🔄 用户信息更新")
// 	updates := map[string]interface{}{
// 		"age":    30,
// 		"email":  "zhangsan_new@example.com",
// 		"status": "active",
// 	}

// 	result, err = tripleGS.Invoke(ctx, "updateUser",
// 		[]string{"int64", "map", "bool"},
// 		[]interface{}{int64(1001), updates, true})
// 	if err != nil {
// 		fmt.Printf("❌ 调用失败: %v\n", err)
// 	} else {
// 		fmt.Printf("✅ 调用成功: %v\n", result)
// 	}

// 	// 示例6: 带附件的调用
// 	fmt.Println("\n6. 📎 带附件的服务调用")
// 	attachments := map[string]interface{}{
// 		"traceId":    "trace-123456",
// 		"userId":     "current-user-789",
// 		"requestId":  fmt.Sprintf("req-%d", time.Now().Unix()),
// 		"clientType": "web",
// 		"version":    "v1.0.0",
// 	}

// 	result, err = tripleGS.InvokeWithAttachments(ctx, "getUserProfile",
// 		[]string{"int64"}, []interface{}{int64(1001)}, attachments)
// 	if err != nil {
// 		fmt.Printf("❌ 调用失败: %v\n", err)
// 	} else {
// 		fmt.Printf("✅ 调用成功: %v\n", result)
// 	}

// 	// 示例7: 使用构建器创建附件
// 	fmt.Println("\n7. 🔧 使用附件构建器")
// 	builderAttachments := tripleGS.CreateAttachmentBuilder().
// 		SetString("service", "user-service").
// 		SetString("method", "getUserProfile").
// 		SetString("source", "mobile-app").
// 		SetInt("timeout", 5000).
// 		SetBool("cache", true).
// 		SetString("priority", "high").
// 		Build()

// 	result, err = tripleGS.InvokeWithAttachments(ctx, "getUserProfile",
// 		[]string{"int64"}, []interface{}{int64(1002)}, builderAttachments)
// 	if err != nil {
// 		fmt.Printf("❌ 调用失败: %v\n", err)
// 	} else {
// 		fmt.Printf("✅ 调用成功: %v\n", result)
// 	}

// 	fmt.Println("\n🎉 基础使用示例完成!")
// 	fmt.Println("💡 提示: 以上示例展示了Triple泛化调用的主要功能，实际使用时需要确保服务端已启动")
// }
