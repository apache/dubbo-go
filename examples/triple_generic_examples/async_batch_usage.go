//go:build example_async_batch
// +build example_async_batch

/*
 * Triple Generic Call Async and Batch Usage Example
 */

package main

import (
	"context"
	"fmt"
	"sync"
	"time"

	"dubbo.apache.org/dubbo-go/v3/protocol/triple"
)

func main() {
	fmt.Println("🚀 Triple Generic Call Async and Batch Example")
	fmt.Println("=================================")

	tripleGS := triple.NewTripleGenericService("tri://127.0.0.1:20000/com.example.OrderService?serialization=hessian2")
	ctx := context.Background()

	// 示例1: 异步调用
	fmt.Println("\n1. ⏰ Asynchronous Call Example")
	asyncExample(tripleGS, ctx)

	// 示例2: 批量同步调用
	fmt.Println("\n2. 📦 Batch Synchronous Call Example")
	batchSyncExample(tripleGS, ctx)

	// 示例3: 批量异步调用
	fmt.Println("\n3. ⚡ Batch Asynchronous Call Example")
	batchAsyncExample(tripleGS, ctx)

	// 示例4: 高级批量调用配置
	fmt.Println("\n4. ⚙️ Advanced Batch Call Configuration")
	advancedBatchExample(tripleGS, ctx)

	// 示例5: 异步调用管理
	fmt.Println("\n5. 🎛️ Asynchronous Call Management Example")
	asyncManagementExample(tripleGS, ctx)

	fmt.Println("\n🎉 Async and batch examples completed!")
}

func asyncExample(tripleGS *triple.TripleGenericService, ctx context.Context) {
	fmt.Println("Starting asynchronous call...")

	// 创建等待组来同步异步调用
	var wg sync.WaitGroup
	results := make(chan string, 3)

	// 异步调用1: 创建订单
	wg.Add(1)
	callID1, err := tripleGS.InvokeAsync(ctx, "createOrder",
		[]string{"map"},
		[]interface{}{
			map[string]interface{}{
				"userId":    int64(1001),
				"productId": int64(2001),
				"quantity":  2,
				"amount":    199.99,
			},
		},
		map[string]interface{}{"priority": "high"},
		func(result interface{}, err error) {
			defer wg.Done()
			if err != nil {
				results <- fmt.Sprintf("Failed to create order: %v", err)
			} else {
				results <- fmt.Sprintf("Order created successfully: %v", result)
			}
		})

	if err != nil {
		fmt.Printf("❌ Failed to start asynchronous call 1: %v\n", err)
		wg.Done()
	} else {
		fmt.Printf("🚀 Asynchronous call 1 started, ID: %s\n", callID1)
	}

	// 异步调用2: 查询库存
	wg.Add(1)
	callID2, err := tripleGS.InvokeAsync(ctx, "checkInventory",
		[]string{"int64", "int32"},
		[]interface{}{int64(2001), int32(2)},
		nil,
		func(result interface{}, err error) {
			defer wg.Done()
			if err != nil {
				results <- fmt.Sprintf("Failed to check inventory: %v", err)
			} else {
				results <- fmt.Sprintf("Inventory checked successfully: %v", result)
			}
		})

	if err != nil {
		fmt.Printf("❌ Failed to start asynchronous call 2: %v\n", err)
		wg.Done()
	} else {
		fmt.Printf("🚀 Asynchronous call 2 started, ID: %s\n", callID2)
	}

	// 异步调用3: 计算运费
	wg.Add(1)
	callID3, err := tripleGS.InvokeAsyncWithTimeout(ctx, "calculateShipping",
		[]string{"string", "float64"},
		[]interface{}{"北京市朝阳区", 199.99},
		map[string]interface{}{"expressType": "standard"},
		func(result interface{}, err error) {
			defer wg.Done()
			if err != nil {
				results <- fmt.Sprintf("Failed to calculate shipping: %v", err)
			} else {
				results <- fmt.Sprintf("Shipping calculated successfully: %v", result)
			}
		},
		3*time.Second) // 3秒超时

	if err != nil {
		fmt.Printf("❌ Failed to start asynchronous call 3: %v\n", err)
		wg.Done()
	} else {
		fmt.Printf("🚀 Asynchronous call 3 started, ID: %s\n", callID3)
	}

	// 等待所有异步调用完成
	go func() {
		wg.Wait()
		close(results)
	}()

	// 收集结果
	fmt.Println("Waiting for asynchronous call results...")
	timeout := time.After(5 * time.Second)
	resultCount := 0

	for {
		select {
		case result, ok := <-results:
			if !ok {
				fmt.Printf("✅ All asynchronous calls completed (total %d calls)\n", resultCount)
				return
			}
			fmt.Printf("📝 %s\n", result)
			resultCount++
		case <-timeout:
			fmt.Println("⏰ Waiting for asynchronous call timeout")
			return
		}
	}
}

func batchSyncExample(tripleGS *triple.TripleGenericService, ctx context.Context) {
	fmt.Println("Preparing batch synchronous call...")

	// 准备批量订单处理请求
	invocations := []triple.TripleInvocationRequest{
		{
			MethodName: "processPayment",
			Types:      []string{"string", "float64"},
			Args:       []interface{}{"ORDER_001", 299.99},
			Attachments: map[string]interface{}{
				"orderId":     "ORDER_001",
				"paymentType": "alipay",
			},
		},
		{
			MethodName: "processPayment",
			Types:      []string{"string", "float64"},
			Args:       []interface{}{"ORDER_002", 159.99},
			Attachments: map[string]interface{}{
				"orderId":     "ORDER_002",
				"paymentType": "wechat",
			},
		},
		{
			MethodName: "processPayment",
			Types:      []string{"string", "float64"},
			Args:       []interface{}{"ORDER_003", 89.99},
			Attachments: map[string]interface{}{
				"orderId":     "ORDER_003",
				"paymentType": "bank_card",
			},
		},
		{
			MethodName: "updateOrderStatus",
			Types:      []string{"string", "string"},
			Args:       []interface{}{"ORDER_001", "paid"},
			Attachments: map[string]interface{}{
				"orderId": "ORDER_001",
				"action":  "status_update",
			},
		},
		{
			MethodName: "sendNotification",
			Types:      []string{"int64", "string", "map"},
			Args: []interface{}{
				int64(1001),
				"payment_success",
				map[string]interface{}{
					"orderId": "ORDER_001",
					"amount":  299.99,
				},
			},
			Attachments: map[string]interface{}{
				"notificationType": "sms_email",
			},
		},
	}

	start := time.Now()
	results, err := tripleGS.BatchInvoke(ctx, invocations)
	duration := time.Since(start)

	if err != nil {
		fmt.Printf("❌ Batch call failed: %v\n", err)
		return
	}

	fmt.Printf("✅ Batch call completed, duration: %v\n", duration)
	fmt.Printf("📊 Processed %d requests, results are as follows:\n", len(results))

	for i, result := range results {
		if result.Error != nil {
			fmt.Printf("  [%d] ❌ %s failed: %v\n", i+1, invocations[i].MethodName, result.Error)
		} else {
			fmt.Printf("  [%d] ✅ %s succeeded: %v\n", i+1, invocations[i].MethodName, result.Result)
		}
	}
}

func batchAsyncExample(tripleGS *triple.TripleGenericService, ctx context.Context) {
	fmt.Println("Starting batch asynchronous call...")

	// 准备用户通知批量请求
	var invocations []triple.TripleInvocationRequest
	userIDs := []int64{1001, 1002, 1003, 1004, 1005}

	for i, userID := range userIDs {
		invocations = append(invocations, triple.TripleInvocationRequest{
			MethodName: "sendPromotionNotification",
			Types:      []string{"int64", "map"},
			Args: []interface{}{
				userID,
				map[string]interface{}{
					"title":   "Double 11 Promotion",
					"content": "80% off on all products, limited to 3 days!",
					"type":    "promotion",
				},
			},
			Attachments: map[string]interface{}{
				"userId":     userID,
				"campaignId": "PROMO_2023_1111",
				"batchIndex": i,
				"sendTime":   time.Now().Format("2006-01-02 15:04:05"),
			},
		})
	}

	// 批量异步调用结果处理
	resultChan := make(chan string, len(invocations))
	var wg sync.WaitGroup

	callIDs, err := tripleGS.InvokeAsyncBatch(ctx, invocations,
		func(results []triple.TripleAsyncResult) {
			fmt.Printf("📬 Batch asynchronous call callback triggered, received %d results\n", len(results))

			for _, result := range results {
				wg.Add(1)
				go func(r triple.TripleAsyncResult) {
					defer wg.Done()
					if r.Error != nil {
						resultChan <- fmt.Sprintf("User notification failed: %v", r.Error)
					} else {
						resultChan <- fmt.Sprintf("User notification succeeded: %v", r.Result)
					}
				}(result)
			}
		})

	if err != nil {
		fmt.Printf("❌ Failed to start batch asynchronous call: %v\n", err)
		return
	}

	fmt.Printf("🚀 Batch asynchronous call started, total %d calls\n", len(callIDs))
	for i, callID := range callIDs {
		fmt.Printf("  Call %d ID: %s\n", i+1, callID)
	}

	// 等待结果
	go func() {
		wg.Wait()
		close(resultChan)
	}()

	fmt.Println("Waiting for batch asynchronous call results...")
	timeout := time.After(10 * time.Second)
	resultCount := 0

	for {
		select {
		case result, ok := <-resultChan:
			if !ok {
				fmt.Printf("✅ All batch asynchronous calls completed (total %d results)\n", resultCount)
				return
			}
			fmt.Printf("📝 %s\n", result)
			resultCount++
		case <-timeout:
			fmt.Println("⏰ Waiting for batch asynchronous call timeout")
			return
		}
	}
}

func advancedBatchExample(tripleGS *triple.TripleGenericService, ctx context.Context) {
	fmt.Println("Testing advanced batch call configuration...")

	// 准备大量数据处理请求
	var invocations []triple.TripleInvocationRequest
	for i := 0; i < 20; i++ {
		invocations = append(invocations, triple.TripleInvocationRequest{
			MethodName: "processDataAnalytics",
			Types:      []string{"int32", "map"},
			Args: []interface{}{
				int32(i),
				map[string]interface{}{
					"dataId":    fmt.Sprintf("DATA_%03d", i),
					"timestamp": time.Now().Unix(),
					"metrics": map[string]interface{}{
						"cpu":    fmt.Sprintf("%.2f", 10.0+float64(i)*2.5),
						"memory": fmt.Sprintf("%.2f", 30.0+float64(i)*1.8),
						"disk":   fmt.Sprintf("%.2f", 50.0+float64(i)*0.9),
					},
				},
			},
			Attachments: map[string]interface{}{
				"dataIndex": i,
				"source":    "monitoring_system",
			},
		})
	}

	// 测试不同的批量调用配置
	testConfigs := []struct {
		name    string
		options triple.BatchInvokeOptions
	}{
		{
			name: "Low Concurrency Stable Mode",
			options: triple.BatchInvokeOptions{
				MaxConcurrency: 3,
				FailFast:       false,
			},
		},
		{
			name: "High Concurrency Fast Mode",
			options: triple.BatchInvokeOptions{
				MaxConcurrency: 10,
				FailFast:       false,
			},
		},
		{
			name: "Fail Fast Mode",
			options: triple.BatchInvokeOptions{
				MaxConcurrency: 5,
				FailFast:       true,
			},
		},
	}

	for _, config := range testConfigs {
		fmt.Printf("\n🔧 Testing configuration: %s\n", config.name)
		fmt.Printf("   Concurrency: %d, Fail Fast: %v\n",
			config.options.MaxConcurrency, config.options.FailFast)

		start := time.Now()
		results, err := tripleGS.BatchInvokeWithOptions(ctx, invocations, config.options)
		duration := time.Since(start)

		if err != nil {
			fmt.Printf("❌ Batch call failed: %v\n", err)
			continue
		}

		successCount := 0
		errorCount := 0
		for _, result := range results {
			if result.Error == nil {
				successCount++
			} else {
				errorCount++
			}
		}

		fmt.Printf("📊 Result statistics:\n")
		fmt.Printf("   Total: %d, Success: %d, Failure: %d\n",
			len(results), successCount, errorCount)
		fmt.Printf("   Duration: %v, Average: %v\n",
			duration, duration/time.Duration(len(results)))
	}
}

func asyncManagementExample(tripleGS *triple.TripleGenericService, ctx context.Context) {
	fmt.Println("Demonstrating asynchronous call management features...")

	// 启动几个长时间运行的异步调用
	var callIDs []string

	for i := 0; i < 3; i++ {
		callID, err := tripleGS.InvokeAsync(ctx, "longRunningTask",
			[]string{"int32", "string"},
			[]interface{}{int32(i), fmt.Sprintf("task_%d", i)},
			map[string]interface{}{"taskId": i},
			func(result interface{}, err error) {
				if err != nil {
					fmt.Printf("⚠️ 长任务 %d 完成但有错误: %v\n", i, err)
				} else {
					fmt.Printf("✅ 长任务 %d 成功完成: %v\n", i, result)
				}
			})

		if err != nil {
			fmt.Printf("❌ Failed to start long task %d: %v\n", i, err)
		} else {
			callIDs = append(callIDs, callID)
			fmt.Printf("🚀 Long task %d started, ID: %s\n", i, callID)
		}
	}

	if len(callIDs) == 0 {
		fmt.Println("❌ No asynchronous calls successfully started")
		return
	}

	// 等待1秒后查看状态
	time.Sleep(1 * time.Second)

	// 查看活跃调用
	fmt.Println("\n📋 View active asynchronous calls:")
	activeCalls := tripleGS.GetActiveAsyncCalls()
	fmt.Printf("Current number of active calls: %d\n", len(activeCalls))

	for callID, asyncCall := range activeCalls {
		fmt.Printf("  Call ID: %s, Method: %s, Start Time: %v\n",
			callID, asyncCall.MethodName, asyncCall.StartTime.Format("15:04:05"))
	}

	// 取消第一个调用
	if len(callIDs) > 0 {
		fmt.Printf("\n🛑 Cancel the first asynchronous call: %s\n", callIDs[0])
		cancelled := tripleGS.CancelAsyncCall(callIDs[0])
		if cancelled {
			fmt.Println("✅ Call successfully cancelled")
		} else {
			fmt.Println("❌ Call cancellation failed")
		}
	}

	// 等待其中一个调用完成
	if len(callIDs) > 1 {
		fmt.Printf("\n⏳ Waiting for call completion: %s\n", callIDs[1])
		result, err := tripleGS.WaitForAsyncCall(callIDs[1], 3*time.Second)
		if err != nil {
			fmt.Printf("❌ Waiting for call failed: %v\n", err)
		} else {
			fmt.Printf("✅ Call completed: %v\n", result)
		}
	}

	// 最终状态检查
	time.Sleep(500 * time.Millisecond)
	finalActiveCalls := tripleGS.GetActiveAsyncCalls()
	fmt.Printf("\n📊 Final number of active calls: %d\n", len(finalActiveCalls))
}
