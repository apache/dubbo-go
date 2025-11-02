/*
 * Triple 泛化调用异步和批量使用示例
 */

package main

import (
	"context"
	"fmt"
	"sync"
	"time"
)

import (
	"dubbo.apache.org/dubbo-go/v3/protocol/triple"
)

func main() {
	fmt.Println("🚀 Triple 泛化调用异步和批量示例")
	fmt.Println("=================================")

	tripleGS := triple.NewTripleGenericService("tri://127.0.0.1:20000/com.example.OrderService?serialization=hessian2")
	ctx := context.Background()

	// 示例1: 异步调用
	fmt.Println("\n1. ⏰ 异步调用示例")
	asyncExample(tripleGS, ctx)

	// 示例2: 批量同步调用
	fmt.Println("\n2. 📦 批量同步调用示例")
	batchSyncExample(tripleGS, ctx)

	// 示例3: 批量异步调用
	fmt.Println("\n3. ⚡ 批量异步调用示例")
	batchAsyncExample(tripleGS, ctx)

	// 示例4: 高级批量调用配置
	fmt.Println("\n4. ⚙️ 高级批量调用配置")
	advancedBatchExample(tripleGS, ctx)

	// 示例5: 异步调用管理
	fmt.Println("\n5. 🎛️ 异步调用管理示例")
	asyncManagementExample(tripleGS, ctx)

	fmt.Println("\n🎉 异步和批量示例完成!")
}

func asyncExample(tripleGS *triple.TripleGenericService, ctx context.Context) {
	fmt.Println("启动异步调用...")

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
				results <- fmt.Sprintf("创建订单失败: %v", err)
			} else {
				results <- fmt.Sprintf("创建订单成功: %v", result)
			}
		})

	if err != nil {
		fmt.Printf("❌ 启动异步调用1失败: %v\n", err)
		wg.Done()
	} else {
		fmt.Printf("🚀 异步调用1已启动, ID: %s\n", callID1)
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
				results <- fmt.Sprintf("查询库存失败: %v", err)
			} else {
				results <- fmt.Sprintf("查询库存成功: %v", result)
			}
		})

	if err != nil {
		fmt.Printf("❌ 启动异步调用2失败: %v\n", err)
		wg.Done()
	} else {
		fmt.Printf("🚀 异步调用2已启动, ID: %s\n", callID2)
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
				results <- fmt.Sprintf("计算运费失败: %v", err)
			} else {
				results <- fmt.Sprintf("计算运费成功: %v", result)
			}
		},
		3*time.Second) // 3秒超时

	if err != nil {
		fmt.Printf("❌ 启动异步调用3失败: %v\n", err)
		wg.Done()
	} else {
		fmt.Printf("🚀 异步调用3已启动, ID: %s\n", callID3)
	}

	// 等待所有异步调用完成
	go func() {
		wg.Wait()
		close(results)
	}()

	// 收集结果
	fmt.Println("等待异步调用结果...")
	timeout := time.After(5 * time.Second)
	resultCount := 0

	for {
		select {
		case result, ok := <-results:
			if !ok {
				fmt.Printf("✅ 所有异步调用完成 (共 %d 个)\n", resultCount)
				return
			}
			fmt.Printf("📝 %s\n", result)
			resultCount++
		case <-timeout:
			fmt.Println("⏰ 等待异步调用超时")
			return
		}
	}
}

func batchSyncExample(tripleGS *triple.TripleGenericService, ctx context.Context) {
	fmt.Println("准备批量同步调用...")

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
		fmt.Printf("❌ 批量调用失败: %v\n", err)
		return
	}

	fmt.Printf("✅ 批量调用完成，耗时: %v\n", duration)
	fmt.Printf("📊 处理了 %d 个请求，结果如下:\n", len(results))

	for i, result := range results {
		if result.Error != nil {
			fmt.Printf("  [%d] ❌ %s 失败: %v\n", i+1, invocations[i].MethodName, result.Error)
		} else {
			fmt.Printf("  [%d] ✅ %s 成功: %v\n", i+1, invocations[i].MethodName, result.Result)
		}
	}
}

func batchAsyncExample(tripleGS *triple.TripleGenericService, ctx context.Context) {
	fmt.Println("启动批量异步调用...")

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
					"title":   "双11大促销",
					"content": "全场商品8折优惠，限时3天！",
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
			fmt.Printf("📬 批量异步调用回调触发，收到 %d 个结果\n", len(results))

			for _, result := range results {
				wg.Add(1)
				go func(r triple.TripleAsyncResult) {
					defer wg.Done()
					if r.Error != nil {
						resultChan <- fmt.Sprintf("用户通知失败: %v", r.Error)
					} else {
						resultChan <- fmt.Sprintf("用户通知成功: %v", r.Result)
					}
				}(result)
			}
		})

	if err != nil {
		fmt.Printf("❌ 启动批量异步调用失败: %v\n", err)
		return
	}

	fmt.Printf("🚀 批量异步调用已启动，共 %d 个调用\n", len(callIDs))
	for i, callID := range callIDs {
		fmt.Printf("  调用 %d ID: %s\n", i+1, callID)
	}

	// 等待结果
	go func() {
		wg.Wait()
		close(resultChan)
	}()

	fmt.Println("等待批量异步调用结果...")
	timeout := time.After(10 * time.Second)
	resultCount := 0

	for {
		select {
		case result, ok := <-resultChan:
			if !ok {
				fmt.Printf("✅ 批量异步调用全部完成 (共 %d 个结果)\n", resultCount)
				return
			}
			fmt.Printf("📝 %s\n", result)
			resultCount++
		case <-timeout:
			fmt.Println("⏰ 等待批量异步调用超时")
			return
		}
	}
}

func advancedBatchExample(tripleGS *triple.TripleGenericService, ctx context.Context) {
	fmt.Println("测试高级批量调用配置...")

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
			name: "低并发稳定模式",
			options: triple.BatchInvokeOptions{
				MaxConcurrency: 3,
				FailFast:       false,
			},
		},
		{
			name: "高并发快速模式",
			options: triple.BatchInvokeOptions{
				MaxConcurrency: 10,
				FailFast:       false,
			},
		},
		{
			name: "快速失败模式",
			options: triple.BatchInvokeOptions{
				MaxConcurrency: 5,
				FailFast:       true,
			},
		},
	}

	for _, config := range testConfigs {
		fmt.Printf("\n🔧 测试配置: %s\n", config.name)
		fmt.Printf("   并发数: %d, 快速失败: %v\n",
			config.options.MaxConcurrency, config.options.FailFast)

		start := time.Now()
		results, err := tripleGS.BatchInvokeWithOptions(ctx, invocations, config.options)
		duration := time.Since(start)

		if err != nil {
			fmt.Printf("❌ 批量调用失败: %v\n", err)
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

		fmt.Printf("📊 结果统计:\n")
		fmt.Printf("   总数: %d, 成功: %d, 失败: %d\n",
			len(results), successCount, errorCount)
		fmt.Printf("   耗时: %v, 平均: %v\n",
			duration, duration/time.Duration(len(results)))
	}
}

func asyncManagementExample(tripleGS *triple.TripleGenericService, ctx context.Context) {
	fmt.Println("演示异步调用管理功能...")

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
			fmt.Printf("❌ 启动长任务 %d 失败: %v\n", i, err)
		} else {
			callIDs = append(callIDs, callID)
			fmt.Printf("🚀 长任务 %d 已启动, ID: %s\n", i, callID)
		}
	}

	if len(callIDs) == 0 {
		fmt.Println("❌ 没有成功启动的异步调用")
		return
	}

	// 等待1秒后查看状态
	time.Sleep(1 * time.Second)

	// 查看活跃调用
	fmt.Println("\n📋 查看活跃的异步调用:")
	activeCalls := tripleGS.GetActiveAsyncCalls()
	fmt.Printf("当前活跃调用数量: %d\n", len(activeCalls))

	for callID, asyncCall := range activeCalls {
		fmt.Printf("  调用ID: %s, 方法: %s, 开始时间: %v\n",
			callID, asyncCall.MethodName, asyncCall.StartTime.Format("15:04:05"))
	}

	// 取消第一个调用
	if len(callIDs) > 0 {
		fmt.Printf("\n🛑 取消第一个异步调用: %s\n", callIDs[0])
		cancelled := tripleGS.CancelAsyncCall(callIDs[0])
		if cancelled {
			fmt.Println("✅ 调用已成功取消")
		} else {
			fmt.Println("❌ 调用取消失败")
		}
	}

	// 等待其中一个调用完成
	if len(callIDs) > 1 {
		fmt.Printf("\n⏳ 等待调用完成: %s\n", callIDs[1])
		result, err := tripleGS.WaitForAsyncCall(callIDs[1], 3*time.Second)
		if err != nil {
			fmt.Printf("❌ 等待调用失败: %v\n", err)
		} else {
			fmt.Printf("✅ 调用完成: %v\n", result)
		}
	}

	// 最终状态检查
	time.Sleep(500 * time.Millisecond)
	finalActiveCalls := tripleGS.GetActiveAsyncCalls()
	fmt.Printf("\n📊 最终活跃调用数量: %d\n", len(finalActiveCalls))
}

