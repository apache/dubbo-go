# [Dubbo-go-hessian2 v1.7.0 发布](https://www.oschina.net/news/118648/dubbogo-hessian2-1-7-0-released)

Dubbo-go-hessian2 v1.7.0已发布，详见 [https://github.com/apache/dubbo-go-hessian2/releases/tag/v1.7.0，](https://github.com/apache/dubbo-go-hessian2/releases/tag/v1.7.0%EF%BC%8C) 以下对这次更新内容进行详细整理。

另外v1.6.3 将 attachment 类型由 map\[string\]stiring 改为map\[string\]interface{} 导致版本不兼容问题，这部分已还原，后续的计划是将dubbo协议的request/response对象整体迁移到dubbogo项目中进行迭代修改， hessian2中将不再改动到request/response对象。

## 1\. New Features

### 1.1 add GetStackTrace method into Throwabler and its implements. [#207](https://github.com/apache/dubbo-go-hessian2/pull/207)

> contributed by [https://github.com/cvictory](https://github.com/cvictory)

go语言client请求java语言服务时，如果java语言抛出了异常，异常对应的堆栈信息是被保存在StackTraceElement中。

这个异常信息在日志中最好能被打印出来，以方便客户端排查问题，所以在Throwabler和对应子类中增加了StackTraceElement的获取。

注：其实还有一种更好的方法，所有的具体的异常类型都包含java\_exception/exception.go的Throwable struct。这样只需要在Throwable中增加GetStackTrace方法就可以了。但是这种方式需要更多的测试验证，改动的逻辑相对会复杂一些。但是代码会更整洁。 这里先不用这种方法。

### 1.2 catch user defined exceptions. [#208](https://github.com/apache/dubbo-go-hessian2/pull/208)

> contributed by [https://github.com/cvictory](https://github.com/cvictory)

golang中增加一个java中Exception对象的序列化输出方法：

```css
func JavaException() []byte {
	e := hessian.NewEncoder()
	exception := java_exception.NewException("java_exception")
	e.Encode(exception)
	return e.Buffer()
}
```

在output/output.go 提供调用入口:添加如下函数初始化声明

```plain
func init() {
    funcMap["JavaException"] = testfuncs.JavaException
}
```

java代码中增加调用go方法序列化结果: **说明**: Assert.assertEquals 不能直接比较Exception对象是否相等

```php
    /**
     * test java java.lang.Exception object and go java_exception Exception struct
     */
    @Test
    public void testException() {
        Exception exception = new Exception("java_exception");
        Object javaException = GoTestUtil.readGoObject("JavaException");
        if (javaException instanceof Exception) {
            Assert.assertEquals(exception.getMessage(), ((Exception) javaException).getMessage());
        }
    }
```

### 1.3 support java8 time object. [#212](https://github.com/apache/dubbo-go-hessian2/pull/212), [#221](https://github.com/apache/dubbo-go-hessian2/pull/221)

> contributed by [https://github.com/willson-chen](https://github.com/willson-chen), [https://github.com/cyb-code](https://github.com/cyb-code)

golang中增加一个java8对象的序列化输出方法：

```go
// test java8 java.time.Year
func Java8TimeYear() []byte {
    e := hessian.NewEncoder()
    year := java8_time.Year{Year: 2020}
    e.Encode(year)
    return e.Buffer()
}

// test java8 java.time.LocalDate
func Java8LocalDate() []byte {
    e := hessian.NewEncoder()
    date := java8_time.LocalDate{Year: 2020, Month: 9, Day: 12}
    e.Encode(date)
    return e.Buffer()
}
```

在output/output.go 提供调用入口:添加函数初始化声明

```plain
func init() {
	funcMap["Java8TimeYear"] = testfuncs.Java8TimeYear
	funcMap["Java8LocalDate"] = testfuncs.Java8LocalDate
}
```

java代码中增加调用go方法序列化结果:

```java
/**
 * test java8 java.time.* object and go java8_time/* struct
 */
@Test
public void testJava8Year() {
    Year year = Year.of(2020);
    Assert.assertEquals(year
            , GoTestUtil.readGoObject("Java8TimeYear"));
    LocalDate localDate = LocalDate.of(2020, 9, 12);
    Assert.assertEquals(localDate, GoTestUtil.readGoObject("Java8LocalDate"));
}
```

### 1.4 support test golang encoding data in java. [#213](https://github.com/apache/dubbo-go-hessian2/pull/213)

> contributed by [https://github.com/wongoo](https://github.com/wongoo)

为了更好的测试验证hessian库，原来已经支持在golang中测试java的序列化数据，现在增加在java中测试golang的序列化数据，实现双向测试验证。

golang中增加序列化输出方法:

```css
func HelloWorldString() []byte {
    e := hessian.NewEncoder()
    e.Encode("hello world")
    return e.Buffer()
}
```

将该方法注册到output/output.go中

```plain
 // add all output func here
 func init() {
     funcMap["HelloWorldString"] = testfuncs.HelloWorldString
}
```

output/output.go 提供调用入口:

```go
func main() {
    flag.Parse()

    if *funcName == "" {
        _, _ = fmt.Fprintln(os.Stderr, "func name required")
        os.Exit(1)
    }
    f, exist := funcMap[*funcName]
    if !exist {
        _, _ = fmt.Fprintln(os.Stderr, "func name not exist: ", *funcName)
        os.Exit(1)
    }

    defer func() {
        if err := recover(); err != nil {
            _, _ = fmt.Fprintln(os.Stderr, "error: ", err)
            os.Exit(1)
        }
    }()
    if _, err := os.Stdout.Write(f()); err != nil {
        _, _ = fmt.Fprintln(os.Stderr, "call error: ", err)
        os.Exit(1)
    }
    os.Exit(0)
}
```

java代码中增加调用go方法序列化结果:

```plain
public class GoTestUtil {

    public static Object readGoObject(String func) {
        System.out.println("read go data: " + func);
        try {
            Process process = Runtime.getRuntime()
                    .exec("go run output/output.go -func_name=" + func,
                            null,
                            new File(".."));

            int exitValue = process.waitFor();
            if (exitValue != 0) {
                Assert.fail(readString(process.getErrorStream()));
                return null;
            }

            InputStream is = process.getInputStream();
            Hessian2Input input = new Hessian2Input(is);
            return input.readObject();
        } catch (Exception e) {
            e.printStackTrace();
            return null;
        }
    }

    private static String readString(InputStream in) throws IOException {
        StringBuilder out = new StringBuilder();
        InputStreamReader reader = new InputStreamReader(in, StandardCharsets.UTF_8);
        char[] buffer = new char[4096];

        int bytesRead;
        while ((bytesRead = reader.read(buffer)) != -1) {
            out.append(buffer, 0, bytesRead);
        }

        return out.toString();
    }
}
```

增加java测试代码:

```java
@Test
public void testHelloWordString() {
    Assert.assertEquals("hello world"
            , GoTestUtil.readGoObject("HelloWorldString"));
}
```

### 1.5 support java.sql.Time & java.sql.Date. [#219](https://github.com/apache/dubbo-go-hessian2/pull/219)

> contributed by [https://github.com/zhangshen023](https://github.com/zhangshen023)

增加了 java 类 java.sql.Time, java.sql.Date 支持，分别对应到hessian.Time 和 hessian.Date， 详见 [https://github.com/apache/dubbo-go-hessian2/pull/219/files。](https://github.com/apache/dubbo-go-hessian2/pull/219/files%E3%80%82)

## 2\. Enhancement

### 2.1 Export function EncNull. [#225](https://github.com/apache/dubbo-go-hessian2/pull/225)

> contributed by [https://github.com/cvictory](https://github.com/cvictory)

开放 hessian.EncNull 方法，以便用户特定情况下使用。

## 3\. Bugfixes

### 3.1 fix enum encode error in request. [#203](https://github.com/apache/dubbo-go-hessian2/pull/203)

> contributed by [https://github.com/pantianying](https://github.com/pantianying)

原来在 dubbo request 对象中没有判断 enum 类型的情况，此pr增加了判断是不是POJOEnum类型。详见 [https://github.com/apache/dubbo-go-hessian2/pull/203/files](https://github.com/apache/dubbo-go-hessian2/pull/203/files)

### 3.2 fix \[\]byte field decoding issue. [#216](https://github.com/apache/dubbo-go-hessian2/pull/216)

> contributed by [https://github.com/wongoo](https://github.com/wongoo)

v1.7.0 之前如果 struct中包含\[\]byte字段时无法反序列化, 报错“error list tag: 0x29”，主要原因是被当做list进行处理，对于这种情况应该按照binary数据进行处理即可。

```go
type Circular struct {
    Num      int
	Previous *Circular
	Next     *Circular
	ResponseDataBytes    []byte // <---- 
}

func (Circular) JavaClassName() string {
	return "com.company.Circular"
}
```

### 3.3 fix decoding error for map in map. [#229](https://github.com/apache/dubbo-go-hessian2/pull/229)

> contributed by [https://github.com/wongoo](https://github.com/wongoo)

v1.7.0 之前嵌套map无法正确解析，主要原因是对应的map对象被当做一个数据类型却未被自动加到类引用列表中，而嵌套map类信息是同一类型的引用，去类引用列表找，找不到就报错了。 解决这个问题的方法就是遇到map类对象，也将其加入到类引用列表中即可。 问题详细参考 [#119](https://github.com/apache/dubbo-go-hessian2/issues/119).

### 3.4 fix fields name mismatch in Duration class. [#234](https://github.com/apache/dubbo-go-hessian2/pull/234)

> contributed by [https://github.com/skyao](https://github.com/skyao)

这个 PR 解决了Duration对象中字段错误定义，原来是"second/nano"， 应该是"seconds/nanos"。

同时改善了测试验证数据。之前使用0作为int字段的测试数据，这是不准确的，因为int类型默认值就是0.