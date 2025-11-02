/*
 * Triple 泛化调用真实世界场景示例
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
	fmt.Println("🌍 Triple 泛化调用真实世界场景示例")
	fmt.Println("=====================================")

	// 初始化不同的服务客户端
	userService := triple.NewTripleGenericService("tri://user-service:20000/com.company.UserService?serialization=hessian2")
	orderService := triple.NewTripleGenericService("tri://order-service:20001/com.company.OrderService?serialization=hessian2")
	paymentService := triple.NewTripleGenericService("tri://payment-service:20002/com.company.PaymentService?serialization=hessian2")
	inventoryService := triple.NewTripleGenericService("tri://inventory-service:20003/com.company.InventoryService?serialization=hessian2")
	notificationService := triple.NewTripleGenericService("tri://notification-service:20004/com.company.NotificationService?serialization=hessian2")

	ctx := context.Background()

	// 场景1: 电商下单完整流程
	fmt.Println("\n🛒 场景1: 电商下单完整流程")
	eCommerceOrderFlow(ctx, userService, orderService, paymentService, inventoryService, notificationService)

	// 场景2: 用户管理系统
	fmt.Println("\n👥 场景2: 用户管理系统")
	userManagementSystem(ctx, userService, notificationService)

	// 场景3: 数据分析和报表
	fmt.Println("\n📊 场景3: 数据分析和报表生成")
	dataAnalyticsScenario(ctx, orderService, userService)

	// 场景4: 微服务链路调用
	fmt.Println("\n🔗 场景4: 微服务链路调用")
	microserviceChainCall(ctx, userService, orderService, paymentService)

	// 场景5: 批量数据处理
	fmt.Println("\n⚡ 场景5: 批量数据处理")
	batchDataProcessing(ctx, inventoryService, orderService)

	fmt.Println("\n🎉 真实世界场景示例完成!")
}

// 电商下单完整流程
func eCommerceOrderFlow(ctx context.Context, userService, orderService, paymentService, inventoryService, notificationService *triple.TripleGenericService) {
	fmt.Println("开始电商下单流程...")

	// 用户信息
	userID := int64(12345)
	productID := int64(67890)
	quantity := int32(2)
	unitPrice := 299.99

	// 1. 验证用户信息
	fmt.Println("🔍 步骤1: 验证用户信息")
	userResult, err := userService.InvokeWithAttachments(ctx, "getUserById",
		[]string{"int64"},
		[]interface{}{userID},
		map[string]interface{}{
			"traceId":     "order-flow-001",
			"step":        "user_verification",
			"requestTime": time.Now().Format("2006-01-02 15:04:05"),
		})

	if err != nil {
		fmt.Printf("❌ 用户验证失败: %v\n", err)
		return
	}
	fmt.Printf("✅ 用户验证成功: %v\n", userResult)

	// 2. 检查库存
	fmt.Println("📦 步骤2: 检查商品库存")
	inventoryResult, err := inventoryService.InvokeWithAttachments(ctx, "checkStock",
		[]string{"int64", "int32"},
		[]interface{}{productID, quantity},
		map[string]interface{}{
			"traceId":   "order-flow-001",
			"step":      "inventory_check",
			"productId": productID,
		})

	if err != nil {
		fmt.Printf("❌ 库存检查失败: %v\n", err)
		return
	}
	fmt.Printf("✅ 库存检查成功: %v\n", inventoryResult)

	// 3. 创建订单
	fmt.Println("📝 步骤3: 创建订单")
	orderData := map[string]interface{}{
		"userId":     userID,
		"productId":  productID,
		"quantity":   quantity,
		"unitPrice":  unitPrice,
		"totalPrice": float64(quantity) * unitPrice,
		"orderTime":  time.Now().Format("2006-01-02 15:04:05"),
		"status":     "pending",
	}

	orderResult, err := orderService.InvokeWithAttachments(ctx, "createOrder",
		[]string{"map"},
		[]interface{}{orderData},
		map[string]interface{}{
			"traceId":       "order-flow-001",
			"step":          "order_creation",
			"userId":        userID,
			"estimatedTime": "2-3 business days",
		})

	if err != nil {
		fmt.Printf("❌ 订单创建失败: %v\n", err)
		return
	}
	fmt.Printf("✅ 订单创建成功: %v\n", orderResult)

	// 假设从订单结果中提取订单ID
	orderID := "ORDER_20231201_001"

	// 4. 处理支付
	fmt.Println("💳 步骤4: 处理支付")
	paymentData := map[string]interface{}{
		"orderId":       orderID,
		"amount":        float64(quantity) * unitPrice,
		"paymentMethod": "alipay",
		"currency":      "CNY",
	}

	paymentResult, err := paymentService.InvokeWithAttachments(ctx, "processPayment",
		[]string{"map"},
		[]interface{}{paymentData},
		map[string]interface{}{
			"traceId":       "order-flow-001",
			"step":          "payment_processing",
			"orderId":       orderID,
			"securityLevel": "high",
		})

	if err != nil {
		fmt.Printf("❌ 支付处理失败: %v\n", err)
		return
	}
	fmt.Printf("✅ 支付处理成功: %v\n", paymentResult)

	// 5. 更新库存
	fmt.Println("📉 步骤5: 更新库存")
	_, err = inventoryService.InvokeWithAttachments(ctx, "reduceStock",
		[]string{"int64", "int32", "string"},
		[]interface{}{productID, quantity, orderID},
		map[string]interface{}{
			"traceId": "order-flow-001",
			"step":    "inventory_update",
			"orderId": orderID,
		})

	if err != nil {
		fmt.Printf("❌ 库存更新失败: %v\n", err)
	} else {
		fmt.Printf("✅ 库存更新成功\n")
	}

	// 6. 发送通知
	fmt.Println("📧 步骤6: 发送订单确认通知")
	notificationData := map[string]interface{}{
		"userId":   userID,
		"orderId":  orderID,
		"type":     "order_confirmation",
		"content":  fmt.Sprintf("您的订单 %s 已确认，预计2-3个工作日内发货", orderID),
		"channels": []string{"email", "sms", "push"},
	}

	_, err = notificationService.InvokeWithAttachments(ctx, "sendNotification",
		[]string{"map"},
		[]interface{}{notificationData},
		map[string]interface{}{
			"traceId":  "order-flow-001",
			"step":     "notification",
			"priority": "high",
		})

	if err != nil {
		fmt.Printf("❌ 通知发送失败: %v\n", err)
	} else {
		fmt.Printf("✅ 通知发送成功\n")
	}

	fmt.Println("🎉 电商下单流程完成!")
}

// 用户管理系统
func userManagementSystem(ctx context.Context, userService, notificationService *triple.TripleGenericService) {
	fmt.Println("开始用户管理系统演示...")

	// 批量用户操作
	userOperations := []map[string]interface{}{
		{
			"action": "create",
			"data": map[string]interface{}{
				"username": "alice_chen",
				"email":    "alice@example.com",
				"profile": map[string]interface{}{
					"firstName": "Alice",
					"lastName":  "Chen",
					"age":       28,
					"city":      "Shanghai",
				},
			},
		},
		{
			"action": "create",
			"data": map[string]interface{}{
				"username": "bob_wang",
				"email":    "bob@example.com",
				"profile": map[string]interface{}{
					"firstName": "Bob",
					"lastName":  "Wang",
					"age":       32,
					"city":      "Beijing",
				},
			},
		},
		{
			"action": "update",
			"userId": int64(1001),
			"data": map[string]interface{}{
				"profile": map[string]interface{}{
					"age":  29,
					"city": "Guangzhou",
				},
				"lastLoginTime": time.Now().Format("2006-01-02 15:04:05"),
			},
		},
	}

	// 批量处理用户操作
	var invocations []triple.TripleInvocationRequest
	for i, operation := range userOperations {
		var methodName string
		var args []interface{}

		switch operation["action"] {
		case "create":
			methodName = "createUser"
			args = []interface{}{operation["data"]}
		case "update":
			methodName = "updateUser"
			args = []interface{}{operation["userId"], operation["data"]}
		}

		invocations = append(invocations, triple.TripleInvocationRequest{
			MethodName: methodName,
			Types:      []string{"map"},
			Args:       args,
			Attachments: map[string]interface{}{
				"operationId":   fmt.Sprintf("user_op_%d", i),
				"operationType": operation["action"],
				"batchId":       "user_batch_001",
				"operatorId":    "admin_001",
			},
		})
	}

	// 执行批量用户操作
	results, err := userService.BatchInvoke(ctx, invocations)
	if err != nil {
		fmt.Printf("❌ 批量用户操作失败: %v\n", err)
		return
	}

	// 处理结果并发送通知
	for i, result := range results {
		operation := userOperations[i]
		if result.Error != nil {
			fmt.Printf("❌ 用户操作 %d (%s) 失败: %v\n", i, operation["action"], result.Error)
		} else {
			fmt.Printf("✅ 用户操作 %d (%s) 成功: %v\n", i, operation["action"], result.Result)

			// 发送操作完成通知
			if operation["action"] == "create" {
				notificationData := map[string]interface{}{
					"type":    "welcome",
					"email":   operation["data"].(map[string]interface{})["email"],
					"content": "欢迎加入我们的平台！",
				}

				notificationService.InvokeAsync(ctx, "sendWelcomeEmail",
					[]string{"map"},
					[]interface{}{notificationData},
					nil,
					func(result interface{}, err error) {
						if err != nil {
							fmt.Printf("⚠️ 欢迎邮件发送失败: %v\n", err)
						} else {
							fmt.Printf("📧 欢迎邮件发送成功\n")
						}
					})
			}
		}
	}
}

// 数据分析和报表场景
func dataAnalyticsScenario(ctx context.Context, orderService, userService *triple.TripleGenericService) {
	fmt.Println("开始数据分析和报表生成...")

	// 分析参数
	analysisParams := map[string]interface{}{
		"startDate": "2023-11-01",
		"endDate":   "2023-11-30",
		"metrics":   []string{"revenue", "orders", "users", "conversion"},
		"groupBy":   []string{"date", "region"},
	}

	// 并行分析多个维度
	analysisInvocations := []triple.TripleInvocationRequest{
		{
			MethodName: "generateSalesReport",
			Types:      []string{"map"},
			Args:       []interface{}{analysisParams},
			Attachments: map[string]interface{}{
				"reportType": "sales",
				"format":     "json",
			},
		},
		{
			MethodName: "generateUserActivityReport",
			Types:      []string{"map"},
			Args:       []interface{}{analysisParams},
			Attachments: map[string]interface{}{
				"reportType": "user_activity",
				"format":     "json",
			},
		},
		{
			MethodName: "generateProductPerformanceReport",
			Types:      []string{"map"},
			Args:       []interface{}{analysisParams},
			Attachments: map[string]interface{}{
				"reportType": "product_performance",
				"format":     "json",
			},
		},
	}

	// 使用高并发批量分析
	options := triple.BatchInvokeOptions{
		MaxConcurrency: 3,
		FailFast:       false,
	}

	start := time.Now()
	results, err := orderService.BatchInvokeWithOptions(ctx, analysisInvocations, options)
	duration := time.Since(start)

	if err != nil {
		fmt.Printf("❌ 数据分析失败: %v\n", err)
		return
	}

	fmt.Printf("📈 数据分析完成，耗时: %v\n", duration)

	reportTypes := []string{"销售报表", "用户活动报表", "产品表现报表"}
	for i, result := range results {
		if result.Error != nil {
			fmt.Printf("❌ %s 生成失败: %v\n", reportTypes[i], result.Error)
		} else {
			fmt.Printf("✅ %s 生成成功\n", reportTypes[i])
		}
	}

	// 获取热门产品数据
	fmt.Println("📊 获取热门产品数据...")
	topProductsParams := map[string]interface{}{
		"limit":    10,
		"sortBy":   "sales_volume",
		"period":   "last_30_days",
		"category": "electronics",
	}

	_, err = orderService.InvokeWithAttachments(ctx, "getTopProducts",
		[]string{"map"},
		[]interface{}{topProductsParams},
		map[string]interface{}{
			"cacheExpiry": 3600,
			"priority":    "normal",
		})

	if err != nil {
		fmt.Printf("❌ 获取热门产品失败: %v\n", err)
	} else {
		fmt.Printf("✅ 热门产品数据获取成功\n")
	}
}

// 微服务链路调用
func microserviceChainCall(ctx context.Context, userService, orderService, paymentService *triple.TripleGenericService) {
	fmt.Println("开始微服务链路调用演示...")

	// 模拟一个复杂的业务链路：用户升级VIP会员
	userID := int64(54321)
	membershipType := "VIP_GOLD"

	// 链路1: 用户服务 -> 验证用户资格
	fmt.Println("🔗 链路1: 验证用户VIP升级资格")
	userEligibility, err := userService.InvokeWithAttachments(ctx, "checkVipEligibility",
		[]string{"int64", "string"},
		[]interface{}{userID, membershipType},
		map[string]interface{}{
			"chainId":     "vip_upgrade_001",
			"step":        1,
			"serviceName": "user-service",
		})

	if err != nil {
		fmt.Printf("❌ 用户资格验证失败: %v\n", err)
		return
	}
	fmt.Printf("✅ 用户资格验证完成: %v\n", userEligibility)

	// 链路2: 订单服务 -> 计算历史消费
	fmt.Println("🔗 链路2: 计算用户历史消费")
	consumptionHistory, err := orderService.InvokeWithAttachments(ctx, "calculateUserConsumption",
		[]string{"int64", "string"},
		[]interface{}{userID, "last_12_months"},
		map[string]interface{}{
			"chainId":     "vip_upgrade_001",
			"step":        2,
			"serviceName": "order-service",
			"fromStep":    1,
		})

	if err != nil {
		fmt.Printf("❌ 历史消费计算失败: %v\n", err)
		return
	}
	fmt.Printf("✅ 历史消费计算完成: %v\n", consumptionHistory)

	// 链路3: 支付服务 -> 处理VIP费用
	fmt.Println("🔗 链路3: 处理VIP会员费用")
	vipPaymentData := map[string]interface{}{
		"userId":         userID,
		"membershipType": membershipType,
		"amount":         999.00,
		"validityPeriod": "12_months",
		"benefits": []string{
			"free_shipping",
			"priority_support",
			"exclusive_discounts",
		},
	}

	paymentResult, err := paymentService.InvokeWithAttachments(ctx, "processVipPayment",
		[]string{"map"},
		[]interface{}{vipPaymentData},
		map[string]interface{}{
			"chainId":     "vip_upgrade_001",
			"step":        3,
			"serviceName": "payment-service",
			"fromStep":    2,
		})

	if err != nil {
		fmt.Printf("❌ VIP费用处理失败: %v\n", err)
		return
	}
	fmt.Printf("✅ VIP费用处理完成: %v\n", paymentResult)

	// 链路4: 用户服务 -> 激活VIP会员
	fmt.Println("🔗 链路4: 激活VIP会员资格")
	vipActivationData := map[string]interface{}{
		"userId":           userID,
		"membershipType":   membershipType,
		"activationDate":   time.Now().Format("2006-01-02"),
		"expirationDate":   time.Now().AddDate(1, 0, 0).Format("2006-01-02"),
		"paymentReference": paymentResult,
	}

	_, err = userService.InvokeWithAttachments(ctx, "activateVipMembership",
		[]string{"map"},
		[]interface{}{vipActivationData},
		map[string]interface{}{
			"chainId":     "vip_upgrade_001",
			"step":        4,
			"serviceName": "user-service",
			"fromStep":    3,
			"chainEnd":    true,
		})

	if err != nil {
		fmt.Printf("❌ VIP会员激活失败: %v\n", err)
		return
	}
	fmt.Printf("✅ VIP会员激活成功\n")

	fmt.Println("🎉 微服务链路调用完成!")
}

// 批量数据处理
func batchDataProcessing(ctx context.Context, inventoryService, orderService *triple.TripleGenericService) {
	fmt.Println("开始批量数据处理演示...")

	// 模拟库存盘点任务
	fmt.Println("📦 执行批量库存盘点")

	// 生成大量库存盘点请求
	var inventoryChecks []triple.TripleInvocationRequest
	warehouseIDs := []string{"WH_001", "WH_002", "WH_003", "WH_004", "WH_005"}

	for _, warehouseID := range warehouseIDs {
		for categoryID := 1; categoryID <= 10; categoryID++ {
			inventoryChecks = append(inventoryChecks, triple.TripleInvocationRequest{
				MethodName: "performInventoryCheck",
				Types:      []string{"string", "int32", "map"},
				Args: []interface{}{
					warehouseID,
					int32(categoryID),
					map[string]interface{}{
						"checkType":      "full_audit",
						"tolerance":      0.02,
						"includeExpired": false,
					},
				},
				Attachments: map[string]interface{}{
					"warehouseId": warehouseID,
					"categoryId":  categoryID,
					"batchId":     "inventory_audit_20231201",
					"auditType":   "scheduled",
				},
			})
		}
	}

	fmt.Printf("📋 准备检查 %d 个库存单位\n", len(inventoryChecks))

	// 分批处理，避免过载
	batchSize := 15
	for i := 0; i < len(inventoryChecks); i += batchSize {
		end := i + batchSize
		if end > len(inventoryChecks) {
			end = len(inventoryChecks)
		}

		batchInvocations := inventoryChecks[i:end]
		fmt.Printf("🔄 处理批次 %d-%d (%d个检查项)\n", i+1, end, len(batchInvocations))

		start := time.Now()
		results, err := inventoryService.BatchInvokeWithOptions(ctx, batchInvocations,
			triple.BatchInvokeOptions{
				MaxConcurrency: 5,
				FailFast:       false,
			})
		duration := time.Since(start)

		if err != nil {
			fmt.Printf("❌ 批次处理失败: %v\n", err)
			continue
		}

		successCount := 0
		for _, result := range results {
			if result.Error == nil {
				successCount++
			}
		}

		fmt.Printf("✅ 批次完成: %d/%d 成功, 耗时: %v\n",
			successCount, len(results), duration)

		// 短暂延迟避免服务过载
		time.Sleep(100 * time.Millisecond)
	}

	// 生成库存报告
	fmt.Println("📊 生成库存盘点报告")
	reportParams := map[string]interface{}{
		"auditId":        "inventory_audit_20231201",
		"warehouses":     warehouseIDs,
		"reportType":     "summary",
		"format":         "excel",
		"includeDetails": true,
	}

	_, err := inventoryService.InvokeWithAttachments(ctx, "generateInventoryReport",
		[]string{"map"},
		[]interface{}{reportParams},
		map[string]interface{}{
			"priority":     "high",
			"deliveryMode": "email",
			"recipients":   []string{"inventory@company.com", "manager@company.com"},
		})

	if err != nil {
		fmt.Printf("❌ 库存报告生成失败: %v\n", err)
	} else {
		fmt.Printf("✅ 库存报告生成成功\n")
	}

	fmt.Println("🎉 批量数据处理完成!")
}

