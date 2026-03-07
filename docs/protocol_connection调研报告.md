# Dubbo-go 协议与连接类型调研报告

## 1. 支持的协议列表

在 `common/constant/default.go` 中定义了支持的协议常量：

| 协议名称 | 常量值 | 描述 |
|---------|--------|------|
| Dubbo | `dubbo` | 经典 Dubbo 协议 (基于 TCP/Telnet) |
| Triple | `tri` | Dubbo 3.0 的新协议 (基于 HTTP/2/gRPC) |
| gRPC | `grpc` | 原生 gRPC 协议 |
| JSONRPC | `jsonrpc` | JSON-RPC 协议 (基于 HTTP/1.1) |
| REST | `rest` | RESTful 协议 (基于 HTTP) |

---

## 2. 长连接 vs 短连接分类

### 支持长连接的协议

| 协议 | 连接类型 | 实现方式 |
|------|---------|---------|
| **Dubbo** | 长连接 | 基于 getty (TCP 连接)，使用 `ExchangeClient`，支持心跳机制 |
| **Triple** | 长连接 | 基于 HTTP/2 或 HTTP/3，使用 gRPC 流，支持 KeepAlive |
| **gRPC** | 长连接 | 基于原生 gRPC `grpc.ClientConn`，支持 HTTP/2 长连接 |

### 只能短连接的协议

| 协议 | 连接类型 | 实现方式 |
|------|---------|---------|
| **JSONRPC** | 短连接 | 每次请求创建新的 TCP 连接，请求结束后立即关闭 |
| **REST** | 短连接 | 基于 HTTP/RESTy 客户端，默认使用短连接 |

---

## 3. 长连接和短连接的实现细节

### 3.1 Dubbo 协议 (长连接)

**文件位置：** `protocol/dubbo/`

**连接实现：**
- 使用 `remoting.ExchangeClient` 管理连接
- 基于 getty 框架 (Apache Dubbo Getty) 实现 TCP 长连接
- 连接通过 `exchangeClientMap` (sync.Map) 缓存，相同地址复用连接

**心跳机制：**
- 位置：`remoting/getty/config.go`
- `HeartbeatPeriod`: 默认 60s
- `HeartbeatTimeout`: 默认 5s

**关键代码：** `protocol/dubbo/dubbo_protocol.go:179-222`

```go
func getExchangeClient(url *common.URL) *remoting.ExchangeClient {
    clientTmp, ok := exchangeClientMap.Load(url.Location)
    if !ok {
        // 创建新连接
        exchangeClientTmp = remoting.NewExchangeClient(url, getty.NewClient(...))
        exchangeClientMap.Store(url.Location, exchangeClientTmp)
    }
    // 复用已有连接
    exchangeClient := clientTmp.(*remoting.ExchangeClient)
    exchangeClient.IncreaseActiveNumber()
    return exchangeClient
}
```

### 3.2 Triple 协议 (长连接)

**文件位置：** `protocol/triple/`

**连接实现：**
- 基于 HTTP/2 (默认) 或 HTTP/3
- 使用 `tri.Client` 和 `clientManager` 管理连接
- 支持 KeepAlive 心跳机制

**支持三种 HTTP 模式：**
- HTTP/1 (已废弃)
- HTTP/2 (默认)
- HTTP/3 (需要 TLS)

**关键代码：** `protocol/triple/client.go:199-219`

```go
case constant.CallHTTP2:
    transport = &http2.Transport{
        ReadIdleTimeout: keepAliveInterval,  // 默认 10s
        PingTimeout:     keepAliveTimeout,   // 默认 20s
        DialTLSContext: func(ctx context.Context, network, addr string, tlsConfig *tls.Config) (net.Conn, error) {
            return (&tls.Dialer{Config: tlsConfig}).DialContext(ctx, network, addr)
        },
    }
```

### 3.3 gRPC 协议 (长连接)

**文件位置：** `protocol/grpc/`

**连接实现：**
- 使用原生 gRPC `grpc.Dial()` 创建长连接
- 基于 HTTP/2 协议

**关键代码：** `protocol/grpc/client.go:138`

```go
conn, err := grpc.Dial(url.Location, dialOpts...)
```

### 3.4 JSONRPC 协议 (短连接)

**文件位置：** `protocol/jsonrpc/`

**连接实现：**
- 每次请求创建新的 TCP 连接
- 通过 `httpReq.Close = true` 强制短连接

**关键代码：** `protocol/jsonrpc/http.go:163`

```go
httpReq.Close = true  // 每次请求后关闭连接
```

### 3.5 REST 协议 (短连接)

**文件位置：** `protocol/rest/`

**连接实现：**
- 使用 Resty 客户端库 (基于 Go 的 HTTP 客户端)
- 默认使用 HTTP 短连接

**关键代码：** `protocol/rest/client/client_impl/resty_client.go:49-66`

```go
func NewRestyClient(restOption *client.RestOptions) client.RestClient {
    client := resty.New()
    client.SetTransport(
        &http.Transport{
            DialContext: func(_ context.Context, network, addr string) (net.Conn, error) {
                c, err := net.DialTimeout(network, addr, restOption.ConnectTimeout)
                return c, err
            },
            IdleConnTimeout: restOption.KeppAliveTimeout,  // 空闲连接超时
        })
    return &RestyClient{client: client}
}
```

---

## 4. 总结

| 协议 | 连接类型 | 底层协议 | 心跳支持 | 连接复用 |
|------|---------|---------|---------|---------|
| Dubbo | 长连接 | TCP (getty) | 是 (60s/5s) | 是 (sync.Map) |
| Triple | 长连接 | HTTP/2/HTTP/3 | 是 (10s/20s) | 是 |
| gRPC | 长连接 | HTTP/2 | 是 (gRPC内置) | 是 |
| JSONRPC | 短连接 | HTTP/1.1 | 无 | 否 |
| REST | 短连接 | HTTP | 无 | 可配置 |

### 长连接实现关键技术点

1. **Dubbo**: 使用 getty 框架的 TCP 连接 + 心跳定时器
2. **Triple/gRPC**: 使用 HTTP/2 的流式连接 + KeepAlive 机制
3. **连接池管理**: 使用 `sync.Map` 缓存 `ExchangeClient`，相同地址复用连接

### 短连接实现关键技术点

1. **JSONRPC**: 每次请求 `net.DialTimeout` 创建新连接 + `httpReq.Close = true`
2. **REST**: 使用 Resty 客户端，每次请求独立发送

---

# 主动通知机制可行性分析

## 1. 概述

主动通知是指在服务端（Provider）下线时，主动向所有已连接的客户端（Consumer）发送关闭通知，使客户端能够立即感知并更新实例状态，避免请求发送到已下线的服务实例。

## 2. 各协议连接管理现状

| 协议 | 连接管理方式 | 是否有获取连接列表 | 是否有关闭连接接口 |
|------|-------------|-------------------|------------------|
| **Dubbo** | getty `sessionMap` | ✅ `RpcServerHandler.sessionMap` | ✅ `session.Close()` |
| **gRPC** | 原生 `grpc.Server` | ❌ 无 | ✅ `GracefulStop()` |
| **Triple** | HTTP/2 `http.Server` | ❌ 无 | ⚠️ 需透传 `GracefulStop` |

### 2.1 Dubbo 协议连接管理

**服务端连接存储：** `remoting/getty/listener.go`

```go
// RpcServerHandler handles server-side getty sessions and request processing.
type RpcServerHandler struct {
    maxSessionNum  int
    sessionTimeout time.Duration
    sessionMap     map[getty.Session]*rpcSession  // 活跃连接列表
    rwlock         sync.RWMutex
    server         *Server
    timeoutTimes   int
}
```

**获取连接列表：**
- `RpcServerHandler.sessionMap` 存储了所有活跃的客户端连接
- 可以通过遍历 `sessionMap` 获取每个连接并发送关闭通知

**关闭连接：**
- `session.Close()` 可以关闭单个连接
- getty 框架支持心跳检测，超时自动关闭

### 2.2 gRPC 协议连接管理

**服务端实现：** `protocol/grpc/server.go`

```go
// Server is a gRPC server
type Server struct {
    grpcServer *grpc.Server
    bufferSize int
}

// GracefulStop gRPC server
func (s *Server) GracefulStop() {
    s.grpcServer.GracefulStop()
}
```

**关闭机制：**
- gRPC 原生支持 `GracefulStop()`，会等待所有正在处理的请求完成后关闭
- 无需遍历连接列表，直接调用即可关闭所有连接

### 2.3 Triple 协议连接管理

**服务端实现：** `protocol/triple/triple_protocol/server.go`

```go
type Server struct {
    addr         string
    mux          *http.ServeMux
    handlers     map[string]*Handler
    httpSrv      *http.Server
    http3Srv     *http3.Server
    tripleConfig *global.TripleConfig
}
```

**关闭机制：**
- Triple 基于 Go 标准库 `http.Server`
- 可以调用 `httpServer.Shutdown()` 实现优雅关闭
- HTTP/2 支持 `GOAWAY` 帧通知客户端停止发送新请求

## 3. 技术实现方案

### 3.1 Dubbo 协议实现

```go
// notifyDubboConsumers 通知 Dubbo 协议 Consumer
func notifyDubboConsumers() error {
    // 1. 获取 getty Server 实例
    gettyServer := getty.GetServer()
    if gettyServer == nil {
        return nil
    }

    // 2. 获取 RpcServerHandler
    rpcHandler := gettyServer.GetRpcHandler()
    if rpcHandler == nil {
        return nil
    }

    serverHandler, ok := rpcHandler.(*RpcServerHandler)
    if !ok {
        return nil
    }

    // 3. 遍历 sessionMap，发送关闭通知
    serverHandler.rwlock.RLock()
    defer serverHandler.rwlock.RUnlock()

    for session := range serverHandler.sessionMap {
        // 发送自定义关闭消息
        sendClosingNotification(session)
        // 延迟关闭，等待客户端感知
        session.Close()
    }

    return nil
}

// sendClosingNotification 发送关闭通知
func sendClosingNotification(session getty.Session) {
    // 构造关闭通知消息
    // Dubbo 协议支持自定义事件消息
    req := &remoting.Request{
        ID:      uuid.New().String(),
        Version: "2.0.2",
        Event:   true,
        Data:    "CLOSING",
    }
    // 发送通知
    session.Write(req)
}
```

**需要修改：**
1. 导出 `getty.GetServer()` 函数
2. 导出 `GetRpcHandler()` 方法
3. 定义 `CLOSING` 事件类型

### 3.2 gRPC 协议实现

```go
// notifyGrpcConsumers 通知 gRPC 协议 Consumer
func notifyGrpcConsumers() error {
    // 1. 获取 gRPC Server 实例
    grpcServer := getGrpcServer()
    if grpcServer == nil {
        return nil
    }

    // 2. 调用 GracefulStop 优雅关闭
    grpcServer.GracefulStop()

    return nil
}
```

**需要修改：**
1. 导出获取 `grpc.Server` 实例的接口

### 3.3 Triple 协议实现

```go
// notifyTripleConsumers 通知 Triple 协议 Consumer
func notifyTripleConsumers() error {
    // 1. 获取 Triple Server 实例
    tripleServer := getTripleServer()
    if tripleServer == nil {
        return nil
    }

    // 2. 获取底层 http.Server
    httpSrv := tripleServer.GetHTTPServer()
    if httpSrv != nil {
        // 优雅关闭，等待正在处理的请求完成
        ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
        defer cancel()
        httpSrv.Shutdown(ctx)
    }

    // 3. 如果是 HTTP/3
    http3Srv := tripleServer.GetHTTP3Server()
    if http3Srv != nil {
        http3Srv.Close()
    }

    return nil
}
```

**需要修改：**
1. 导出获取 Triple Server 实例的接口
2. 导出获取 `http.Server` 和 `http3.Server` 的方法

## 4. 关闭消息协议设计

### 4.1 Dubbo 协议关闭消息

Dubbo 协议支持事件消息，可以发送 `CLOSING` 事件：

```go
// 消息类型定义
const (
    DubboHeartbeatEvent = "R 心跳"      // 心跳
    DubboClosingEvent   = "CLOSING"     // 关闭通知
)

// 在 Consumer 端处理
func handleClosingEvent(session getty.Session) {
    // 标记该服务端实例为 closing 状态
    // 后续请求不再发送到该实例
}
```

### 4.2 HTTP/2 关闭消息

HTTP/2 支持 `GOAWAY` 帧：

```go
// 发送 GOAWAY 帧
// code: 0 表示正常关闭
// debugData: 包含关闭原因
```

### 4.3 gRPC 关闭消息

gRPC 直接关闭连接即可，客户端会收到 `context.Canceled` 或 `context.DeadlineExceeded` 错误。

## 5. 需要解决的问题

| 问题 | 解决方案 |
|------|---------|
| 获取 getty Server 实例 | 导出 `getty.GetServer()` 函数 |
| 获取各协议 Server 实例 | 通过 Protocol 接口或全局注册表获取 |
| Dubbo 发送关闭消息 | 定义 `CLOSING` 事件类型，客户端处理事件 |
| gRPC/Triple 连接列表 | 无需遍历，直接调用 `GracefulStop`/`Shutdown` |
| 超时控制 | 每个协议单独设置超时 |
| 通知重试 | 失败记录日志，注册中心作为兜底 |

## 6. 实现步骤

### 第一阶段：基础设施

1. **修改 getty 包**
   - 导出 `GetServer()` 函数
   - 导出 `GetRpcHandler()` 方法
   - 添加 `CLOSING` 事件类型支持

2. **修改各协议 Server**
   - 导出获取底层 Server 的接口
   - 添加 `GracefulShutdown()` 方法

### 第二阶段：通知实现

3. **实现 Dubbo 协议通知**
   - 遍历 `sessionMap`
   - 发送 `CLOSING` 事件
   - 延迟关闭连接

4. **实现 gRPC/Triple 通知**
   - 调用 `GracefulStop()` / `Shutdown()`

### 第三阶段：完善

5. **添加超时控制**
6. **添加重试机制**
7. **测试验证**

## 7. 风险与注意事项

1. **连接关闭时机**: 发送关闭通知后需要等待客户端感知，建议延迟关闭
2. **并发安全**: 遍历 `sessionMap` 时需要加锁
3. **注册中心兜底**: 即使主动通知失败，注册中心反注册也能作为兜底
4. **向后兼容**: 不影响现有业务逻辑

## 8. 结论

**方案可行**，主动通知机制可以有效提升优雅下线的效率，使客户端更快感知服务端下线，减少无效请求。

**推荐实现顺序**：
1. 先实现 gRPC/Triple（实现简单，直接调用框架方法）
2. 再实现 Dubbo（需要先暴露 getty 连接管理接口）
3. 最后添加超时控制和重试机制
