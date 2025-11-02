/*
 * Triple泛化调用网关示例
 * 演示如何在网关中处理HTTP/JSON到Triple/Protobuf的转换
 */

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
)

import (
	"dubbo.apache.org/dubbo-go/v3/protocol/triple"
)

// 网关结构
type TripleGateway struct {
	services map[string]*triple.TripleGenericService
	schemas  map[string]*ServiceSchema
}

// 服务Schema定义
type ServiceSchema struct {
	ServiceName string
	Methods     map[string]*MethodSchema
}

type MethodSchema struct {
	MethodName     string
	RequestType    string
	ResponseType   string
	ParameterTypes []string
}

func main() {
	fmt.Println("🚀 Triple泛化调用网关演示")
	fmt.Println("=============================")

	// 创建网关实例
	gateway := NewTripleGateway()

	// 注册后端服务
	gateway.RegisterService("UserService", "tri://user-service:20000/com.example.UserService")
	gateway.RegisterService("OrderService", "tri://order-service:20001/com.example.OrderService")

	// 初始化服务Schema (模拟配置)
	gateway.InitializeSchemas()

	// 设置HTTP路由
	http.HandleFunc("/api/v1/", gateway.HandleHTTPRequest)
	http.HandleFunc("/health", gateway.HealthCheck)

	fmt.Println("网关已启动在端口 :8080")
	fmt.Println("示例请求:")
	fmt.Println("  POST /api/v1/UserService/createUser")
	fmt.Println("  POST /api/v1/UserService/getUserById")
	fmt.Println("  POST /api/v1/OrderService/createOrder")

	log.Fatal(http.ListenAndServe(":8080", nil))
}

// 创建网关实例
func NewTripleGateway() *TripleGateway {
	return &TripleGateway{
		services: make(map[string]*triple.TripleGenericService),
		schemas:  make(map[string]*ServiceSchema),
	}
}

// 注册服务
func (gw *TripleGateway) RegisterService(serviceName, serviceURL string) {
	tripleService := triple.NewTripleGenericService(serviceURL)
	gw.services[serviceName] = tripleService
	log.Printf("注册服务: %s -> %s", serviceName, serviceURL)
}

// 初始化服务Schema (模拟配置)
func (gw *TripleGateway) InitializeSchemas() {
	// UserService Schema
	userServiceSchema := &ServiceSchema{
		ServiceName: "UserService",
		Methods: map[string]*MethodSchema{
			"createUser": {
				MethodName:     "createUser",
				RequestType:    "com.example.CreateUserRequest",
				ResponseType:   "com.example.CreateUserResponse",
				ParameterTypes: []string{"com.example.User"},
			},
			"getUserById": {
				MethodName:     "getUserById",
				RequestType:    "com.example.GetUserRequest",
				ResponseType:   "com.example.User",
				ParameterTypes: []string{"int64"},
			},
			"updateUser": {
				MethodName:     "updateUser",
				RequestType:    "com.example.UpdateUserRequest",
				ResponseType:   "com.example.UpdateUserResponse",
				ParameterTypes: []string{"int64", "com.example.User"},
			},
		},
	}

	// OrderService Schema
	orderServiceSchema := &ServiceSchema{
		ServiceName: "OrderService",
		Methods: map[string]*MethodSchema{
			"createOrder": {
				MethodName:     "createOrder",
				RequestType:    "com.example.CreateOrderRequest",
				ResponseType:   "com.example.CreateOrderResponse",
				ParameterTypes: []string{"com.example.Order"},
			},
			"getOrder": {
				MethodName:     "getOrder",
				RequestType:    "com.example.GetOrderRequest",
				ResponseType:   "com.example.Order",
				ParameterTypes: []string{"string"},
			},
		},
	}

	gw.schemas["UserService"] = userServiceSchema
	gw.schemas["OrderService"] = orderServiceSchema

	log.Printf("初始化Schema: UserService (%d methods), OrderService (%d methods)",
		len(userServiceSchema.Methods), len(orderServiceSchema.Methods))
}

// HTTP请求处理
func (gw *TripleGateway) HandleHTTPRequest(w http.ResponseWriter, r *http.Request) {
	// 设置CORS headers
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	log.Printf("收到请求: %s %s", r.Method, r.URL.Path)

	ctx := context.Background()

	// 1. 解析请求路径
	serviceName, methodName, err := gw.parseRequestPath(r.URL.Path)
	if err != nil {
		gw.writeErrorResponse(w, http.StatusBadRequest, "Invalid request path: "+err.Error())
		return
	}

	log.Printf("解析路径: service=%s, method=%s", serviceName, methodName)

	// 2. 验证服务和方法是否存在
	schema, exists := gw.schemas[serviceName]
	if !exists {
		gw.writeErrorResponse(w, http.StatusNotFound, "Service not found: "+serviceName)
		return
	}

	methodSchema, exists := schema.Methods[methodName]
	if !exists {
		gw.writeErrorResponse(w, http.StatusNotFound, "Method not found: "+methodName)
		return
	}

	// 3. 解析请求体
	var requestData map[string]any
	if r.Body != nil {
		if err := json.NewDecoder(r.Body).Decode(&requestData); err != nil {
			// 如果JSON解析失败，尝试作为空请求处理
			requestData = make(map[string]any)
		}
	} else {
		requestData = make(map[string]any)
	}

	log.Printf("请求数据: %+v", requestData)

	// 4. 转换参数
	tripleArgs, err := gw.convertJSONToTripleArgs(requestData, methodSchema)
	if err != nil {
		gw.writeErrorResponse(w, http.StatusBadRequest, "Parameter conversion error: "+err.Error())
		return
	}

	log.Printf("转换参数: types=%v, args=%+v", methodSchema.ParameterTypes, tripleArgs)

	// 5. 构建附件
	attachments := gw.buildAttachments(r)

	// 6. 执行Triple泛化调用
	tripleService := gw.services[serviceName]
	result, err := tripleService.InvokeWithAttachments(ctx, methodName,
		methodSchema.ParameterTypes, tripleArgs, attachments)

	if err != nil {
		log.Printf("服务调用失败: %v", err)
		gw.writeErrorResponse(w, http.StatusInternalServerError, "Service call error: "+err.Error())
		return
	}

	log.Printf("服务调用成功: %+v", result)

	// 7. 转换响应
	jsonResult := gw.convertTripleResultToJSON(result, methodSchema.ResponseType)

	// 8. 返回响应
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]any{
		"success": true,
		"data":    jsonResult,
	})
}

// 解析请求路径
func (gw *TripleGateway) parseRequestPath(path string) (serviceName, methodName string, err error) {
	// 路径格式: /api/v1/{serviceName}/{methodName}
	// 例如: /api/v1/UserService/getUserById

	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) < 4 {
		return "", "", fmt.Errorf("invalid path format, expected /api/v1/{service}/{method}")
	}

	if parts[0] != "api" || parts[1] != "v1" {
		return "", "", fmt.Errorf("invalid API version, expected /api/v1/...")
	}

	serviceName = parts[2]
	methodName = parts[3]

	return serviceName, methodName, nil
}

// JSON到Triple参数转换
func (gw *TripleGateway) convertJSONToTripleArgs(jsonData map[string]any, methodSchema *MethodSchema) ([]any, error) {
	paramTypes := methodSchema.ParameterTypes

	if len(paramTypes) == 0 {
		// 无参数方法
		return []any{}, nil
	}

	if len(paramTypes) == 1 {
		// 单参数处理
		paramType := paramTypes[0]

		// 如果是基础类型，从JSON中提取值
		if gw.isBasicType(paramType) {
			if value, exists := jsonData["value"]; exists {
				convertedValue, err := gw.convertValueByType(value, paramType)
				if err != nil {
					return nil, fmt.Errorf("failed to convert value: %w", err)
				}
				return []any{convertedValue}, nil
			} else {
				return nil, fmt.Errorf("missing 'value' field for basic type %s", paramType)
			}
		} else {
			// 复杂类型：直接使用整个JSON作为参数
			return []any{jsonData}, nil
		}
	}

	// 多参数处理
	args := make([]any, len(paramTypes))
	for i, paramType := range paramTypes {
		paramKey := fmt.Sprintf("param%d", i)
		if value, exists := jsonData[paramKey]; exists {
			convertedValue, err := gw.convertValueByType(value, paramType)
			if err != nil {
				return nil, fmt.Errorf("failed to convert parameter %d: %w", i, err)
			}
			args[i] = convertedValue
		} else {
			return nil, fmt.Errorf("missing parameter %s", paramKey)
		}
	}

	return args, nil
}

// 判断是否为基础类型
func (gw *TripleGateway) isBasicType(typeName string) bool {
	basicTypes := []string{"string", "int32", "int64", "float32", "float64", "bool", "bytes"}
	for _, basicType := range basicTypes {
		if typeName == basicType {
			return true
		}
	}
	return false
}

// 类型转换
func (gw *TripleGateway) convertValueByType(value any, targetType string) (any, error) {
	switch targetType {
	case "string":
		if str, ok := value.(string); ok {
			return str, nil
		}
		return fmt.Sprintf("%v", value), nil

	case "int32":
		switch v := value.(type) {
		case float64:
			return int32(v), nil
		case int:
			return int32(v), nil
		case string:
			if i, err := strconv.ParseInt(v, 10, 32); err == nil {
				return int32(i), nil
			}
		}
		return nil, fmt.Errorf("cannot convert %T to int32", value)

	case "int64":
		switch v := value.(type) {
		case float64:
			return int64(v), nil
		case int:
			return int64(v), nil
		case string:
			if i, err := strconv.ParseInt(v, 10, 64); err == nil {
				return i, nil
			}
		}
		return nil, fmt.Errorf("cannot convert %T to int64", value)

	case "float64":
		switch v := value.(type) {
		case float64:
			return v, nil
		case int:
			return float64(v), nil
		case string:
			if f, err := strconv.ParseFloat(v, 64); err == nil {
				return f, nil
			}
		}
		return nil, fmt.Errorf("cannot convert %T to float64", value)

	case "bool":
		if b, ok := value.(bool); ok {
			return b, nil
		}
		return nil, fmt.Errorf("cannot convert %T to bool", value)

	default:
		// 复杂类型直接返回
		return value, nil
	}
}

// 构建请求附件
func (gw *TripleGateway) buildAttachments(r *http.Request) map[string]any {
	attachments := make(map[string]any)

	// 传递HTTP headers
	if auth := r.Header.Get("Authorization"); auth != "" {
		attachments["authorization"] = auth
	}

	if userAgent := r.Header.Get("User-Agent"); userAgent != "" {
		attachments["user-agent"] = userAgent
	}

	if requestID := r.Header.Get("X-Request-ID"); requestID != "" {
		attachments["request-id"] = requestID
	}

	// 添加网关信息
	attachments["gateway-version"] = "v1.0.0"
	attachments["gateway-type"] = "triple-gateway"

	// 添加客户端信息
	if clientIP := gw.getClientIP(r); clientIP != "" {
		attachments["client-ip"] = clientIP
	}

	return attachments
}

// 获取客户端IP
func (gw *TripleGateway) getClientIP(r *http.Request) string {
	// 检查X-Forwarded-For header
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		return strings.Split(xff, ",")[0]
	}

	// 检查X-Real-IP header
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return xri
	}

	// 使用远程地址
	parts := strings.Split(r.RemoteAddr, ":")
	if len(parts) > 0 {
		return parts[0]
	}

	return "unknown"
}

// Triple结果到JSON转换
func (gw *TripleGateway) convertTripleResultToJSON(result any, responseType string) any {
	// 简单实现：直接返回结果
	// 在实际应用中，这里可以添加更复杂的Protobuf到JSON转换逻辑

	if result == nil {
		return map[string]any{"message": "success"}
	}

	// 如果结果已经是map类型，直接返回
	if resultMap, ok := result.(map[string]any); ok {
		return resultMap
	}

	// 否则包装返回
	return map[string]any{
		"result": result,
		"type":   responseType,
	}
}

// 写入错误响应
func (gw *TripleGateway) writeErrorResponse(w http.ResponseWriter, statusCode int, message string) {
	log.Printf("错误响应: %d - %s", statusCode, message)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(map[string]any{
		"success": false,
		"error":   message,
		"code":    statusCode,
	})
}

// 健康检查
func (gw *TripleGateway) HealthCheck(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]any{
		"status":   "healthy",
		"services": len(gw.services),
		"schemas":  len(gw.schemas),
	})
}
