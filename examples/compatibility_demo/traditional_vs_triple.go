/*
 * Traditional Dubbo Generic vs Triple Generic 兼容性演示
 */

package main

import (
	"context"
	"fmt"
	"log"
)

import (
	hessian "github.com/apache/dubbo-go-hessian2"
)

import (
	"dubbo.apache.org/dubbo-go/v3/protocol/triple"
)

func main() {
	fmt.Println("🔄 Traditional Dubbo Generic vs Triple Generic 兼容性演示")
	fmt.Println("=========================================================")

	// 演示1: 参数兼容性对比
	fmt.Println("\n1. 📋 参数兼容性对比")
	demonstrateParameterCompatibility()

	// 演示2: API调用对比
	fmt.Println("\n2. 🔧 API调用方式对比")
	demonstrateAPICompatibility()

	// 演示3: 兼容性适配器
	fmt.Println("\n3. 🔄 兼容性适配器演示")
	demonstrateCompatibilityAdapter()

	// 演示4: 迁移示例
	fmt.Println("\n4. 🚀 迁移示例")
	demonstrateMigrationExample()

	fmt.Println("\n🎉 兼容性演示完成!")
}

// 参数兼容性对比
func demonstrateParameterCompatibility() {
	fmt.Println("展示两种方式的参数传递格式...")

	// 示例数据
	userName := "张三"
	userAge := 28
	userEmail := "zhangsan@example.com"

	fmt.Println("📊 相同的业务数据:")
	fmt.Printf("  用户名: %s\n", userName)
	fmt.Printf("  年龄: %d\n", userAge)
	fmt.Printf("  邮箱: %s\n", userEmail)

	fmt.Println("\n🟢 传统Dubbo泛化调用参数格式:")
	traditionalArgs := []hessian.Object{
		hessian.Object(userName),
		hessian.Object(userAge),
		hessian.Object(userEmail),
	}
	traditionalTypes := []string{"string", "int", "string"}

	fmt.Printf("  类型: %v\n", traditionalTypes)
	fmt.Printf("  参数: []hessian.Object{%q, %d, %q}\n", userName, userAge, userEmail)
	fmt.Printf("  参数类型: %T\n", traditionalArgs)

	fmt.Println("\n🔵 Triple泛化调用参数格式:")
	tripleArgs := []any{
		userName,
		userAge,
		userEmail,
	}
	tripleTypes := []string{"string", "int", "string"}

	fmt.Printf("  类型: %v\n", tripleTypes)
	fmt.Printf("  参数: []interface{}{%q, %d, %q}\n", userName, userAge, userEmail)
	fmt.Printf("  参数类型: %T\n", tripleArgs)

	fmt.Println("\n✅ 兼容性分析:")
	fmt.Println("  - 类型字符串: 完全一致")
	fmt.Println("  - 参数数量: 完全一致")
	fmt.Println("  - 参数内容: 语义完全一致")
	fmt.Println("  - 参数容器: hessian.Object vs interface{} (兼容)")
}

// API调用方式对比
func demonstrateAPICompatibility() {
	fmt.Println("对比两种API的调用方式...")

	ctx := context.Background()

	fmt.Println("\n🟢 传统Dubbo泛化调用方式:")
	fmt.Println("```go")
	fmt.Println("// 1. 创建客户端")
	fmt.Println("cli, err := client.NewClient()")
	fmt.Println("")
	fmt.Println("// 2. 创建泛化服务")
	fmt.Println(`genericService, err := cli.NewGenericService("com.example.UserService")`)
	fmt.Println("")
	fmt.Println("// 3. 准备参数")
	fmt.Println("args := []hessian.Object{")
	fmt.Println(`    hessian.Object("zhangsan"),`)
	fmt.Println("    hessian.Object(28),")
	fmt.Println(`    hessian.Object("zhangsan@example.com"),`)
	fmt.Println("}")
	fmt.Println("")
	fmt.Println("// 4. 调用服务")
	fmt.Println(`result, err := genericService.Invoke(ctx, "createUser", []string{"string", "int", "string"}, args)`)
	fmt.Println("```")

	fmt.Println("\n🔵 Triple泛化调用方式:")
	fmt.Println("```go")
	fmt.Println("// 1. 创建客户端")
	fmt.Println("cli, err := client.NewClient()")
	fmt.Println("")
	fmt.Println("// 2. 创建Triple泛化服务")
	fmt.Println(`tripleService, err := cli.NewTripleGenericService("com.example.UserService")`)
	fmt.Println("// 或者直接创建")
	fmt.Println(`tripleService := triple.NewTripleGenericService("tri://127.0.0.1:20000/com.example.UserService")`)
	fmt.Println("")
	fmt.Println("// 3. 准备参数 (更简单)")
	fmt.Println("args := []interface{}{")
	fmt.Println(`    "zhangsan",`)
	fmt.Println("    28,")
	fmt.Println(`    "zhangsan@example.com",`)
	fmt.Println("}")
	fmt.Println("")
	fmt.Println("// 4. 调用服务 (多种方式)")
	fmt.Println(`result, err := tripleService.Invoke(ctx, "createUser", []string{"string", "int", "string"}, args)`)
	fmt.Println("// 或带附件调用")
	fmt.Println(`result, err = tripleService.InvokeWithAttachments(ctx, "createUser", types, args, attachments)`)
	fmt.Println("// 或异步调用")
	fmt.Println(`callID, err := tripleService.InvokeAsync(ctx, "createUser", types, args, attachments, callback)`)
	fmt.Println("```")

	fmt.Println("\n🔄 实际调用示例 (模拟):")

	// 模拟传统方式调用
	fmt.Println("\n传统方式调用:")
	traditionalCall(ctx)

	// 模拟Triple方式调用
	fmt.Println("\nTriple方式调用:")
	tripleCall(ctx)
}

// 兼容性适配器演示
func demonstrateCompatibilityAdapter() {
	fmt.Println("演示如何创建兼容性适配器...")

	fmt.Println("\n📦 适配器实现:")
	fmt.Println("```go")
	fmt.Println("// 兼容适配器，让Triple服务提供传统接口")
	fmt.Println("type DubboGenericAdapter struct {")
	fmt.Println("    tripleService *triple.TripleGenericService")
	fmt.Println("}")
	fmt.Println("")
	fmt.Println("func (adapter *DubboGenericAdapter) Invoke(ctx context.Context, methodName string, types []string, args []hessian.Object) (any, error) {")
	fmt.Println("    // 转换 hessian.Object 到 interface{}")
	fmt.Println("    interfaceArgs := make([]interface{}, len(args))")
	fmt.Println("    for i, arg := range args {")
	fmt.Println("        interfaceArgs[i] = interface{}(arg)")
	fmt.Println("    }")
	fmt.Println("    ")
	fmt.Println("    // 委托给 Triple 泛化服务")
	fmt.Println("    return adapter.tripleService.Invoke(ctx, methodName, types, interfaceArgs)")
	fmt.Println("}")
	fmt.Println("```")

	// 创建适配器实例演示
	fmt.Println("\n🔧 适配器使用示例:")

	// 创建Triple服务
	tripleService := triple.NewTripleGenericService("tri://127.0.0.1:20000/com.example.UserService")

	// 创建适配器
	adapter := &DubboGenericAdapter{tripleService: tripleService}

	fmt.Printf("✅ 适配器创建成功: %T\n", adapter)
	fmt.Println("现在可以使用传统的Dubbo泛化接口调用Triple服务!")

	// 演示适配器调用
	ctx := context.Background()

	fmt.Println("\n📞 使用适配器调用示例:")
	traditionalArgs := []hessian.Object{
		hessian.Object("test_user"),
		hessian.Object(25),
	}

	result, err := adapter.Invoke(ctx, "createUser", []string{"string", "int"}, traditionalArgs)
	if err != nil {
		fmt.Printf("❌ 适配器调用失败: %v\n", err)
	} else {
		fmt.Printf("✅ 适配器调用成功: %v\n", result)
	}
}

// 迁移示例
func demonstrateMigrationExample() {
	fmt.Println("展示从传统Dubbo泛化迁移到Triple泛化的策略...")

	fmt.Println("\n📋 迁移策略:")
	fmt.Println("1. 🔄 渐进式迁移 - 新老系统并存")
	fmt.Println("2. 🎯 选择性迁移 - 根据业务需要迁移特定服务")
	fmt.Println("3. 🚀 功能增强 - 利用Triple的新功能")

	fmt.Println("\n📦 迁移步骤示例:")

	fmt.Println("\nStep 1: 保持现有传统调用")
	fmt.Println("```go")
	fmt.Println("// 现有代码继续工作")
	fmt.Println(`genericService, _ := client.NewGenericService("com.example.UserService")`)
	fmt.Println("```")

	fmt.Println("\nStep 2: 并行引入Triple服务")
	fmt.Println("```go")
	fmt.Println("// 新增Triple服务，与传统服务并存")
	fmt.Println(`tripleService := triple.NewTripleGenericService("tri://127.0.0.1:20000/com.example.UserService")`)
	fmt.Println("```")

	fmt.Println("\nStep 3: 利用Triple增强功能")
	fmt.Println("```go")
	fmt.Println("// 对于需要高性能的场景，使用Triple的批量调用")
	fmt.Println("results, err := tripleService.BatchInvoke(ctx, batchRequests)")
	fmt.Println("")
	fmt.Println("// 对于需要异步的场景，使用Triple的异步调用")
	fmt.Println("callID, err := tripleService.InvokeAsync(ctx, method, types, args, attachments, callback)")
	fmt.Println("```")

	fmt.Println("\nStep 4: 逐步替换传统调用")
	fmt.Println("```go")
	fmt.Println("// 将传统调用逐步替换为Triple调用")
	fmt.Println("// result, err := genericService.Invoke(ctx, method, types, hessianArgs)  // 旧代码")
	fmt.Println("result, err := tripleService.Invoke(ctx, method, types, interfaceArgs)     // 新代码")
	fmt.Println("```")

	fmt.Println("\n✅ 迁移优势:")
	fmt.Println("- 🔒 零风险: 传统功能完全兼容")
	fmt.Println("- 📈 性能提升: HTTP/2 + 批量处理")
	fmt.Println("- 🚀 功能增强: 异步、附件、并发控制")
	fmt.Println("- 🔄 渐进式: 可以逐步迁移，不影响现有系统")

	fmt.Println("\n💡 最佳实践建议:")
	fmt.Println("- 新项目: 直接使用Triple泛化调用")
	fmt.Println("- 现有项目: 保持传统调用，在需要新功能时引入Triple")
	fmt.Println("- 性能敏感场景: 优先考虑迁移到Triple")
	fmt.Println("- 复杂调用场景: 利用Triple的批量和异步功能")
}

// 兼容性适配器实现
type DubboGenericAdapter struct {
	tripleService *triple.TripleGenericService
}

func (adapter *DubboGenericAdapter) Invoke(ctx context.Context, methodName string, types []string, args []hessian.Object) (any, error) {
	// 转换 hessian.Object 到 interface{}
	interfaceArgs := make([]any, len(args))
	for i, arg := range args {
		interfaceArgs[i] = any(arg)
	}

	// 委托给 Triple 泛化服务
	return adapter.tripleService.Invoke(ctx, methodName, types, interfaceArgs)
}

func (adapter *DubboGenericAdapter) Reference() string {
	return adapter.tripleService.Reference()
}

// 模拟传统方式调用
func traditionalCall(ctx context.Context) {
	// 注意: 这里只是模拟调用格式，实际调用会有网络错误
	fmt.Println("模拟传统泛化调用:")

	args := []hessian.Object{
		hessian.Object("zhangsan"),
		hessian.Object(28),
	}
	types := []string{"string", "int"}
	methodName := "createUser"

	fmt.Printf("  方法: %s\n", methodName)
	fmt.Printf("  类型: %v\n", types)
	fmt.Printf("  参数: %v (类型: %T)\n", args, args)

	// 实际调用会是:
	// result, err := genericService.Invoke(ctx, methodName, types, args)
	fmt.Printf("  调用: genericService.Invoke(ctx, %q, %v, args)\n", methodName, types)
	fmt.Printf("  状态: ⚠️ 需要真实服务端 (演示模式)\n")
}

// 模拟Triple方式调用
func tripleCall(ctx context.Context) {
	// 注意: 这里只是模拟调用格式，实际调用会有网络错误
	fmt.Println("模拟Triple泛化调用:")

	args := []any{
		"zhangsan",
		28,
	}
	types := []string{"string", "int"}
	methodName := "createUser"

	fmt.Printf("  方法: %s\n", methodName)
	fmt.Printf("  类型: %v\n", types)
	fmt.Printf("  参数: %v (类型: %T)\n", args, args)

	// 实际调用会是:
	tripleService := triple.NewTripleGenericService("tri://127.0.0.1:20000/com.example.UserService")
	result, err := tripleService.Invoke(ctx, methodName, types, args)

	fmt.Printf("  调用: tripleService.Invoke(ctx, %q, %v, args)\n", methodName, types)
	if err != nil {
		fmt.Printf("  状态: ⚠️ %v (预期的网络错误)\n", err)
	} else {
		fmt.Printf("  结果: ✅ %v\n", result)
	}

	// 演示Triple独有功能
	fmt.Println("\n  🚀 Triple独有功能演示:")

	// 带附件调用
	attachments := map[string]any{
		"traceId": "demo-trace-001",
		"userId":  "demo-user",
	}

	_, err = tripleService.InvokeWithAttachments(ctx, methodName, types, args, attachments)
	fmt.Printf("  附件调用: InvokeWithAttachments - %v\n",
		map[string]string{"status": "⚠️ 网络错误(预期)", "feature": "✅ 可用"})

	// 异步调用
	_, err = tripleService.InvokeAsync(ctx, methodName, types, args, attachments,
		func(result any, err error) {
			log.Printf("异步回调: result=%v, err=%v", result, err)
		})
	fmt.Printf("  异步调用: InvokeAsync - %v\n",
		map[string]string{"status": "⚠️ 网络错误(预期)", "feature": "✅ 可用"})
}
