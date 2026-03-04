# dubbo-go 新版优雅下线技术方案

## 一、设计目标

参考 Kratos 优雅下线设计理念，实现一个更可靠、更灵活的优雅下线机制：

1. **不依赖注册中心**（只兜底）- 主动通知机制替代注册中心通知
2. **依赖标准库 + K8s 探针** - 保证任何场景都能优雅下线
3. **保留配置灵活性** - 支持多协议、多场景配置
4. **保留多协议支持** - 适用于 Dubbo、Triple 等各种协议

---

## 二、核心改进点

### 2.1 架构对比

| 维度          | 旧方案           | 新方案                  |
| ------------- | ---------------- | ----------------------- |
| 流量摘除      | 依赖注册中心通知 | 主动通知 + 注册中心兜底 |
| 请求感知      | Filter 层        | Filter 层 + 滑动窗口    |
| 超时控制      | 简单等待         | 多步骤超时控制          |
| Consumer 感知 | 依赖注册中心     | 被动通知 + 主动通知     |
| K8s 集成      | 无               | 支持健康检查关闭        |

### 2.2 核心流程对比

**旧方案流程**：
```
1. 销毁注册中心
2. 等待消费者更新
3. 拒绝新请求
4. 等待请求完成
5. 销毁协议
```

**新方案流程**（8步）：
```
1. 标记 Closing 状态 ← 新增
2. 主动通知长连接消费者 ← 核心改进
3. 反注册注册中心（兜底）
4. 拒绝新请求
5. 等待请求完成（counter + 滑动窗口）
6. 等待响应完成
7. 销毁协议
8. 回调
```

---

## 三、详细设计

### 3.1 新增配置

```go
// graceful_shutdown/new_config.go
type NewConfig struct {
    TotalTimeout               time.Duration  // 总超时时间 (默认 60s)
    StepTimeout                time.Duration  // 每个步骤超时 (默认 10s)
    ConsumerUpdateWaitTime     time.Duration  // 消费者更新等待 (默认 3s)
    OfflineRequestWindowTimeout time.Duration // 离线请求窗口 (默认 5s)
    ActiveNotifyTimeout        time.Duration  // 主动通知超时 (默认 5s)
    EnableActiveNotify        bool           // 启用主动通知 (默认 true)
    EnableK8sProbe            bool           // 启用 K8s 探针 (默认 false)
    EnableRegistryAsFallback  bool           // 注册中心兜底 (默认 true)
}
```

### 3.2 新增状态

```go
// global/shutdown_config.go
type ShutdownConfig struct {
    // ... 原有字段
    Closing atomic.Bool  // 新增：标记 Provider 正在优雅下线
}
```

### 3.3 GrpcManager 核心逻辑

```go
// graceful_shutdown/grpc_manager.go
type GrpcManager struct {
    config   *NewConfig
    shutdown *global.ShutdownConfig
}

func (m *GrpcManager) StartGracefulShutdown() {
    // Step 1: 标记 Closing 状态
    m.markClosing()

    // Step 2: 主动通知长连接消费者（核心：不依赖注册中心）
    m.activeNotifyConsumers()

    // Step 3: 反注册注册中心（兜底）
    m.deregisterFromRegistry()

    // Step 4: 拒绝新请求
    m.rejectNewRequests()

    // Step 5: 等待请求完成（counter + 滑动窗口）
    m.waitForRequestsComplete()

    // Step 6: 等待响应完成
    m.waitForResponsesComplete()

    // Step 7: 销毁协议
    m.destroyProtocols()

    // Step 8: 执行回调
    m.executeCallbacks()
}
```

---

## 四、通知机制设计

### 4.1 主动通知（核心）

**目的**：不依赖注册中心，通过长连接直接通知 Consumer

**实现**：
```go
func (m *GrpcManager) activeNotifyConsumers() {
    // 1. 获取所有活跃连接
    connections := m.getActiveConnections()

    // 2. 并发通知所有连接
    for _, conn := range connections {
        go m.notifyConnection(conn)
    }

    // 3. 等待确认或超时
    select {
    case <-time.After(m.config.ActiveNotifyTimeout):
        // 超时后进行重试
        m.retryNotifyFailedConnections()
    }
}

func (m *GrpcManager) notifyConnection(conn interface{}) error {
    // 发送 Closing 消息到对端
    // 等待确认
    // 返回结果
}
```

### 4.2 被动通知

**实现**：在 Provider 的响应中携带 Closing 标记

```go
// provider_filter.go
func (f *providerGracefulShutdownFilter) OnResponse(...) result.Result {
    // 如果处于 Closing 状态，在响应中携带标记
    if f.isClosing() {
        result.AddAttachment(constant.GracefulShutdownClosingKey, "true")
    }
    f.shutdownConfig.ProviderActiveCount.Dec()
    return result
}

// consumer_filter.go
func (f *consumerGracefulShutdownFilter) OnResponse(...) result.Result {
    // 检测响应中的 Closing 标记
    if f.isClosingResponse(result) {
        f.markClosingInvoker(invoker)
    }
    return result
}
```

---

## 五、滑动窗口算法

### 5.1 问题背景

原来的方案使用简单的 counter 轮询，可能出现：
- 某一瞬间 counter 为 0，但前后仍有请求

### 5.2 解决方案

使用滑动窗口算法：

```go
func (m *GrpcManager) waitForRequestsComplete() {
    deadline := time.Now().Add(m.config.StepTimeout)
    offlineWindow := m.config.OfflineRequestWindowTimeout

    for time.Now().Before(deadline) {
        activeCount := m.shutdown.ProviderActiveCount.Load()
        lastRequestTime := m.shutdown.ProviderLastReceivedRequestTime.Load()

        // 判断条件：
        // 1. 没有进行中的请求
        // 2. 最后一个请求已经超过了离线窗口时间
        if activeCount == 0 &&
           time.Now().After(lastRequestTime.Add(offlineWindow)) {
            return
        }

        time.Sleep(10 * time.Millisecond)
    }
}
```

---

## 六、Consumer 端处理

### 6.1 接收通知的两种方式

1. **被动通知**：Consumer 的 Filter 在处理 Provider 响应时，检测到响应中携带的 "Closing" 标记
2. **主动通知**：Consumer 的长连接接收到 Provider 主动发送的 Closing 下线请求

### 6.2 核心处理动作

将已下线的 Provider 实例从可用实例列表中移除，后续请求不再发给该实例：

```go
func (f *consumerGracefulShutdownFilter) Invoke(...) result.Result {
    // 检查 invoker 是否正在关闭
    if f.isClosingInvoker(invoker) {
        return &result.RPCResult{Err: errors.New("invoker is closing")}
    }
    // ...
}

func (f *consumerGracefulShutdownFilter) isClosingInvoker(invoker base.Invoker) bool {
    // 检查 invoker 是否在 closingInvokers 中
    // 检查是否已过期
}
```

---

## 七、协议层接口设计

为了支持主动通知，协议层需要实现以下接口：

```go
// 协议接口扩展
type ProtocolWithNotify interface {
    Protocol
    // 获取所有活跃连接
    GetActiveConnections() []Connection
    // 通知连接关闭
    NotifyClosing(conn Connection) error
}

// 连接接口
type Connection interface {
    // 发送关闭通知
    SendClosing() error
    // 等待确认
    WaitAck() error
    // 获取连接标识
    GetID() string
}
```

---

## 八、K8s 探针集成（可选）

### 8.1 设计

参考 Kratos，在关闭服务时同时关闭健康检查：

```go
// 可选功能：需要在配置中启用
func (m *GrpcManager) shutdownK8sProbe() {
    if !m.config.EnableK8sProbe {
        return
    }

    // 关闭健康检查
    // K8s readiness 探针检测到后不再转发流量
    healthServer.Shutdown()
}
```

### 8.2 K8s 配置

```yaml
readinessProbe:
  httpGet:
    path: /healthz
    port: 8080
  failureThreshold: 1
```

---

## 九、超时配置建议

| 场景     | TotalTimeout | StepTimeout | ActiveNotifyTimeout | OfflineWindow |
| -------- | ------------ | ----------- | ------------------- | ------------- |
| 快速响应 | 30s          | 5s          | 3s                  | 3s            |
| 普通     | 60s          | 10s         | 5s                  | 5s            |
| 慢响应   | 120s         | 20s         | 10s                 | 10s           |

**计算公式**：
```
TotalTimeout > StepTimeout * 4 + ActiveNotifyTimeout + OfflineWindow
```

---

## 十、与旧方案兼容性

1. **配置兼容**：旧配置仍然有效，新配置可选
2. **API 兼容**：原有 Init() 方法仍然可用
3. **渐进式迁移**：可以通过配置启用/禁用新功能

---

## 十一、文件变更清单

| 文件                                          | 变更                    |
| --------------------------------------------- | ----------------------- |
| `graceful_shutdown/new_config.go`             | 新增：配置结构          |
| `graceful_shutdown/grpc_manager.go`           | 新增：核心管理器        |
| `global/shutdown_config.go`                   | 修改：新增 Closing 状态 |
| `filter/graceful_shutdown/provider_filter.go` | 修改：支持 Closing 状态 |
| `filter/graceful_shutdown/consumer_filter.go` | 修改：增强通知处理      |