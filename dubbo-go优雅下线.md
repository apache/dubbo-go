# dubbo-go K8s 优雅下线机制详解

## 概述

dubbo-go 的优雅下线机制旨在确保在应用进程终止前，完成正在处理的请求，并阻止新请求进入，从而实现服务平滑下线。本文详细介绍其完整逻辑，特别针对 K8s 环境下的 SIGTERM 信号触发场景。

---

## 一、核心组件

### 1.1 Filter 层

#### Provider 端 Filter (`providerGracefulShutdownFilter`)

**文件位置**: `filter/graceful_shutdown/provider_filter.go`

```go
// 核心功能：
// 1. Invoke 方法：增加请求计数，如果应用正在关闭则拒绝新请求
// 2. OnResponse 方法：减少请求计数，在响应中附加关闭标志
```

- **请求计数**: 使用 `ProviderActiveCount` 原子计数器跟踪正在处理的请求
- **最后请求时间**: 记录 `ProviderLastReceivedRequestTime` 用于判断是否还有进行中的请求
- **拒绝策略**: 当 `RejectRequest=true` 时，新请求被拒绝并返回指定处理器

#### Consumer 端 Filter (`consumerGracefulShutdownFilter`)

**文件位置**: `filter/graceful_shutdown/consumer_filter.go`

```go
// 核心功能：
// 1. Invoke 方法：检查invoker是否正在关闭，拒绝已关闭invoker的请求
// 2. OnResponse 方法：检查响应中是否包含关闭标志，标记对应的invoker为closing状态
```

- **Closing Invoker 管理**: 使用 `sync.Map` 存储正在关闭的 invoker 及其过期时间
- **TTL 机制**: invoker 关闭状态默认 60 秒后过期

### 1.2 配置结构

**文件位置**: `global/shutdown_config.go`

```go
type ShutdownConfig struct {
    Timeout                      string        // 总超时时间 (默认60s)
    StepTimeout                  string        // 步骤超时 (默认3s)
    ConsumerUpdateWaitTime       string        // 消费者更新等待时间 (默认3s)
    RejectRequestHandler        string        // 拒绝请求处理器
    InternalSignal              *bool         // 是否监听内部信号 (默认true)
    OfflineRequestWindowTimeout string        // 离线请求窗口超时 (默认3s)
    RejectRequest               atomic.Bool   // 拒绝新请求标志
    ConsumerActiveCount         atomic.Int32 // 消费者活跃请求数
    ProviderActiveCount         atomic.Int32 // 提供者活跃请求数
    ProviderLastReceivedRequestTime atomic.Time // 提供者最后接收请求时间
}
```

---

## 二、K8s 优雅下线完整流程

### 2.1 流程图

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                           K8s 优雅下线流程                                   │
└─────────────────────────────────────────────────────────────────────────────┘

K8s Pod 终止
    │
    ▼
┌─────────────────────────────────────────────────────────────────────────────┐
│ Step 1: K8s 发送 SIGTERM 信号                                               │
└─────────────────────────────────────────────────────────────────────────────┘
    │
    ▼
┌─────────────────────────────────────────────────────────────────────────────┐
│ Step 2: 捕获信号 → 执行 beforeShutdown()                                    │
│  位置: graceful_shutdown/shutdown.go                                        │
└─────────────────────────────────────────────────────────────────────────────┘
    │
    ├──────────────────┬──────────────────┬──────────────────┐
    ▼                  ▼                  ▼                  ▼
┌─────────────┐  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐
│ 2.1 销毁    │  │ 2.2 等待     │  │ 2.3 拒绝     │  │ 2.4 等待    │
│ 注册中心    │  │ 消费者更新  │  │ 新请求      │  │ 请求完成    │
│ destroy    │  │ waitAnd     │  │ waitFor     │  │ destroy     │
│ Registries │  │ AcceptNew   │  │ SendingAnd  │  │ Protocols   │
│            │  │ Requests    │  │ Receiving   │  │             │
└─────────────┘  └─────────────┘  └─────────────┘  └─────────────┘
    │                  │                  │                  │
    └──────────────────┴──────────────────┴──────────────────┘
    │
    ▼
┌─────────────────────────────────────────────────────────────────────────────┐
│ Step 3: 执行自定义回调 → 退出进程                                            │
└─────────────────────────────────────────────────────────────────────────────┘
```

### 2.2 详细步骤

#### Step 1: 信号监听与触发

**文件位置**: `graceful_shutdown/shutdown.go:84-107`

```go
// Init 函数中注册信号监听
if newOpts.Shutdown.InternalSignal != nil && *newOpts.Shutdown.InternalSignal {
    signals := make(chan os.Signal, 1)
    signal.Notify(signals, ShutdownSignals...)

    go func() {
        sig := <-signals
        logger.Infof("get signal %s, applicationConfig will shutdown.", sig)

        // 设置总超时定时器
        time.AfterFunc(totalTimeout(newOpts.Shutdown), func() {
            logger.Warn("Shutdown gracefully timeout, applicationConfig will shutdown immediately.")
            os.Exit(0)
        })

        beforeShutdown(newOpts.Shutdown)
        os.Exit(0)
    }()
}
```

**监听的信号** (graceful_shutdown_signal_linux.go):

```go
ShutdownSignals = []os.Signal{
    os.Interrupt, os.Kill, syscall.SIGKILL, syscall.SIGSTOP,
    syscall.SIGHUP, syscall.SIGINT, syscall.SIGQUIT, syscall.SIGILL,
    syscall.SIGTRAP, syscall.SIGABRT, syscall.SIGSYS, syscall.SIGTERM,
}
```

> **K8s 场景**: K8s 在 Pod 终止时会发送 SIGTERM 信号，dubbo-go 捕获该信号后开始优雅下线流程。

---

#### Step 2: beforeShutdown() 完整流程

**文件位置**: `graceful_shutdown/shutdown.go:127-144`

```go
func beforeShutdown(shutdown *global.ShutdownConfig) {
    // 2.1 销毁注册中心
    destroyRegistries()

    // 2.2 等待消费者更新 (默认3s)
    // 让消费者有足够时间感知服务下线
    waitAndAcceptNewRequests(shutdown)

    // 2.3 拒绝新请求，等待请求处理完成
    waitForSendingAndReceivingRequests(shutdown)

    // 2.4 销毁所有协议
    destroyProtocols()

    // 2.5 执行自定义回调
    customCallbacks := extension.GetAllCustomShutdownCallbacks()
    for callback := customCallbacks.Front(); callback != nil; callback = callback.Next() {
        callback.Value.(func())()
    }
}
```

---

### 2.3 各步骤详细逻辑

#### 2.1 销毁注册中心

```go
func destroyRegistries() {
    logger.Info("Graceful shutdown --- Destroy all registriesConfig. ")
    registryProtocol := extension.GetProtocol(constant.RegistryProtocol)
    registryProtocol.Destroy()
}
```

**作用**:
- 从注册中心注销当前服务实例
- 消费者将收到服务下线通知
- 新的请求不会再路由到当前实例

---

#### 2.2 等待消费者更新 (waitAndAcceptNewRequests)

**文件位置**: `graceful_shutdown/shutdown.go:153-166`

```go
func waitAndAcceptNewRequests(shutdown *global.ShutdownConfig) {
    logger.Info("Graceful shutdown --- Keep waiting and accept new requests for a short time. ")

    // 等待消费者更新Invokers
    updateWaitTime := parseDuration(shutdown.ConsumerUpdateWaitTime, ...)
    time.Sleep(updateWaitTime)

    // 等待进行中的请求处理完成
    stepTimeout := parseDuration(shutdown.StepTimeout, ...)
    if stepTimeout < 0 {
        return
    }
    waitingProviderProcessedTimeout(shutdown, stepTimeout)
}
```

**等待条件** (`waitingProviderProcessedTimeout`):

```go
func waitingProviderProcessedTimeout(shutdown *global.ShutdownConfig, timeout time.Duration) {
    deadline := time.Now().Add(timeout)
    offlineRequestWindowTimeout := parseDuration(shutdown.OfflineRequestWindowTimeout, ...)

    for time.Now().Before(deadline) &&
        (shutdown.ProviderActiveCount.Load() > 0 ||
         time.Now().Before(shutdown.ProviderLastReceivedRequestTime.Load().Add(offlineRequestWindowTimeout))) {
        time.Sleep(10 * time.Millisecond)
        logger.Infof("waiting for provider active invocation count = %d", ...)
    }
}
```

**退出条件** (满足任一即退出):
1. **ProviderActiveCount == 0**: 所有请求处理完成
2. **最后请求时间超过 OfflineRequestWindowTimeout**: 即使有请求在处理，也认为已超出合理窗口

---

#### 2.3 拒绝新请求 & 等待消费者请求完成

**文件位置**: `graceful_shutdown/shutdown.go:183-187`

```go
func waitForSendingAndReceivingRequests(shutdown *global.ShutdownConfig) {
    logger.Info("Graceful shutdown --- Keep waiting until sending/accepting requests finish or timeout. ")

    // 设置拒绝标志
    shutdown.RejectRequest.Store(true)

    // 等待消费者请求完成
    waitingConsumerProcessedTimeout(shutdown)
}
```

**Provider Filter 中的拒绝逻辑** (`provider_filter.go:67-80`):

```go
func (f *providerGracefulShutdownFilter) Invoke(...) result.Result {
    if f.rejectNewRequest() {
        logger.Info("The application is closing, new request will be rejected.")
        // 返回拒绝执行处理器
        handler := constant.DefaultKey
        if f.shutdownConfig != nil && len(f.shutdownConfig.RejectRequestHandler) > 0 {
            handler = f.shutdownConfig.RejectRequestHandler
        }
        rejectedExecutionHandler, err := extension.GetRejectedExecutionHandler(handler)
        // ...
    }
    // 增加请求计数
    f.shutdownConfig.ProviderActiveCount.Inc()
    f.shutdownConfig.ProviderLastReceivedRequestTime.Store(time.Now())
    return invoker.Invoke(ctx, invocation)
}
```

**Consumer Filter 中的响应处理** (`consumer_filter.go:95-98`):

```go
func (f *consumerGracefulShutdownFilter) OnResponse(...) result.Result {
    // 检查响应中是否包含关闭标志
    if f.isClosingResponse(result) {
        f.markClosingInvoker(invoker)
    }
    f.shutdownConfig.ConsumerActiveCount.Dec()
    return result
}
```

---

#### 2.4 销毁协议

**文件位置**: `graceful_shutdown/shutdown.go:205-214`

```go
func destroyProtocols() {
    logger.Info("Graceful shutdown --- Destroy protocols. ")

    proMu.Lock()
    defer proMu.Unlock()
    for name := range protocols {
        extension.GetProtocol(name).Destroy()
    }
}
```

**注册协议** (在协议初始化时):

```go
// 例如在 triple 协议中
graceful_shutdown.RegisterProtocol("tri")
```

---

#### 2.5 执行自定义回调

```go
customCallbacks := extension.GetAllCustomShutdownCallbacks()
for callback := customCallbacks.Front(); callback != nil; callback = callback.Next() {
    callback.Value.(func())()
}
```

允许用户注册自定义的关闭回调函数。

---

## 三、K8s 配置建议

### 3.1 K8s Pod 生命周期配置

```yaml
apiVersion: v1
kind: Pod
spec:
  containers:
  - name: dubbo-app
    lifecycle:
      preStop:
        exec:
          command: ["/bin/sh", "-c", "sleep 5"]
```

**说明**:
- `preStop` 钩子在 K8s 发送 SIGTERM 之前执行
- 建议执行一个短暂的 sleep，确保 dubbo-go 有时间开始优雅下线
- sleep 时间应小于 `terminationGracePeriodSeconds`

### 3.2 推荐配置

```yaml
dubbo:
  shutdown:
    timeout: 60s           # 总超时
    step-timeout: 10s      # 每个步骤超时
    consumer-update-wait-time: 3s  # 消费者更新等待
    offline-request-window-timeout: 5s  # 离线请求窗口
    internal-signal: true  # 监听信号
```

### 3.3 超时配置计算

```
Timeout > StepTimeout * 3 + ConsumerUpdateWaitTime + offlineRequestWindowTimeout
```

假设:
- 消费者感知服务下线时间: 10s
- 99.9% 请求响应时间: 2s

则:
- StepTimeout > 10s + 2s * 3 = 16s，建议 20s
- Timeout > 20s * 3 = 60s

---

## 四、配置参数说明

| 参数                             | 默认值 | 说明                           |
| -------------------------------- | ------ | ------------------------------ |
| `timeout`                        | 60s    | 优雅下线总超时，超过后强制退出 |
| `step-timeout`                   | 3s     | 每个步骤的超时时间             |
| `consumer-update-wait-time`      | 3s     | 等待消费者更新invoker的时间    |
| `offline-request-window-timeout` | 3s     | 最后请求到达后的窗口时间       |
| `internal-signal`                | true   | 是否监听系统信号               |
| `reject-handler`                 | -      | 拒绝请求时的处理器             |

---

## 五、关键源码文件

| 文件                                          | 职责               |
| --------------------------------------------- | ------------------ |
| `graceful_shutdown/shutdown.go`               | 优雅下线核心流程   |
| `graceful_shutdown/options.go`                | 配置选项           |
| `graceful_shutdown/common.go`                 | 工具函数           |
| `filter/graceful_shutdown/provider_filter.go` | Provider 端 Filter |
| `filter/graceful_shutdown/consumer_filter.go` | Consumer 端 Filter |
| `global/shutdown_config.go`                   | 配置结构定义       |
| `common/constant/key.go`                      | 常量定义           |

---

## 六、注意事项

1. **信号兼容性**: 确保 K8s 的 `terminationGracePeriodSeconds` 大于 `dubbo.shutdown.timeout`
2. **注册中心**: 销毁注册中心后，服务立即从注册表消失，消费者会立即更新 invoker
3. **请求计数**: Provider 和 Consumer 端的活跃请求计数是独立的
4. **强制退出**: 如果在 `timeout` 时间内未完成优雅下线，进程会被强制终止