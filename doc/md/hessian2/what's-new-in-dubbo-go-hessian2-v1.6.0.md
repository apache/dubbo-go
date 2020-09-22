# [What's new in Dubbo-go-hessian2 v1.6.0](https://my.oschina.net/dubbogo/blog/4318016)

发版人：[望哥](https://github.com/wongoo)

## 1\. 增加缓存优化

dubbo-go-hessian2 在解析数据的数据大量使用到了 struct 的结构信息，这部分信息可以缓存起来反复利用，使得性能提升了一倍。优化过程记录可以详细阅读[《记一次对 dubbo-go-hessian2 的性能优化》](https://mp.weixin.qq.com/s/ouVxldQAt0_4BET7srjJ6Q).

对应 pr [#179](https://github.com/apache/dubbo-go-hessian2/pull/179)，作者 [micln](https://github.com/micln)。

## 2\. string 解析性能优化

由于 hessian （ dubbo 序列化协议，下称：hessian ）对 string 的定义是16 bit 的 unicode 的 UTF-8 表示形式，字符长度表示是16 bit 的字符数。这是仅针对 java 制定的规范，java 中一个字符是16 bit，对应到 UTF-16. hessian 库也是对每一个字符进行转码序列化。但 golang 里面字符是和 UTF-8 对应的，dubbo-go-hessian2 里面的 rune 是 32bit，和 unicode一一映射。对于 U+10000 ~ U+10FFFF 的字符，需按照 UTF16 的规范，将字符转换为 2 个字节的代理字符，再做转换，才能和 java 的序列化方式对应起来。

原来不管是编码还是解析都是一个字符一个字符处理，特别是解析的时候，从流里面一个字节一个字节读取并组装成 rune，然后再转换为 string，这样效率特别低。我们的优化方案是，批次读取字节流到 buffer 中，对 buffer 进行解析转为 UTF-8 数组，并统计字符数量。其中需要对代理对字符将其转换为标准 UTF-8 子节数组。如果统计的字符数量不足，再进一步读取流种的数据进行解析。通过此方式提升一倍的解析效率。

对应 pr [#188](https://github.com/apache/dubbo-go-hessian2/pull/188)，作者 [zonghaishang](https://github.com/zonghaishang)。

## 3\. 解析忽略不存在的字段

hessian 库在解析数据的时候，对于一个 class 字段，如果不存在，则直接忽略掉。但 v1.6.0 版本之前 dubbo-go-hessian2 解析数据，如果遇到不存在的字段，会返回 error。从 v1.6.0 开始，与 hessian 一样，忽略不存在的字段。**因为这是一个特性的变更，所以升级的同学一定要注意了。**

对应 pr [#201](https://github.com/apache/dubbo-go-hessian2/pull/201)，作者 [micln](https://github.com/micln) & [fangyincheng](https://github.com/fangyincheng)。

## 4\. 解决浮点数精度丢失问题

在对 float32 类型进行序列化时，我们一律强制转换为 float64 再进行序列化操作。由于浮点数的精度问题，在这个转换过程中可能出现小数点后出现多余的尾数，例如 (float32)99.8-->(float64)99.80000305175781。

1.6.0 版本对 float32 的序列化进行了优化：

*   如果小数尾数小于 3 位，根据 hessian2 协议序列化为 double 32-bit 格式
*   否则先转换为 string 类型，再转换为 float64 类型，这样做可以避免由于浮点数精度问题产生多余的尾数，最后对 float64 进行序列化。

虽然对 float32 类型进行了优化，但是依然建议使用浮点数的时候优先使用 float64 类型。

对应 pr [#196](https://github.com/apache/dubbo-go-hessian2/pull/196)，作者 [willson-chen](https://github.com/willson-chen)。

## 5\. 解决 attachment 空值丢失问题

dubbo 请求中包含 attachment 信息，之前如果 attachment 里面含有如 `"key1":""`，这种 value 为空的情况，解析出来的结果会直接丢失这个属性 key1 ，v1.6.0 修复了此问题，现在解析出来的 attachment 会正确解析出空 value 的属性。

对应 pr [#191](https://github.com/apache/dubbo-go-hessian2/pull/191)，作者 [champly](https://github.com/champly)。

## 6\. 支持 ‘继承’ 和忽略冗余字段

由于 go 没有继承的概念，所以在之前的版本，Java 父类的字段不被 dubbo-go-hessian2 所支持。新版本中，dubbo-go-hessian2 将Java来自父类的字段用匿名结构体对应，如：

```rust
type Dog struct {
    Animal
    Gender  string
    DogName string `hessian:"-"`
}
```

同时，就像 json 编码中通过 immediately 可以在序列化中忽略该字段，同理，通过 hessian:"-" 用户也可以让冗余字段不参与 hessian 序列化。

对应pr [#154](https://github.com/apache/dubbo-go-hessian2/pull/154)，作者 [micln](https://github.com/micln)

## 欢迎加入 dubbo-go 社区

钉钉群: **23331795**

[github](https://www.oschina.net/p/github)[apache](https://www.oschina.net/p/apache+http+server)[java](https://www.oschina.net/p/java)