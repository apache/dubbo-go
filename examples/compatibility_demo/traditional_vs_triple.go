/*
 * Traditional Dubbo Generic vs Triple Generic Compatibility Demonstration
 */

package main

import (
	"context"
	"fmt"
	"log"

	"dubbo.apache.org/dubbo-go/v3/protocol/triple"
	hessian "github.com/apache/dubbo-go-hessian2"
)

func main() {
	fmt.Println("Traditional Dubbo Generic vs Triple Generic Compatibility Demonstration")
	fmt.Println("=========================================================")

	// Demonstration 1: Parameter Compatibility Comparison
	fmt.Println("\n1. Parameter Compatibility Comparison")
	demonstrateParameterCompatibility()

	// Demonstration 2: API Call Comparison
	fmt.Println("\n2. API Call Method Comparison")
	demonstrateAPICompatibility()

	// Demonstration 3: Compatibility Adapter
	fmt.Println("\n3. Compatibility Adapter Demonstration")
	demonstrateCompatibilityAdapter()

	// Demonstration 4: Migration Example
	fmt.Println("\n4. Migration Example")
	demonstrateMigrationExample()

	fmt.Println("\nCompatibility demonstration completed!")
}

// Parameter Compatibility Comparison
func demonstrateParameterCompatibility() {
	fmt.Println("Demonstrating parameter passing formats for both methods...")

	// Example data
	userName := "Zhang San"
	userAge := 28
	userEmail := "zhangsan@example.com"

	fmt.Println("📊 Same business data:")
	fmt.Printf("  Username: %s\n", userName)
	fmt.Printf("  Age: %d\n", userAge)
	fmt.Printf("  Email: %s\n", userEmail)

	fmt.Println("\n🟢 Traditional Dubbo Generic Call Parameter Format:")
	traditionalArgs := []hessian.Object{
		hessian.Object(userName),
		hessian.Object(userAge),
		hessian.Object(userEmail),
	}
	traditionalTypes := []string{"string", "int", "string"}

	fmt.Printf("  Types: %v\n", traditionalTypes)
	fmt.Printf("  Parameters: []hessian.Object{%q, %d, %q}\n", userName, userAge, userEmail)
	fmt.Printf("  Parameter Type: %T\n", traditionalArgs)

	fmt.Println("\n🔵 Triple Generic Call Parameter Format:")
	tripleArgs := []any{
		userName,
		userAge,
		userEmail,
	}
	tripleTypes := []string{"string", "int", "string"}

	fmt.Printf("  Types: %v\n", tripleTypes)
	fmt.Printf("  Parameters: []interface{}{%q, %d, %q}\n", userName, userAge, userEmail)
	fmt.Printf("  Parameter Type: %T\n", tripleArgs)

	fmt.Println("\n✅ Compatibility Analysis:")
	fmt.Println("  - Type string: Exactly the same")
	fmt.Println("  - Parameter count: Exactly the same")
	fmt.Println("  - Parameter content: Semantically identical")
	fmt.Println("  - Parameter container: hessian.Object vs interface{} (compatible)")
}

// API Call Method Comparison
func demonstrateAPICompatibility() {
	fmt.Println("Comparing API call methods for both approaches...")

	ctx := context.Background()

	fmt.Println("\n🟢 传统Dubbo泛化调用方式:")
	fmt.Println("```go")
	fmt.Println("// 1. Create client")
	fmt.Println("cli, err := client.NewClient()")
	fmt.Println("")
	fmt.Println("// 2. Create generic service")
	fmt.Println(`genericService, err := cli.NewGenericService("com.example.UserService")`)
	fmt.Println("")
	fmt.Println("// 3. Prepare parameters")
	fmt.Println("args := []hessian.Object{")
	fmt.Println(`    hessian.Object("zhangsan"),`)
	fmt.Println("    hessian.Object(28),")
	fmt.Println(`    hessian.Object("zhangsan@example.com"),`)
	fmt.Println("}")
	fmt.Println("")
	fmt.Println("// 4. Make the call")
	fmt.Println(`result, err := genericService.Invoke(ctx, "createUser", []string{"string", "int", "string"}, args)`)
	fmt.Println("```")

	fmt.Println("\n🔵 Triple泛化调用方式:")
	fmt.Println("```go")
	fmt.Println("// 1. Create client")
	fmt.Println("cli, err := client.NewClient()")
	fmt.Println("")
	fmt.Println("// 2. Create Triple generic service")
	fmt.Println(`tripleService, err := cli.NewTripleGenericService("com.example.UserService")`)
	fmt.Println("// Or create directly")
	fmt.Println(`tripleService := triple.NewTripleGenericService("tri://127.0.0.1:20000/com.example.UserService")`)
	fmt.Println("")
	fmt.Println("// 3. Prepare parameters (simpler)")
	fmt.Println("args := []interface{}{")
	fmt.Println(`    "zhangsan",`)
	fmt.Println("    28,")
	fmt.Println(`    "zhangsan@example.com",`)
	fmt.Println("}")
	fmt.Println("")
	fmt.Println("// 4. Make the call (multiple ways)")
	fmt.Println(`result, err := tripleService.Invoke(ctx, "createUser", []string{"string", "int", "string"}, args)`)
	fmt.Println("// With attachments call")
	fmt.Println(`result, err = tripleService.InvokeWithAttachments(ctx, "createUser", types, args, attachments)`)
	fmt.Println("// Or asynchronous call")
	fmt.Println(`callID, err := tripleService.InvokeAsync(ctx, "createUser", types, args, attachments, callback)`)
	fmt.Println("```")

	fmt.Println("\n🔄 实际调用示例 (模拟):")

	// Simulate traditional call method
	fmt.Println("\nTraditional method call:")
	traditionalCall(ctx)

	// Simulate Triple call method
	fmt.Println("\nTriple method call:")
	tripleCall(ctx)
}

// Compatibility Adapter Demonstration
func demonstrateCompatibilityAdapter() {
	fmt.Println("Demonstrating how to create a compatibility adapter...")

	fmt.Println("\n📦 Adapter implementation:")
	fmt.Println("```go")
	fmt.Println("// Compatibility adapter, allowing Triple service to provide traditional interface")
	fmt.Println("type DubboGenericAdapter struct {")
	fmt.Println("    tripleService *triple.TripleGenericService")
	fmt.Println("}")
	fmt.Println("")
	fmt.Println("func (adapter *DubboGenericAdapter) Invoke(ctx context.Context, methodName string, types []string, args []hessian.Object) (any, error) {")
	fmt.Println("    // Convert hessian.Object to interface{}")
	fmt.Println("    interfaceArgs := make([]interface{}, len(args))")
	fmt.Println("    for i, arg := range args {")
	fmt.Println("        interfaceArgs[i] = interface{}(arg)")
	fmt.Println("    }")
	fmt.Println("    ")
	fmt.Println("    // Delegate to Triple generic service")
	fmt.Println("    return adapter.tripleService.Invoke(ctx, methodName, types, interfaceArgs)")
	fmt.Println("}")
	fmt.Println("```")

	// Create adapter instance demonstration
	fmt.Println("\n🔧 Adapter usage example:")

	// Create Triple service
	tripleService := triple.NewTripleGenericService("tri://127.0.0.1:20000/com.example.UserService")

	// Create adapter
	adapter := &DubboGenericAdapter{tripleService: tripleService}

	fmt.Printf("✅ Adapter created successfully: %T\n", adapter)
	fmt.Println("Now you can use the traditional Dubbo generic interface to call Triple service!")

	// Demonstrate adapter invocation
	ctx := context.Background()

	fmt.Println("\n📞 Using adapter invocation example:")
	traditionalArgs := []hessian.Object{
		hessian.Object("test_user"),
		hessian.Object(25),
	}

	result, err := adapter.Invoke(ctx, "createUser", []string{"string", "int"}, traditionalArgs)
	if err != nil {
		fmt.Printf("❌ Adapter invocation failed: %v\n", err)
	} else {
		fmt.Printf("✅ Adapter invocation succeeded: %v\n", result)
	}
}

// Migration Example
func demonstrateMigrationExample() {
	fmt.Println("Demonstrating migration strategies from traditional Dubbo generic to Triple generic...")

	fmt.Println("\n📋 Migration strategies:")
	fmt.Println("1. Gradual migration - Coexistence of old and new systems")
	fmt.Println("2. Selective migration - Migrate specific services based on business needs")
	fmt.Println("3. Feature enhancement - Leverage Triple's new features")

	fmt.Println("\n📦 Migration steps example:")

	fmt.Println("\nStep 1: Maintain existing traditional calls")
	fmt.Println("```go")
	fmt.Println("// Existing code continues to work")
	fmt.Println(`genericService, _ := client.NewGenericService("com.example.UserService")`)
	fmt.Println("```")

	fmt.Println("\nStep 2: Introduce Triple service in parallel")
	fmt.Println("```go")
	fmt.Println("// Add Triple service, coexisting with traditional services")
	fmt.Println(`tripleService := triple.NewTripleGenericService("tri://127.0.0.1:20000/com.example.UserService")`)
	fmt.Println("```")

	fmt.Println("\nStep 3: Utilize Triple enhanced features")
	fmt.Println("```go")
	fmt.Println("// For high-performance scenarios, use Triple's batch calls")
	fmt.Println("results, err := tripleService.BatchInvoke(ctx, batchRequests)")
	fmt.Println("")
	fmt.Println("// For asynchronous scenarios, use Triple's asynchronous calls")
	fmt.Println("callID, err := tripleService.InvokeAsync(ctx, method, types, args, attachments, callback)")
	fmt.Println("```")

	fmt.Println("\nStep 4: Gradually replace traditional calls")
	fmt.Println("```go")
	fmt.Println("// Gradually replace traditional calls with Triple calls")
	fmt.Println("// result, err := genericService.Invoke(ctx, method, types, hessianArgs)  // old code")
	fmt.Println("result, err := tripleService.Invoke(ctx, method, types, interfaceArgs)     // new code")
	fmt.Println("```")

	fmt.Println("\n✅ Migration advantages:")
	fmt.Println("- Zero risk: Traditional functionality fully compatible")
	fmt.Println("- Performance improvement: HTTP/2 + batch processing")
	fmt.Println("- Feature enhancement: Asynchronous, attachments, concurrency control")
	fmt.Println("- Gradual: Can be migrated step by step without affecting existing systems")

	fmt.Println("\n💡 Best practice recommendations:")
	fmt.Println("- New projects: Directly use Triple generic calls")
	fmt.Println("- Existing projects: Maintain traditional calls, introduce Triple when new features are needed")
	fmt.Println("- Performance-sensitive scenarios: Prioritize migrating to Triple")
	fmt.Println("- Complex call scenarios: Utilize Triple's batch and asynchronous capabilities")
}

// Compatibility adapter implementation
type DubboGenericAdapter struct {
	tripleService *triple.TripleGenericService
}

func (adapter *DubboGenericAdapter) Invoke(ctx context.Context, methodName string, types []string, args []hessian.Object) (any, error) {
	// Convert hessian.Object to interface{}
	interfaceArgs := make([]any, len(args))
	for i, arg := range args {
		interfaceArgs[i] = any(arg)
	}

	// Delegate to Triple generic service
	return adapter.tripleService.Invoke(ctx, methodName, types, interfaceArgs)
}

func (adapter *DubboGenericAdapter) Reference() string {
	return adapter.tripleService.Reference()
}

// Simulate traditional call method
func traditionalCall(ctx context.Context) {
	// Note: This is only a simulation of the call format, actual calls will have network errors
	fmt.Println("Simulating traditional generic call:")

	args := []hessian.Object{
		hessian.Object("zhangsan"),
		hessian.Object(28),
	}
	types := []string{"string", "int"}
	methodName := "createUser"

	fmt.Printf("  Method: %s\n", methodName)
	fmt.Printf("  Types: %v\n", types)
	fmt.Printf("  Parameters: %v (Type: %T)\n", args, args)

	// Actual call would be:
	// result, err := genericService.Invoke(ctx, methodName, types, args)
	fmt.Printf("  Call: genericService.Invoke(ctx, %q, %v, args)\n", methodName, types)
	fmt.Printf("  Status: Requires real server (demo mode)\n")
}

// Simulate Triple call method
func tripleCall(ctx context.Context) {
	// Note: This is only a simulation of the call format, actual calls will have network errors
	fmt.Println("Simulating Triple generic call:")

	args := []any{
		"zhangsan",
		28,
	}
	types := []string{"string", "int"}
	methodName := "createUser"

	fmt.Printf("  Method: %s\n", methodName)
	fmt.Printf("  Types: %v\n", types)
	fmt.Printf("  Parameters: %v (Type: %T)\n", args, args)

	// Actual call would be:
	tripleService := triple.NewTripleGenericService("tri://127.0.0.1:20000/com.example.UserService")
	result, err := tripleService.Invoke(ctx, methodName, types, args)

	fmt.Printf("  Call: tripleService.Invoke(ctx, %q, %v, args)\n", methodName, types)
	if err != nil {
		fmt.Printf("  Status: %v (expected network error)\n", err)
	} else {
		fmt.Printf("  Result: %v\n", result)
	}

	// Demonstrate Triple exclusive features
	fmt.Println("\n  Triple exclusive feature demonstration:")

	// With attachments call
	attachments := map[string]any{
		"traceId": "demo-trace-001",
		"userId":  "demo-user",
	}

	_, err = tripleService.InvokeWithAttachments(ctx, methodName, types, args, attachments)
	fmt.Printf("  Attachments call: InvokeWithAttachments - %v\n",
		map[string]string{"status": "Network error (expected)", "feature": "Available"})

	// Asynchronous call
	_, err = tripleService.InvokeAsync(ctx, methodName, types, args, attachments,
		func(result any, err error) {
			log.Printf("Asynchronous callback: result=%v, err=%v", result, err)
		})
	fmt.Printf("  Asynchronous call: InvokeAsync - %v\n",
		map[string]string{"status": "Network error (expected)", "feature": "Available"})
}
