#!/bin/bash

# Triple泛化调用网关测试脚本

echo "🚀 测试Triple泛化调用网关"
echo "=========================="

# 网关地址
GATEWAY_URL="http://localhost:8080"

echo "1. 健康检查"
curl -X GET "$GATEWAY_URL/health" | jq '.'

echo -e "\n2. 测试用户服务 - 根据ID查询用户"
curl -X POST "$GATEWAY_URL/api/v1/UserService/getUserById" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer test-token-123" \
  -H "X-Request-ID: req-$(date +%s)" \
  -d '{"value": 12345}' | jq '.'

echo -e "\n3. 测试用户服务 - 创建用户"
curl -X POST "$GATEWAY_URL/api/v1/UserService/createUser" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer test-token-123" \
  -d '{
    "id": 12345,
    "name": "张三",
    "age": 28,
    "email": "zhangsan@example.com",
    "profile": {
      "city": "北京",
      "company": "阿里巴巴",
      "department": "技术部"
    },
    "hobbies": ["编程", "阅读", "旅游"],
    "active": true
  }' | jq '.'

echo -e "\n4. 测试用户服务 - 更新用户 (多参数)"
curl -X POST "$GATEWAY_URL/api/v1/UserService/updateUser" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer test-token-123" \
  -d '{
    "param0": 12345,
    "param1": {
      "name": "张三（已更新）",
      "age": 29,
      "email": "zhangsan_updated@example.com"
    }
  }' | jq '.'

echo -e "\n5. 测试订单服务 - 创建订单"
curl -X POST "$GATEWAY_URL/api/v1/OrderService/createOrder" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer test-token-123" \
  -H "User-Agent: Test-Client/1.0" \
  -d '{
    "userId": 12345,
    "products": [
      {
        "productId": 1001,
        "name": "iPhone 15 Pro",
        "quantity": 1,
        "price": 999.99
      },
      {
        "productId": 1002,
        "name": "AirPods Pro",
        "quantity": 1,
        "price": 249.99
      }
    ],
    "totalAmount": 1249.98,
    "shippingAddress": {
      "city": "北京",
      "street": "长安街1号",
      "zipcode": "100000"
    },
    "paymentMethod": "alipay"
  }' | jq '.'

echo -e "\n6. 测试订单服务 - 查询订单"
curl -X POST "$GATEWAY_URL/api/v1/OrderService/getOrder" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer test-token-123" \
  -d '{"value": "ORDER_20231201_001"}' | jq '.'

echo -e "\n7. 测试错误情况 - 不存在的服务"
curl -X POST "$GATEWAY_URL/api/v1/NonExistentService/someMethod" \
  -H "Content-Type: application/json" \
  -d '{}' | jq '.'

echo -e "\n8. 测试错误情况 - 不存在的方法"
curl -X POST "$GATEWAY_URL/api/v1/UserService/nonExistentMethod" \
  -H "Content-Type: application/json" \
  -d '{}' | jq '.'

echo -e "\n9. 测试错误情况 - 无效的路径格式"
curl -X POST "$GATEWAY_URL/api/v1/InvalidPath" \
  -H "Content-Type: application/json" \
  -d '{}' | jq '.'

echo -e "\n✅ 网关测试完成！"
echo "说明："
echo "- 所有请求都会显示网络错误（因为没有真实的后端服务）"
echo "- 但是可以验证网关的路由、参数转换、错误处理等功能"
echo "- 检查日志可以看到详细的处理过程"





