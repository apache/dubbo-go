/*
 * Triple 泛化调用压力测试示例
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
	fmt.Println("⚡ Triple 泛化调用压力测试")
	fmt.Println("==========================")

	tripleGS := triple.NewTripleGenericService("tri://127.0.0.1:20000/com.benchmark.TestService?serialization=hessian2")
	ctx := context.Background()

	// 测试1: 并发单个调用压力测试
	fmt.Println("\n1. 🔥 并发单个调用压力测试")
	concurrentSingleCallTest(tripleGS, ctx)

	// 测试2: 批量调用性能测试
	fmt.Println("\n2. 📦 批量调用性能测试")
	batchCallPerformanceTest(tripleGS, ctx)

	// 测试3: 异步调用并发测试
	fmt.Println("\n3. ⚡ 异步调用并发测试")
	asyncConcurrencyTest(tripleGS, ctx)

	// 测试4: 内存使用压力测试
	fmt.Println("\n4. 🧠 内存使用压力测试")
	memoryUsageTest(tripleGS, ctx)

	// 测试5: 长时间运行稳定性测试
	fmt.Println("\n5. ⏰ 长时间运行稳定性测试")
	longRunningStabilityTest(tripleGS, ctx)

	fmt.Println("\n🎉 压力测试完成!")
}

// 并发单个调用压力测试
func concurrentSingleCallTest(tripleGS *triple.TripleGenericService, ctx context.Context) {
	testCases := []struct {
		name        string
		concurrency int
		iterations  int
	}{
		{"低并发", 10, 100},
		{"中等并发", 50, 200},
		{"高并发", 100, 300},
		{"极高并发", 200, 500},
	}

	for _, tc := range testCases {
		fmt.Printf("\n🔧 测试: %s (并发数: %d, 迭代次数: %d)\n", tc.name, tc.concurrency, tc.iterations)

		var wg sync.WaitGroup
		var mu sync.Mutex
		results := make(map[string]int)
		results["success"] = 0
		results["error"] = 0

		start := time.Now()

		// 启动并发goroutines
		for i := 0; i < tc.concurrency; i++ {
			wg.Add(1)
			go func(workerID int) {
				defer wg.Done()

				for j := 0; j < tc.iterations/tc.concurrency; j++ {
					// 测试不同类型的方法调用
					methods := []struct {
						name  string
						types []string
						args  []interface{}
					}{
						{
							"simpleStringMethod",
							[]string{"string"},
							[]interface{}{fmt.Sprintf("worker_%d_call_%d", workerID, j)},
						},
						{
							"mathOperation",
							[]string{"int32", "int32", "string"},
							[]interface{}{int32(j), int32(workerID), "add"},
						},
						{
							"complexObjectMethod",
							[]string{"map"},
							[]interface{}{
								map[string]interface{}{
									"workerId":  workerID,
									"iteration": j,
									"timestamp": time.Now().Unix(),
									"data": map[string]interface{}{
										"type":  "stress_test",
										"value": fmt.Sprintf("test_data_%d_%d", workerID, j),
									},
								},
							},
						},
					}

					method := methods[j%len(methods)]

					_, err := tripleGS.InvokeWithAttachments(ctx, method.name, method.types, method.args,
						map[string]interface{}{
							"workerId":  workerID,
							"iteration": j,
							"testType":  "concurrent_stress",
						})

					mu.Lock()
					if err != nil {
						results["error"]++
					} else {
						results["success"]++
					}
					mu.Unlock()
				}
			}(i)
		}

		wg.Wait()
		duration := time.Since(start)

		totalCalls := results["success"] + results["error"]
		successRate := float64(results["success"]) / float64(totalCalls) * 100
		qps := float64(totalCalls) / duration.Seconds()

		fmt.Printf("📊 结果统计:\n")
		fmt.Printf("   总调用: %d, 成功: %d, 失败: %d\n", totalCalls, results["success"], results["error"])
		fmt.Printf("   成功率: %.2f%%, QPS: %.2f\n", successRate, qps)
		fmt.Printf("   耗时: %v, 平均延迟: %v\n", duration, duration/time.Duration(totalCalls))
	}
}

// 批量调用性能测试
func batchCallPerformanceTest(tripleGS *triple.TripleGenericService, ctx context.Context) {
	batchSizes := []int{10, 50, 100, 200, 500}
	concurrencyLevels := []int{1, 3, 5, 10}

	for _, batchSize := range batchSizes {
		for _, concurrency := range concurrencyLevels {
			fmt.Printf("\n🧪 批量大小: %d, 并发数: %d\n", batchSize, concurrency)

			// 生成批量调用请求
			var invocations []triple.TripleInvocationRequest
			for i := 0; i < batchSize; i++ {
				invocations = append(invocations, triple.TripleInvocationRequest{
					MethodName: "batchTestMethod",
					Types:      []string{"int32", "string", "map"},
					Args: []interface{}{
						int32(i),
						fmt.Sprintf("batch_item_%d", i),
						map[string]interface{}{
							"index":     i,
							"batchSize": batchSize,
							"testData":  fmt.Sprintf("test_batch_%d_%d", batchSize, i),
						},
					},
					Attachments: map[string]interface{}{
						"batchId":   fmt.Sprintf("batch_%d_%d", batchSize, concurrency),
						"itemIndex": i,
						"testType":  "batch_performance",
					},
				})
			}

			options := triple.BatchInvokeOptions{
				MaxConcurrency: concurrency,
				FailFast:       false,
			}

			start := time.Now()
			results, err := tripleGS.BatchInvokeWithOptions(ctx, invocations, options)
			duration := time.Since(start)

			if err != nil {
				fmt.Printf("❌ 批量调用失败: %v\n", err)
				continue
			}

			successCount := 0
			for _, result := range results {
				if result.Error == nil {
					successCount++
				}
			}

			throughput := float64(batchSize) / duration.Seconds()
			fmt.Printf("📈 性能指标:\n")
			fmt.Printf("   成功率: %d/%d (%.2f%%)\n", successCount, batchSize,
				float64(successCount)/float64(batchSize)*100)
			fmt.Printf("   吞吐量: %.2f calls/sec\n", throughput)
			fmt.Printf("   总耗时: %v, 平均耗时: %v\n", duration, duration/time.Duration(batchSize))
		}
	}
}

// 异步调用并发测试
func asyncConcurrencyTest(tripleGS *triple.TripleGenericService, ctx context.Context) {
	asyncCounts := []int{10, 50, 100, 200}

	for _, asyncCount := range asyncCounts {
		fmt.Printf("\n🔄 异步调用数量: %d\n", asyncCount)

		var wg sync.WaitGroup
		var mu sync.Mutex
		completedCalls := 0
		errorCalls := 0
		callIDs := make([]string, 0, asyncCount)

		start := time.Now()

		// 启动大量异步调用
		for i := 0; i < asyncCount; i++ {
			callID, err := tripleGS.InvokeAsync(ctx, "asyncTestMethod",
				[]string{"int32", "string", "bool"},
				[]interface{}{
					int32(i),
					fmt.Sprintf("async_test_%d", i),
					i%2 == 0,
				},
				map[string]interface{}{
					"asyncCallId": i,
					"testType":    "async_concurrency",
				},
				func(result interface{}, err error) {
					mu.Lock()
					defer mu.Unlock()
					if err != nil {
						errorCalls++
					} else {
						completedCalls++
					}
					wg.Done()
				})

			if err != nil {
				fmt.Printf("❌ 启动异步调用 %d 失败: %v\n", i, err)
				continue
			}

			mu.Lock()
			callIDs = append(callIDs, callID)
			mu.Unlock()
			wg.Add(1)
		}

		// 等待所有异步调用完成或超时
		done := make(chan struct{})
		go func() {
			wg.Wait()
			close(done)
		}()

		select {
		case <-done:
			duration := time.Since(start)
			fmt.Printf("✅ 所有异步调用完成\n")
			fmt.printf("📊 统计结果:\n")
			fmt.Printf("   启动调用: %d, 完成调用: %d, 失败调用: %d\n",
				len(callIDs), completedCalls, errorCalls)
			fmt.Printf("   总耗时: %v, 平均完成时间: %v\n",
				duration, duration/time.Duration(completedCalls+errorCalls))

		case <-time.After(30 * time.Second):
			fmt.Printf("⏰ 异步调用超时，部分调用可能仍在执行\n")
			fmt.Printf("📊 当前状态: 完成 %d, 失败 %d\n", completedCalls, errorCalls)

			// 取消剩余的异步调用
			cancelledCount := 0
			for _, callID := range callIDs {
				if tripleGS.CancelAsyncCall(callID) {
					cancelledCount++
				}
			}
			fmt.Printf("🛑 取消了 %d 个未完成的调用\n", cancelledCount)
		}

		// 检查异步管理器状态
		activeCalls := tripleGS.GetActiveAsyncCalls()
		fmt.Printf("🎛️ 当前活跃异步调用数: %d\n", len(activeCalls))
	}
}

// 内存使用压力测试
func memoryUsageTest(tripleGS *triple.TripleGenericService, ctx context.Context) {
	fmt.Println("准备内存使用压力测试...")

	// 测试大对象处理
	largeObjectSizes := []int{1000, 5000, 10000}

	for _, size := range largeObjectSizes {
		fmt.Printf("\n📦 测试大对象 (大小: %d 项)\n", size)

		// 创建大型数据结构
		largeData := make([]interface{}, size)
		for i := 0; i < size; i++ {
			largeData[i] = map[string]interface{}{
				"id":   i,
				"data": fmt.Sprintf("large_data_item_%d_%s", i, generateRandomString(100)),
				"metadata": map[string]interface{}{
					"timestamp": time.Now().Unix(),
					"type":      "large_object_test",
					"index":     i,
					"checksum":  fmt.Sprintf("checksum_%d", i*i),
				},
			}
		}

		start := time.Now()
		_, err := tripleGS.Invoke(ctx, "processLargeData",
			[]string{"[]map"},
			[]interface{}{largeData})
		duration := time.Since(start)

		if err != nil {
			fmt.Printf("❌ 大对象处理失败: %v\n", err)
		} else {
			fmt.Printf("✅ 大对象处理成功\n")
		}
		fmt.Printf("⏱️ 处理时间: %v\n", duration)

		// 短暂等待垃圾回收
		time.Sleep(100 * time.Millisecond)
	}

	// 测试大量小对象的批量处理
	fmt.Println("\n🔄 测试大量小对象批量处理")
	smallObjectCount := 2000

	var invocations []triple.TripleInvocationRequest
	for i := 0; i < smallObjectCount; i++ {
		invocations = append(invocations, triple.TripleInvocationRequest{
			MethodName: "processSmallData",
			Types:      []string{"map"},
			Args: []interface{}{
				map[string]interface{}{
					"id":   i,
					"data": fmt.Sprintf("small_data_%d", i),
				},
			},
			Attachments: map[string]interface{}{
				"objectId": i,
				"testType": "memory_stress",
			},
		})
	}

	start := time.Now()
	results, err := tripleGS.BatchInvokeWithOptions(ctx, invocations,
		triple.BatchInvokeOptions{
			MaxConcurrency: 20,
			FailFast:       false,
		})
	duration := time.Since(start)

	if err != nil {
		fmt.Printf("❌ 大量小对象处理失败: %v\n", err)
	} else {
		successCount := 0
		for _, result := range results {
			if result.Error == nil {
				successCount++
			}
		}
		fmt.Printf("✅ 大量小对象处理完成: %d/%d 成功\n", successCount, len(results))
		fmt.Printf("⏱️ 总处理时间: %v, 平均: %v\n", duration, duration/time.Duration(len(results)))
	}
}

// 长时间运行稳定性测试
func longRunningStabilityTest(tripleGS *triple.TripleGenericService, ctx context.Context) {
	fmt.Println("开始长时间运行稳定性测试（缩短版本）...")

	duration := 2 * time.Minute // 实际测试可以设置更长时间
	interval := 5 * time.Second

	fmt.Printf("⏰ 测试将运行 %v，每 %v 执行一轮测试\n", duration, interval)

	start := time.Now()
	round := 0
	totalCalls := 0
	totalErrors := 0

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	endTime := start.Add(duration)

	for time.Now().Before(endTime) {
		select {
		case <-ticker.C:
			round++
			fmt.Printf("\n🔄 第 %d 轮测试\n", round)

			// 混合调用测试
			roundCalls := 0
			roundErrors := 0

			// 1. 单个调用测试
			for i := 0; i < 10; i++ {
				_, err := tripleGS.Invoke(ctx, "stabilityTestMethod",
					[]string{"int32", "string"},
					[]interface{}{int32(round*10 + i), fmt.Sprintf("stability_test_%d_%d", round, i)})
				roundCalls++
				if err != nil {
					roundErrors++
				}
			}

			// 2. 批量调用测试
			var batchInvocations []triple.TripleInvocationRequest
			for i := 0; i < 5; i++ {
				batchInvocations = append(batchInvocations, triple.TripleInvocationRequest{
					MethodName: "batchStabilityTest",
					Types:      []string{"map"},
					Args: []interface{}{
						map[string]interface{}{
							"round": round,
							"index": i,
							"time":  time.Now().Format("15:04:05"),
						},
					},
					Attachments: map[string]interface{}{
						"testRound": round,
						"testType":  "stability",
					},
				})
			}

			results, err := tripleGS.BatchInvoke(ctx, batchInvocations)
			if err != nil {
				roundErrors += len(batchInvocations)
			} else {
				for _, result := range results {
					roundCalls++
					if result.Error != nil {
						roundErrors++
					}
				}
			}

			// 3. 异步调用测试
			asyncCallsCount := 3
			var asyncWg sync.WaitGroup

			for i := 0; i < asyncCallsCount; i++ {
				asyncWg.Add(1)
				_, err := tripleGS.InvokeAsync(ctx, "asyncStabilityTest",
					[]string{"int32"},
					[]interface{}{int32(round*100 + i)},
					nil,
					func(result interface{}, err error) {
						defer asyncWg.Done()
						roundCalls++
						if err != nil {
							roundErrors++
						}
					})

				if err != nil {
					roundErrors++
					asyncWg.Done()
				}
			}

			// 等待异步调用完成（最多等待2秒）
			asyncDone := make(chan struct{})
			go func() {
				asyncWg.Wait()
				close(asyncDone)
			}()

			select {
			case <-asyncDone:
			case <-time.After(2 * time.Second):
				fmt.Printf("⚠️ 异步调用超时\n")
			}

			totalCalls += roundCalls
			totalErrors += roundErrors

			successRate := float64(roundCalls-roundErrors) / float64(roundCalls) * 100
			fmt.Printf("📊 第 %d 轮结果: 调用 %d, 错误 %d, 成功率 %.2f%%\n",
				round, roundCalls, roundErrors, successRate)

			// 检查异步管理器状态
			activeCalls := tripleGS.GetActiveAsyncCalls()
			if len(activeCalls) > 0 {
				fmt.Printf("🎛️ 活跃异步调用: %d\n", len(activeCalls))
			}

		case <-ctx.Done():
			fmt.Println("⏹️ 上下文取消，停止稳定性测试")
			return
		}
	}

	totalDuration := time.Since(start)
	overallSuccessRate := float64(totalCalls-totalErrors) / float64(totalCalls) * 100
	avgQPS := float64(totalCalls) / totalDuration.Seconds()

	fmt.Printf("\n🏁 长时间稳定性测试完成\n")
	fmt.Printf("📊 总体统计:\n")
	fmt.Printf("   运行时间: %v, 测试轮数: %d\n", totalDuration, round)
	fmt.Printf("   总调用: %d, 总错误: %d\n", totalCalls, totalErrors)
	fmt.Printf("   整体成功率: %.2f%%, 平均QPS: %.2f\n", overallSuccessRate, avgQPS)
}

// 生成随机字符串的辅助函数
func generateRandomString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[i%len(charset)]
	}
	return string(b)
}

