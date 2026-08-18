# Dubbo-Go 外部扩展运行时设计

> 关联 Issue：[apache/dubbo-go#3672](https://github.com/apache/dubbo-go/issues/3672)

## 1. 背景

Issue #3672 的核心目标不是在 dubbo-go 中直接加入 Hystrix，而是在
dubbo-go 与外部扩展之间建立一套通用协议。Hystrix 可以作为第一个外部实现，
用于验证这套协议是否能够支持配置加载、生命周期、资源隔离和 Filter 注入。

dubbo-go 核心负责：

- 注册和查找扩展 Definition；
- 管理 Instance、Client、Server 层级的扩展生命周期；
- 加载扩展原始配置并处理配置优先级；
- 校验 Scope 和 Role；
- 在 Dial、Register 阶段绑定 RPC Resource；
- 合并、排序、去重扩展贡献的 Filter。

外部扩展负责：

- 自己的 typed config；
- 自己的 typed option；
- 解码自己的 YAML 子树；
- 初始化、清理及运行时行为；
- 按 service 或 method 管理运行时状态；
- 创建自己的 Filter。

核心不能依赖具体扩展名称，不能对具体扩展配置做类型断言，也不能把可变扩展
配置保存在进程级全局变量中。

## 2. 总体原则

```text
extension.Option 负责配置什么
WithExtension    负责在哪个层级声明
Context          负责传递 Scope、Role、Config 和 Resource
Definition       负责扩展自己的解码、初始化、清理和 Filter 贡献
```

`Scope` 和 `Role` 是两个独立维度：

- `Scope` 表示生命周期层级；
- `Role` 表示 consumer/provider 角色。

扩展 Option 不得携带 Scope、consumer/provider 或 Client/Server 选择信息。位置由
外层 dubbo-go API 决定。

## 3. 层级、Scope 和 Role

显式创建 Instance 时，Client 和 Server 是它的子上下文：

```text
Instance
├── Client / Consumer
└── Server / Provider
```

现有 API 也允许直接创建：

```go
client.NewClient()
server.NewServer()
```

因此不能假设 Client 或 Server 一定存在显式 Instance 父对象。直接创建的 Client
或 Server 使用隐式根上下文，但后续初始化流程与 Instance 子上下文保持一致。

`Definition.Scopes` 需要同时表达多个支持层级，因此 Scope 使用位标志：

```go
type Scope uint8

const (
	InstanceScope Scope = 1 << iota
	ClientScope
	ServerScope
)

const RoleNone common.RoleType = -1
```

唯一合法的组合为：

```text
InstanceScope + RoleNone
ClientScope   + common.CONSUMER
ServerScope   + common.PROVIDER
```

核心在调用任何扩展代码前完成组合校验。扩展不能修改核心传入的 Scope 和 Role。

## 4. 公共入口

三个框架 Option 的类型边界继续保留：

```go
func dubbo.WithExtension(opts ...extension.Option) InstanceOption
func client.WithExtension(opts ...extension.Option) ClientOption
func server.WithExtension(opts ...extension.Option) ServerOption
```

它们的语义分别为：

```text
dubbo.WithExtension  -> 在 Instance 层声明
client.WithExtension -> ClientScope + ConsumerRole
server.WithExtension -> ServerScope + ProviderRole
```

统一的是 `extension.Option`、Definition 和初始化流程，不统一的是
`InstanceOption`、`ClientOption`、`ServerOption`。

例如：

```go
client.WithExtension(
	hystrix.WithConfig(
		hystrix.WithTimeout(1000),
	),
)

server.WithExtension(
	hystrix.WithConfig(
		hystrix.WithTimeout(1500),
	),
)
```

`hystrix.WithConfig` 只描述 Hystrix 配置。Client/Server 入口负责决定使用
consumer 还是 provider 上下文。

如果 Hystrix 没有声明 `InstanceScope`，下面的调用在 Instance 创建阶段返回错误：

```go
dubbo.WithExtension(hystrix.WithConfig(...))
```

Instance 层声明只有在 Definition 同时支持相应子 Scope 时，才可以作为子上下文
Plan 的输入来源。即便进行派生，子上下文仍会重新创建自己的 Config 和 Runtime，
不会共享 Instance 的可变配置指针。

## 5. Extension Option

通用 Option 协议为：

```go
type Option interface {
	Prefix() string
	Apply(config any) error
}
```

`Prefix` 表示 Option 属于哪个 Definition。核心只负责按 Prefix 分组并调用
`Apply`，具体配置类型的断言和修改由扩展自己完成。

扩展内部仍应保留强类型 Option，只把最外层包装成 `extension.Option`：

```go
type ConfigOption func(*Config) error

func WithTimeout(timeout time.Duration) ConfigOption
func WithConfig(opts ...ConfigOption) extension.Option
```

相同 Prefix 的 Option 按声明顺序执行。子上下文存在继承输入时，Instance 层
Option 先执行，Client/Server 层 Option 后执行。

## 6. Definition 与注册

建议在现有 `common/extension` 包中增加通用 Definition：

```go
type Definition struct {
	Prefix    string
	Scopes    Scope
	NewConfig func() any
	Decode    func(RawConfig, any) error
	Init      func(*Context) error
	Filters   func(*Context) ([]FilterSpec, error)
	Close     func(*Context) error
}
```

外部扩展通过 side-effect import 注册不可变 Definition：

```go
func init() {
	extension.MustRegister(extension.Definition{
		Prefix:    "hystrix",
		Scopes:    extension.ClientScope | extension.ServerScope,
		NewConfig: newHystrixConfig,
		Decode:    decodeHystrixConfig,
		Init:      initHystrix,
		Filters:   hystrixFilters,
		Close:     closeHystrix,
	})
}
```

注册约束：

- Prefix、Scopes 和 NewConfig 必填；
- 相同 Prefix 重复注册时失败；
- 全局只保存不可变 Definition，不保存扩展配置和运行时状态；
- Runtime 使用 Definition 快照，后续注册不影响已创建的 Runtime；
- Definition 初始化顺序必须稳定，例如按 Prefix 字典序；
- Close 按 Init 的相反顺序执行；
- Decode、Init、Filters、Close 在不需要相应能力时可以为空。

## 7. RawConfig

核心只负责定位：

```text
dubbo.extensions.<prefix>
```

公共协议不能暴露 koanf、YAML parser 或 config center 的具体类型。RawConfig
使用保留原始 key 的不可变结构树：

```go
type RawNode interface {
	Child(key string) (RawNode, bool)
	Value() any
	Present() bool
}

type RawConfig struct {
	Full     RawNode
	Selected RawNode
}
```

约束如下：

- `Child` 只按完整的直接子 key 查找，不解释 `.`、`:` 等字符；
- `Value` 返回脱离核心配置树的值，扩展不能反向修改核心配置；
- 扩展自己选择 YAML、mapstructure 等工具完成 typed decode；
- 核心不解析 resource/command key，不理解扩展字段。

核心根据 Scope/Role 构建两个视图：

```text
Scope           Full                              Selected
InstanceScope   dubbo.extensions.<prefix>         Full
ClientScope     dubbo.extensions.<prefix>         Full.consumer
ServerScope     dubbo.extensions.<prefix>         Full.provider
```

扩展通常从 Selected 解码 consumer/provider 配置；如果扩展定义了公共字段，也可以
访问 Full。核心不会继续解释 Full 或 Selected 内部的结构。

示例：

```yaml
dubbo:
  extensions:
    hystrix:
      consumer:
        "greet.GreetService:::Greet":
          timeout: 1000
      provider:
        "greet.GreetService:::Greet":
          timeout: 1500
```

`greet.GreetService:::Greet` 必须作为一个完整 key 保留，不能通过下面这种 dotted
path 读取：

```text
dubbo.extensions.hystrix.consumer.greet.GreetService:::Greet
```

Loader 必须从与普通配置相同的最终合并结果生成 RawConfig，包括 active profile
和 config center 覆盖。扩展子树需要在 delimiter flatten 之前保留，不能尝试从
koanf 已展平的 key 中恢复。

## 8. 激活方式和配置优先级

当前 Scope 中满足任一条件时激活扩展：

- 显式传入了对应 Prefix 的 Option；
- 当前 Scope/Role 对应的 YAML Selected 分支存在。

因此同时支持：

```text
side-effect import + YAML
side-effect import + typed option
```

仅仅 import 一个扩展但没有配置或 Option，不会自动初始化所有已注册扩展。

配置优先级固定为：

```text
NewConfig 默认值 < YAML < typed option
```

每个激活的扩展执行：

```text
NewConfig
    -> Decode(RawConfig)
    -> Apply(Instance inherited options)
    -> Apply(Client/Server local options)
    -> Init
```

YAML 中配置了未注册 Prefix，或 Option 指向未注册 Prefix 时，构造过程返回包含
Prefix、Scope 和 Role 的错误。

## 9. ExtensionPlan 与 Runtime 隔离

Instance 保存不可变 Plan，而不是保存准备共享给子上下文的 live Config：

```text
ExtensionPlan
├── Definition snapshot
├── RawConfig snapshot
└── ordered Option declarations
```

每个 Instance、Client、Server 都从 Plan 创建自己的 Extension Runtime。每个激活
扩展都调用一次 NewConfig，得到当前上下文独占的 typed config。

Instance 创建 Client/Server 时，只派生 Definition、RawConfig 和 Option 等不可变
输入，然后重新执行配置构建流程。以下对象不能作为可变指针跨上下文共享：

- typed extension config；
- Context；
- Filter；
- circuit breaker、limiter、counter 等运行时状态。

直接创建的 Client/Server 从隐式根 Plan 创建相同类型的 Runtime，从而保证直接
创建与 Instance 子上下文的行为一致。

Init 完成后，Config 按只读对象使用。按资源变化的可变状态应保存在资源 Runtime
或 Filter 中，不能写入进程级全局配置。

## 10. Resource

Resource 表示扩展当前作用于哪个具体 RPC 资源，不表示默认配置：

```go
type Resource struct {
	ServiceKey string
	Interface  string
	Method     string
	Group      string
	Version    string
}

type Context struct {
	Scope    Scope
	Role     common.RoleType
	Config   any
	Resource *Resource
}
```

Resource 使用指针是为了明确表示尚未绑定资源：

```text
Resource == nil
    Instance/Client/Server 正处于初始化阶段，尚未绑定具体 RPC 资源。

Resource != nil
    当前 Context 是绑定到具体 service 或 method 的副本。
```

`Resource == nil` 不表示“默认资源”。Init 不得依赖 Resource，资源绑定在 Init
成功之后发生。

Resource 由核心构造，扩展按只读对象使用。ServiceKey 必须由核心中的唯一规范函数
生成。结构字段足够使用时，扩展不得自行拼接或解析 ServiceKey 格式。

Resource 可以帮助扩展：

- 按 service 或 method 选择配置；
- 为不同服务创建独立 Filter、熔断器或限流器；
- 隔离同一个 Client 下的多个服务；
- 在 Dial/Register 阶段提前校验资源配置；
- 避免直接依赖 Dubbo URL、Invoker 和 Invocation 的内部结构。

Resource 只提供统一资源身份，不会自动完成状态隔离。扩展必须基于 Resource 创建
或查找资源级运行时状态。

## 11. Resource 绑定时机

Client/Server 初始化时还不知道具体服务：

```text
NewClient/NewServer
    -> Init(Context{Resource: nil})
```

Client 在 ReferenceOptions 确定后，于 `Dial`、`NewService`、
`DialWithDefinition` 阶段绑定 Resource。Server 在 `Register` 或
`RegisterService` 阶段绑定 Resource。

```go
resource := &extension.Resource{
	ServiceKey: "payment.PaymentService:test:v1",
	Interface:  "payment.PaymentService",
	Group:      "test",
	Version:    "v1",
}

resourceCtx := baseCtx
resourceCtx.Resource = resource
specs, err := definition.Filters(&resourceCtx)
```

基础 Context 不允许被修改，每个资源使用独立的 Context 副本。成功生成的 FilterSpec
可以按规范 Resource identity 缓存，并且并发绑定必须保证无数据竞争。

推荐的方法级处理方式为：

```text
Dial/Register 阶段绑定 Method == "" 的 service 级 Resource
Invoke 阶段通过 invocation.MethodName() 选择 method 级配置和状态
```

只有必须提前为每个方法创建独立状态的扩展，才在绑定阶段根据已知方法列表创建带
Method 的 Resource。这不是核心默认行为。

例如，同一个 Client 可以产生两个相互隔离的运行时：

```text
payment.PaymentService -> timeout 1s -> circuit breaker A
user.UserService       -> timeout 3s -> circuit breaker B
```

熔断器 A 打开不能影响熔断器 B。

## 12. Filter 贡献与合并

扩展自动贡献 Filter，用户不能再通过字符串手工指定 Filter：

```go
type FilterSpec struct {
	ID      string
	Factory func() filter.Filter
	Order   int
}
```

FilterSpec 约束：

- ID 必填，仅用于核心内部标识、排序、去重和诊断，不暴露给用户；
- 扩展应为 ID 增加命名空间，避免不同扩展发生冲突；
- Factory 必填，可以捕获当前资源的配置和 Runtime；
- Factory 为当前 chain 创建 Filter，不读取进程级可变扩展配置；
- Factory 返回 nil 或 ID 为空时，核心直接报告 FilterSpec 非法。

用户不再通过字符串指定 Filter，也不再使用 `default` 占位符或 `-key` 删除语义。
逻辑上的最终结果只包含框架内部 Filter 和扩展自动贡献的 FilterSpec：

```text
现有框架默认 Filter
    + 按 Order 排序的自动扩展 FilterSpec
```

合并规则：

1. 展开框架内部 Filter 和自动扩展 FilterSpec；
2. 按 `Order` 稳定排序；
3. 相同 ID 且来自同一资源上下文时去重；
4. 不同扩展返回相同 ID 时直接报错，不能依赖注册顺序；
5. 每个 Factory 独立创建当前资源的 Filter，不注册临时全局名字。

`protocolwrapper` 只接收已经解析的 FilterSpec，不再解析 URL 中的 filter name，
也不再依赖全局 named filter registry。

## 13. 错误、回滚与关闭

以下情况导致上下文创建失败：

- Scope/Role 组合非法；
- Definition 不支持当前 Scope；
- Prefix 未注册；
- 扩展 YAML 解码失败；
- typed option Apply 失败；
- Init 失败。

以下情况导致资源绑定失败，并从 Dial/Register 返回错误：

- Resource 非法或缺少必要字段；
- 资源配置校验失败；
- Filters 返回错误；
- FilterSpec 非法或冲突。

这些错误不能只记录日志。错误应包含可用的 Prefix、Scope、Role 和 Resource identity。

如果某个扩展失败时前面的扩展已经完成 Init，核心按相反顺序调用已初始化扩展的
Close。Instance、Client、Server 正常关闭时，也要确保各自拥有的 Runtime 只关闭
一次。清理错误需要报告，但不能覆盖最初的构造错误。

## 14. Load 行为

使用声明式配置时，用户只需要 side-effect import：

```go
import _ "example.org/dubbo-go-extension-hystrix"
```

`dubbo.Load` 的流程为：

1. 加载并合并框架配置源；
2. 将 `dubbo.extensions` 保留为 RawConfig；
3. 创建 Instance ExtensionPlan；
4. Load 启动 consumer/provider 时创建 ClientScope、ServerScope Runtime；
5. Dial/Register 时绑定具体服务。

核心只激活存在 Selected YAML 分支或显式 Option 的扩展。仅注册但未使用的
Definition 不产生运行时副作用。

## 15. 热更新边界

第一阶段将 Extension Runtime 及其 Config 视为构造完成后不可变。现有热更新
机制不能直接修改 live Runtime 中的 `dubbo.extensions.*`。

后续如果需要动态更新，应设计事务化流程：

```text
构建新 Runtime
    -> 完整校验和初始化
    -> 原子替换
    -> 关闭旧 Runtime
```

届时可以增加明确的 Reload 协议。本设计不允许静默地原地修改 `Config any`。

## 16. 完整流程

```text
side-effect import 注册不可变 Definition
    -> WithExtension 保存有序 Option
    -> Loader 保留完整 RawConfig
    -> 核心校验 Scope 和 Role
    -> NewConfig 创建上下文独占 typed config
    -> 扩展解码 Selected YAML
    -> 核心依次应用 inherited/local typed option
    -> Init(Context{Resource:nil})
    -> Dial/Register 构造规范 service Resource
    -> Filters 返回资源级 FilterSpec
    -> 核心合并、排序并去重最终 chain
    -> Invoke 阶段按 MethodName 选择 method 配置或状态
    -> 上下文关闭时释放 Extension Runtime
```

## 17. 代码集成位置

预计涉及以下模块：

- `common/extension`：Scope、Context、Resource、RawConfig、Option、Definition、
  FilterSpec、注册表和 Runtime；
- 根包 `options.go`、`dubbo.go`、`loader.go`：Instance Option、Plan 派生及原始
  配置加载；
- `client/options.go`、`client/client.go`、`client/action.go`：Client Runtime、
  Dial 资源绑定及 consumer FilterSpec；
- `server/options.go`、`server/server.go`、`server/action.go`：Server Runtime、
  Register 资源绑定及 provider FilterSpec；
- `protocol/protocolwrapper`：接收资源级 FilterSpec 并构建最终 Filter chain。

字符串形式的 `WithFilter`、`WithClientFilter`、`WithServerFilter` 以及全局
named filter 注册机制不属于新的外部扩展协议，随着 breaking change 一并移除。

## 18. 实施阶段

### 第一阶段：协议和 Runtime

- 实现 Scope/Role 校验；
- 实现 Definition 注册和快照；
- 实现通用 Option 分组和应用；
- 实现 parser-independent RawConfig；
- 实现 ExtensionPlan 以及 Instance/Client/Server 独立 Runtime；
- 实现激活、优先级、Init、失败回滚和 Close。

### 第二阶段：Resource 和 Filter

- 在 Client Dial、Server Register 路径构造规范 Resource；
- 实现资源 Context 副本和缓存；
- 实现 FilterSpec Factory chain；
- 实现稳定排序、ID 冲突检测和去重规则；
- 移除 named filter registry 及字符串 filter option。

### 第三阶段：外部扩展验证

在 dubbo-go 核心之外实现 Hystrix，并验证：

- consumer/provider 配置；
- 包含 `.`、`:` 的完整 resource key；
- 默认值、YAML、typed option 优先级；
- service/method 级配置；
- 同一个 Client 下多个服务的状态隔离；
- 多 Client、多 Server 的配置隔离；
- Instance 派生和直接创建；
- 自动 Filter 贡献；
- Close 和失败回滚；
- 并发绑定和调用无数据竞争。

## 19. 验收标准

满足以下条件时，可以认为通用扩展协议完成：

- dubbo-go 核心不存在 Hystrix 专用 import、名称判断、类型断言或配置字段；
- 外部扩展可以通过 YAML 或 typed option 激活；
- 非法 Scope/Role 和不支持的 Scope 能够提前失败；
- 扩展 resource key 在加载过程中保持原样；
- 两个使用不同扩展配置的 Client 不会相互影响；
- 同一个 Client 下两个服务可以拥有独立运行时状态；
- 直接创建 Client/Server 与 Instance 子上下文行为一致；
- 激活扩展后，用户不需要手工添加其 Filter；
- 扩展 Filter 由 Definition 自动贡献，用户不需要也不能手工指定 filter name；
- 初始化失败能够确定性回滚；
- 扩展生命周期和资源绑定通过 race detector。
