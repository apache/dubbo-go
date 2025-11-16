//go:build example_real_world
// +build example_real_world

/*
 * Triple Generic Call Real-World Scenarios Example
 */

package main

import (
	"context"
	"fmt"
	"time"

	"dubbo.apache.org/dubbo-go/v3/protocol/triple"
)

func main() {
	fmt.Println(" Triple Generic Call Real-World Scenarios Example")
	fmt.Println("=====================================")

	// 初始化不同的服务客户端
	userService := triple.NewTripleGenericService("tri://user-service:20000/com.company.UserService?serialization=hessian2")
	orderService := triple.NewTripleGenericService("tri://order-service:20001/com.company.OrderService?serialization=hessian2")
	paymentService := triple.NewTripleGenericService("tri://payment-service:20002/com.company.PaymentService?serialization=hessian2")
	inventoryService := triple.NewTripleGenericService("tri://inventory-service:20003/com.company.InventoryService?serialization=hessian2")
	notificationService := triple.NewTripleGenericService("tri://notification-service:20004/com.company.NotificationService?serialization=hessian2")

	ctx := context.Background()

	// Scenario 1: E-commerce Order Complete Process
	fmt.Println("\n🛒 Scenario 1: E-commerce Order Complete Process")
	eCommerceOrderFlow(ctx, userService, orderService, paymentService, inventoryService, notificationService)

	// Scenario 2: User Management System
	fmt.Println("\n👥 Scenario 2: User Management System")
	userManagementSystem(ctx, userService, notificationService)

	// Scenario 3: Data Analysis and Reporting
	fmt.Println("\n📊 Scenario 3: Data Analysis and Reporting")
	dataAnalyticsScenario(ctx, orderService, userService)

	// Scenario 4: Microservice Chain Invocation
	fmt.Println("\n🔗 Scenario 4: Microservice Chain Invocation")
	microserviceChainCall(ctx, userService, orderService, paymentService)

	// Scenario 5: Batch Data Processing
	fmt.Println("\n⚡ Scenario 5: Batch Data Processing")
	batchDataProcessing(ctx, inventoryService, orderService)

	fmt.Println("\n🎉 Real-world scenarios example completed!")
}

// E-commerce Order Complete Process
func eCommerceOrderFlow(ctx context.Context, userService, orderService, paymentService, inventoryService, notificationService *triple.TripleGenericService) {
	fmt.Println("Starting e-commerce order process...")

	// User information
	userID := int64(12345)
	productID := int64(67890)
	quantity := int32(2)
	unitPrice := 299.99

	// 1. Verify user information
	fmt.Println("🔍 Step 1: Verify user information")
	userResult, err := userService.InvokeWithAttachments(ctx, "getUserById",
		[]string{"int64"},
		[]interface{}{userID},
		map[string]interface{}{
			"traceId":     "order-flow-001",
			"step":        "user_verification",
			"requestTime": time.Now().Format("2006-01-02 15:04:05"),
		})

	if err != nil {
		fmt.Printf("❌ User verification failed: %v\n", err)
		return
	}
	fmt.Printf("✅ User verification successful: %v\n", userResult)

	// 2. Check inventory
	fmt.Println("📦 Step 2: Check product inventory")
	inventoryResult, err := inventoryService.InvokeWithAttachments(ctx, "checkStock",
		[]string{"int64", "int32"},
		[]interface{}{productID, quantity},
		map[string]interface{}{
			"traceId":   "order-flow-001",
			"step":      "inventory_check",
			"productId": productID,
		})

	if err != nil {
		fmt.Printf("❌ Inventory check failed: %v\n", err)
		return
	}
	fmt.Printf("✅ Inventory check successful: %v\n", inventoryResult)

	// 3. Create order
	fmt.Println("📝 Step 3: Create order")
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
		fmt.Printf("❌ Order creation failed: %v\n", err)
		return
	}
	fmt.Printf("✅ Order creation successful: %v\n", orderResult)

	// 假设从订单结果中提取订单ID
	orderID := "ORDER_20231201_001"

	// 4. Process payment
	fmt.Println("💳 Step 4: Process payment")
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
		fmt.Printf("❌ Payment processing failed: %v\n", err)
		return
	}
	fmt.Printf("✅ Payment processing successful: %v\n", paymentResult)

	// 5. Update inventory
	fmt.Println("📉 Step 5: Update inventory")
	_, err = inventoryService.InvokeWithAttachments(ctx, "reduceStock",
		[]string{"int64", "int32", "string"},
		[]interface{}{productID, quantity, orderID},
		map[string]interface{}{
			"traceId": "order-flow-001",
			"step":    "inventory_update",
			"orderId": orderID,
		})

	if err != nil {
		fmt.Printf("❌ Inventory update failed: %v\n", err)
	} else {
		fmt.Printf("✅ Inventory update successful\n")
	}

	// 6. Send notification
	fmt.Println("📧 Step 6: Send order confirmation notification")
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
		fmt.Printf("❌ Notification sending failed: %v\n", err)
	} else {
		fmt.Printf("✅ Notification sending successful\n")
	}

	fmt.Println("🎉 E-commerce order process completed!")
}

// User Management System
func userManagementSystem(ctx context.Context, userService, notificationService *triple.TripleGenericService) {
	fmt.Println("Starting user management system demonstration...")

	// Batch user operations
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
		fmt.Printf("❌ Batch user operation failed: %v\n", err)
		return
	}

	// 处理结果并发送通知
	for i, result := range results {
		operation := userOperations[i]
		if result.Error != nil {
			fmt.Printf("❌ User operation %d (%s) failed: %v\n", i, operation["action"], result.Error)
		} else {
			fmt.Printf("✅ User operation %d (%s) successful: %v\n", i, operation["action"], result.Result)

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
							fmt.Printf("⚠️ Welcome email sending failed: %v\n", err)
						} else {
							fmt.Printf("📧 Welcome email sent successfully\n")
						}
					})
			}
		}
	}
}

// Data Analysis and Reporting Scenario
func dataAnalyticsScenario(ctx context.Context, orderService, userService *triple.TripleGenericService) {
	fmt.Println("Starting data analysis and reporting generation...")

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
		fmt.Printf("❌ Data analysis failed: %v\n", err)
		return
	}

	fmt.Printf("📈 Data analysis completed, duration: %v\n", duration)

	reportTypes := []string{"销售报表", "用户活动报表", "产品表现报表"}
	for i, result := range results {
		if result.Error != nil {
			fmt.Printf("❌ %s generation failed: %v\n", reportTypes[i], result.Error)
		} else {
			fmt.Printf("✅ %s generated successfully\n", reportTypes[i])
		}
	}

	// Get popular product data
	fmt.Println("📊 Getting popular product data...")
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
		fmt.Printf("❌ Failed to get popular products: %v\n", err)
	} else {
		fmt.Printf("✅ Successfully retrieved popular product data\n")
	}
}

// Microservice Chain Call
func microserviceChainCall(ctx context.Context, userService, orderService, paymentService *triple.TripleGenericService) {
	fmt.Println("Starting microservice chain call demonstration...")

	// 模拟一个复杂的业务链路：用户升级VIP会员
	userID := int64(54321)
	membershipType := "VIP_GOLD"

	// Link 1: User Service -> Verify user eligibility
	fmt.Println("🔗 Link 1: Verify VIP upgrade eligibility")
	userEligibility, err := userService.InvokeWithAttachments(ctx, "checkVipEligibility",
		[]string{"int64", "string"},
		[]interface{}{userID, membershipType},
		map[string]interface{}{
			"chainId":     "vip_upgrade_001",
			"step":        1,
			"serviceName": "user-service",
		})

	if err != nil {
		fmt.Printf("❌ User eligibility verification failed: %v\n", err)
		return
	}
	fmt.Printf("✅ User eligibility verification completed: %v\n", userEligibility)

	// Link 2: Order Service -> Calculate historical consumption
	fmt.Println("🔗 Link 2: Calculate user historical consumption")
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
		fmt.Printf("❌ Historical consumption calculation failed: %v\n", err)
		return
	}
	fmt.Printf("✅ Historical consumption calculation completed: %v\n", consumptionHistory)

	// Link 3: Payment Service -> Process VIP fees
	fmt.Println("🔗 Link 3: Process VIP membership fees")
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
		fmt.Printf("❌ VIP fee processing failed: %v\n", err)
		return
	}
	fmt.Printf("✅ VIP fee processing completed: %v\n", paymentResult)

	// Link 4: User Service -> Activate VIP membership
	fmt.Println("🔗 Link 4: Activate VIP membership qualification")
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

	fmt.Println("🎉 Microservice chain call completed!")
}

// Batch Data Processing
func batchDataProcessing(ctx context.Context, inventoryService, orderService *triple.TripleGenericService) {
	fmt.Println("Starting batch data processing demonstration...")

	// Simulate inventory audit task
	fmt.Println("📦 Executing batch inventory audit")

	// Generate a large number of inventory audit requests
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
			fmt.Printf("❌ Batch processing failed: %v\n", err)
			continue
		}

		successCount := 0
		for _, result := range results {
			if result.Error == nil {
				successCount++
			}
		}

		fmt.Printf("✅ Batch completed: %d/%d successful, duration: %v\n",
			successCount, len(results), duration)

		// 短暂延迟避免服务过载
		time.Sleep(100 * time.Millisecond)
	}

	// Generate inventory report
	fmt.Println("📊 Generating inventory audit report")
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
		fmt.Printf("❌ Inventory report generation failed: %v\n", err)
	} else {
		fmt.Printf("✅ Inventory report generated successfully\n")
	}

	fmt.Println("🎉 Batch data processing completed!")
}
