# dubbo-go vs Kratos 优雅下线对比分析

## 概述

本文对 dubbo-go 和 Kratos 两个框架的 K8s 优雅下线机制进行详细对比分析，从架构设计、核心流程、各维度特性等方面进行全面对比。

---

## 一、架构设计对比

| 维度         | dubbo-go                                         | Kratos                                                       |
| ------------ | ------------------------------------------------ | ------------------------------------------------------------ |
| **设计模式** | Filter 拦截层 + 手动计数                         | 依赖标准库 + 生命周期钩子                                    |
| **请求感知** | 手动维护 ProviderActiveCount/ConsumerActiveCount | 依赖 `http.Server.Shutdown()` / `grpc.Server.GracefulStop()` |
| **组件层次** | Filter → Protocol → Registry                     | App → Server (HTTP/gRPC) → Registrar                         |

---

## 二、核心流程对比

### dubbo-go 流程 (5步)

```
1. 销毁注册中心 (立即从注册表删除)
       ↓
2. 等待消费者更新 (ConsumerUpdateWaitTime: 3s)
       ↓
3. 拒绝新请求 (RejectRequest = true)
       ↓
4. 等待进行中请求完成 (ProviderActiveCount == 0)
       ↓
5. 销毁协议 + 执行自定义回调
```

### Kratos 流程 (3步)

```
1. 执行 BeforeStop 钩子 + 从注册中心注销
       ↓
2. 取消上下文 + Server.Shutdown()/GracefulStop()
       ↓
3. 执行 AfterStop 钩子 + 进程退出
```

---

## 三、各维度详细对比

### 1. 信号监听

| 方面         | dubbo-go                                     | Kratos                   |
| ------------ | -------------------------------------------- | ------------------------ |
| 监听信号     | SIGTERM, SIGINT, SIGQUIT, SIGHUP, SIGKILL 等 | SIGTERM, SIGQUIT, SIGINT |
| **关键差异** | 监听更多信号（包括 SIGHUP 用于 reload）      | 仅监听退出信号           |

**dubbo-go 胜在通用性** - 适配更多场景

---

### 2. 服务注册中心注销时机

| 方面         | dubbo-go               | Kratos                 |
| ------------ | ---------------------- | ---------------------- |
| **执行时机** | 优雅下线第一步立即执行 | 同样第一步执行         |
| **效果**     | 消费者立即感知服务下线 | 消费者立即感知服务下线 |

**平局** - 两者都先注销服务，避免新请求进入

---

### 3. 请求处理机制

| 方面             | dubbo-go                             | Kratos                                  |
| ---------------- | ------------------------------------ | --------------------------------------- |
| **拒绝新请求**   | Filter 层主动检查 RejectRequest 标志 | 依赖 `Server.Shutdown()` 停止接收新连接 |
| **等待完成**     | 轮询 ProviderActiveCount == 0        | 依赖标准库内部机制                      |
| **离线请求窗口** | ✅ 有 OfflineRequestWindowTimeout     | ❌ 无                                    |

**dubbo-go 胜在精细控制** - 提供更细粒度的请求控制

```go
// dubbo-go 的等待条件 (更全面)
for time.Now().Before(deadline) &&
    (shutdown.ProviderActiveCount.Load() > 0 ||
     time.Now().Before(shutdown.ProviderLastReceivedRequestTime.Load().Add(offlineRequestWindowTimeout)))
```

---

### 4. 超时控制

| 方面               | dubbo-go                    | Kratos                     |
| ------------------ | --------------------------- | -------------------------- |
| **配置项**         | Timeout (总超时)            | stopTimeout (优雅下线超时) |
| **步骤超时**       | StepTimeout                 | 无 (统一超时)              |
| **消费者更新等待** | ConsumerUpdateWaitTime      | 无                         |
| **离线请求窗口**   | OfflineRequestWindowTimeout | 无                         |

**dubbo-go 胜在灵活性** - 提供多层超时配置

```yaml
# dubbo-go 配置
dubbo:
  shutdown:
    timeout: 60s
    step-timeout: 10s
    consumer-update-wait-time: 3s
    offline-request-window-timeout: 5s
```

```go
// Kratos 配置
kratos.New(
    kratos.Name("helloworld"),
    kratos.StopTimeout(10 * time.Second),
)
```

---

### 5. gRPC 特殊处理

| 方面         | dubbo-go         | Kratos                            |
| ------------ | ---------------- | --------------------------------- |
| **健康检查** | ❌ 无             | ✅ health.Shutdown() → NOT_SERVING |
| **流量摘除** | 依赖注册中心注销 | K8s 探针可直接感知                |

**Kratos 胜在 K8s 集成** - 健康检查关闭后，K8s readiness 探针可立即感知

```go
// Kratos gRPC 优雅下线关键代码
func (s *Server) Stop(ctx context.Context) error {
    // 关闭健康检查 - 重要：让 K8s 移除该 Pod
    s.health.Shutdown()

    go func() {
        s.GracefulStop()  // 等待所有 rpc 完成
    }()

    select {
    case <-done:
    case <-ctx.Done():  // 超时则强制停止
        s.Server.Stop()
    }
    return nil
}
```

---

### 6. 用户钩子

| 方面       | dubbo-go                         | Kratos     |
| ---------- | -------------------------------- | ---------- |
| **停止前** | 自定义回调 (通过 extension 注册) | BeforeStop |
| **停止后** | 无                               | AfterStop  |

**Kratos 胜在易用性** - API 更简洁

```go
// Kratos
kratos.New(
    kratos.Name("helloworld"),
    kratos.StopTimeout(10 * time.Second),

    // 停止前执行（如：关闭数据库连接、清理缓存）
    kratos.BeforeStop(func(ctx context.Context) error {
        return nil
    }),

    // 停止后执行（如：日志清理、发送通知）
    kratos.AfterStop(func(ctx context.Context) error {
        return nil
    }),
)

// dubbo-go
extension.SetCustomShutdownCallback(func() {...})
```

---

### 7. 代码复杂度

| 方面           | dubbo-go                     | Kratos                         |
| -------------- | ---------------------------- | ------------------------------ |
| **实现行数**   | ~500+ 行 (Filter + Shutdown) | ~200+ 行                       |
| **依赖标准库** | 自实现逻辑                   | 利用 net/http 和 grpc 标准能力 |
| **维护成本**   | 较高                         | 较低                           |

**Kratos 胜在简洁** - 复用标准库能力，减少自研代码

---

### 8. 对外部组件的依赖

| 方面               | dubbo-go                   | Kratos                   |
| ------------------ | -------------------------- | ------------------------ |
| **依赖类型**       | 强依赖注册中心             | 依赖标准库 + K8s 探针    |
| **依赖注册中心**   | ✅ 是                       | ❌ 否                     |
| **流量摘除方式**   | 注册中心通知 + Filter 拒绝 | Server 层关闭 + 健康检查 |
| **注册中心故障时** | 优雅下线可能失效           | 不受影响                 |

**Kratos 胜在稳定性** - 不依赖外部组件，优雅下线更可靠

#### dubbo-go 的注册中心依赖问题

dubbo-go 的优雅下线依赖以下链路：

```
1. 调用 UnRegister() 从注册中心删除服务实例
       ↓
2. 注册中心向消费者推送下线通知
       ↓
3. 消费者收到通知后更新 Invoker
       ↓
4. 不再向该 Provider 发送请求
```

**问题场景**：

| 场景         | 影响                                        |
| ------------ | ------------------------------------------- |
| 注册中心宕机 | `UnRegister()` 可能失败，消费者完全感知不到 |
| 通知延迟     | 消费者继续往正在下线的 Provider 发送请求    |
| 通知丢失     | 部分消费者不知道服务已下线                  |
| 网络分区     | 通知无法及时送达                            |

**dubbo-go 的兜底机制**：

```go
// 1. ConsumerUpdateWaitTime - 固定等待时间
time.Sleep(updateWaitTime)

// 2. Provider 端主动拒绝请求
if f.rejectNewRequest() {
    return rejectedExecutionHandler.RejectedExecution(...)
}

// 3. OfflineRequestWindowTimeout - 避免请求"饿死"
time.Now().Before(shutdown.ProviderLastReceivedRequestTime.Load().Add(offlineRequestWindowTimeout))
```

**问题**：这些都是"被动防御"，不是"主动控制"

#### Kratos 的控制式优雅下线

Kratos 不依赖注册中心，直接在协议层控制：

```
1. health.Shutdown() → K8s 探针立即感知，停止分发流量
       ↓
2. Server.Shutdown()/GracefulStop() → 直接在 Server 层拒绝新请求
       ↓
3. 等待存量请求处理完成
```

**优势**：
- 不依赖任何外部组件
- K8s readiness 探针可直接感知服务状态
- 协议层直接控制，无需中间环节

---

### 9. 可靠性对比

| 方面                   | dubbo-go     | Kratos     |
| ---------------------- | ------------ | ---------- |
| 注册中心故障时优雅下线 | ❌ 可能失效   | ✅ 正常工作 |
| 网络抖动时优雅下线     | ⚠️ 可能延迟   | ✅ 正常工作 |
| K8s 探针感知           | ❌ 无         | ✅ 立即感知 |
| 流量摘除速度           | 依赖注册中心 | 实时       |

**Kratos 胜在可靠性** - 不依赖外部组件，任何场景都能保证优雅下线

---

## 五、优劣势总结

### dubbo-go 优势

| 优势              | 说明                                      |
| ----------------- | ----------------------------------------- |
| ✅ 细粒度控制      | 请求级别计数，可精确控制每个请求          |
| ✅ 多层超时        | 4 种超时配置，适应复杂场景                |
| ✅ 离线请求窗口    | 避免最后一个请求被"饿死"                  |
| ✅ 通用性          | 不仅适用于 HTTP/gRPC，支持所有 Dubbo 协议 |
| ✅ Consumer 端感知 | Provider 下线时 Consumer 端也能正确处理   |

### dubbo-go 劣势

| 劣势             | 说明                           |
| ---------------- | ------------------------------ |
| ❌ 复杂度高       | 500+ 行代码，维护成本高        |
| ❌ 需要手动计数   | Filter 侵入式代码              |
| ❌ K8s 探针集成弱 | 无健康检查关闭机制             |
| ❌ 依赖注册中心   | 注册中心故障时优雅下线可能失效 |

---

### Kratos 优势

| 优势         | 说明                             |
| ------------ | -------------------------------- |
| ✅ K8s 集成好 | health.Shutdown() 让探针立即感知 |
| ✅ 简洁优雅   | 依赖标准库，代码量少             |
| ✅ 钩子易用   | BeforeStop/AfterStop API 友好    |
| ✅ 统一超时   | 一个 stopTimeout 搞定            |

### Kratos 劣势

| 劣势             | 说明               |
| ---------------- | ------------------ |
| ❌ 依赖特定协议   | 仅支持 HTTP/gRPC   |
| ❌ 无离线请求窗口 | 可能导致请求"饿死" |
| ❌ 单一超时       | 不适合复杂场景     |

---

## 六、适用场景建议

| 场景                          | 推荐                          |
| ----------------------------- | ----------------------------- |
| **K8s 环境下 HTTP/gRPC 服务** | Kratos (更简洁，K8s 集成更好) |
| **Dubbo RPC 协议服务**        | dubbo-go (必须使用)           |
| **需要精细控制请求**          | dubbo-go                      |
| **微服务快速开发**            | Kratos                        |
| **多协议混合部署**            | dubbo-go                      |

---

## 七、综合对比表

| 对比项          | dubbo-go               | Kratos           | 胜出     |
| --------------- | ---------------------- | ---------------- | -------- |
| K8s 探针集成    | ❌                      | ✅                | Kratos   |
| 请求控制精细度  | ✅                      | ❌                | dubbo-go |
| 代码简洁性      | ❌                      | ✅                | Kratos   |
| 配置灵活性      | ✅                      | ❌                | dubbo-go |
| 通用性          | ✅ (多协议)             | ❌ (仅 HTTP/gRPC) | dubbo-go |
| Consumer 端感知 | ✅                      | ❌                | dubbo-go |
| 用户钩子易用性  | ❌                      | ✅                | Kratos   |
| 外部组件依赖    | ❌ (强依赖注册中心)     | ✅ (无依赖)       | Kratos   |
| 可靠性          | ❌ (注册中心故障时失效) | ✅ (稳定可靠)     | Kratos   |

---

## 八、结论

**总体评价**:

- **Kratos** 更符合 Go 的哲学，利用标准库能力，代码简洁，K8s 集成更好
- **dubbo-go** 更适合复杂场景，提供更精细的控制，但复杂度较高

**推荐选择**:

| 场景                        | 推荐框架     |
| --------------------------- | ------------ |
| 标准 K8s + gRPC/HTTP 微服务 | **Kratos**   |
| Dubbo RPC 协议服务          | **dubbo-go** |
| 需要精细控制请求处理        | **dubbo-go** |
| 快速开发微服务              | **Kratos**   |
| 多协议混合部署              | **dubbo-go** |

如果你的场景是标准的 K8s + gRPC/HTTP 服务，**Kratos 的方案更优雅**；如果需要处理复杂的 RPC 场景或使用 Dubbo 协议，则 **dubbo-go 更合适**。