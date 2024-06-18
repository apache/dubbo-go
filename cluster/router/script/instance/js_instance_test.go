/*
 * Licensed to the Apache Software Foundation (ASF) under one or more
 * contributor license agreements.  See the NOTICE file distributed with
 * this work for additional information regarding copyright ownership.
 * The ASF licenses this file to You under the Apache License, Version 2.0
 * (the "License"); you may not use this file except in compliance with
 * the License.  You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package instance

import (
	"context"
	"fmt"
	"reflect"
	"sync"
	"testing"
)

import (
	"github.com/dop251/goja"
	_ "github.com/dop251/goja_nodejs/console"
	"github.com/dop251/goja_nodejs/require"

	"github.com/stretchr/testify/assert"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/protocol"
	"dubbo.apache.org/dubbo-go/v3/protocol/invocation"
)

var url1 = func() *common.URL {
	i, _ := common.NewURL("dubbo://127.0.0.1:20000/com.ikurento.user.UserProvider?anyhost=true&" +
		"application=BDTService&category=providers&default.timeout=10000&dubbo=dubbo-provider-golang-1.0.0&" +
		"environment=dev&interface=com.ikurento.user.UserProvider&ip=192.168.56.1&methods=GetUser%2C&" +
		"module=dubbogo+user-info+server&org=ikurento.com&owner=ZX&pid=1447&revision=0.0.1&" +
		"side=provider&timeout=3000&timestamp=1556509797245")
	return i
}
var url2 = func() *common.URL {
	u, _ := common.NewURL("dubbo://127.0.0.1:20001/com.ikurento.user.UserProvider?anyhost=true&" +
		"application=BDTService&category=providers&default.timeout=10000&dubbo=dubbo-provider-golang-1.0.0&" +
		"environment=dev&interface=com.ikurento.user.UserProvider&ip=192.168.56.1&methods=GetUser%2C&" +
		"module=dubbogo+user-info+server&org=ikurento.com&owner=ZX&pid=1448&revision=0.0.1&" +
		"side=provider&timeout=3000&timestamp=1556509797246")
	return u
}
var url3 = func() *common.URL {
	i, _ := common.NewURL("dubbo://127.0.0.1:20002/com.ikurento.user.UserProvider?anyhost=true&" +
		"application=BDTService&category=providers&default.timeout=10000&dubbo=dubbo-provider-golang-1.0.0&" +
		"environment=dev&interface=com.ikurento.user.UserProvider&ip=192.168.56.1&methods=GetUser%2C&" +
		"module=dubbogo+user-info+server&org=ikurento.com&owner=ZX&pid=1449&revision=0.0.1&" +
		"side=provider&timeout=3000&timestamp=1556509797247")
	return i
}

func getRouteArgs() ([]protocol.Invoker, protocol.Invocation, context.Context) {
	return []protocol.Invoker{
			protocol.NewBaseInvoker(url1()), protocol.NewBaseInvoker(url2()), protocol.NewBaseInvoker(url3()),
		}, invocation.NewRPCInvocation("GetUser", nil, map[string]interface{}{
			"attachmentKey": []string{"attachmentValue"},
		}),
		context.TODO()
}

var localIp = common.GetLocalIp()

var Func_Script string = `
(function route(invokers,invocation,context) {
	var result = [];
	for (var i = 0; i < invokers.length; i++) {
	    if ("127.0.0.1" === invokers[i].GetURL().Ip) {
			if (invokers[i].GetURL().Port !== "20000"){
				invokers[i].GetURL().Ip = "` + localIp + `"
				result.push(invokers[i]);
			}
	    }
	}
	return result;
}(invokers,invocation,context));`

func rt_link_external_libraries(runtime *goja.Runtime) {
	require.NewRegistry().Enable(runtime)
}

func rt_init_args(runtime *goja.Runtime) {
	invokers, invocation, context := getRouteArgs()
	// 在 Goja 中注册 Go 的 struct
	err := runtime.Set("invokers", invokers)
	if err != nil {
		panic(err)
	}
	err = runtime.Set("invocation", invocation)
	if err != nil {
		panic(err)
	}
	err = runtime.Set("context", context)
	if err != nil {
		panic(err)
	}
}

func re_init_res_recv(runtime *goja.Runtime) {
	err := runtime.Set(jsScriptResultName, nil)
	if err != nil {
		panic(err)
	}
}

func TestStructureFuncImpl(t *testing.T) {
	runtime := goja.New()

	test_invokers := []protocol.Invoker{protocol.NewBaseInvoker(url1())}
	test_invocation := invocation.NewRPCInvocation("GetUser", nil, map[string]interface{}{"attachmentKey": []string{"attachmentValue"}})
	test_context := context.TODO() // set invoker test field

	for _, invoker := range test_invokers {
		invoker.GetURL().Methods = []string{"testMethods"}
		invoker.GetURL().Username = "testUsername"
		invoker.GetURL().Password = "testPassword"
		invoker.GetURL().SetParam(constant.VersionKey, "testVersion")
		invoker.GetURL().SetParam(constant.GroupKey, "testGroup")
		invoker.GetURL().SetParam(constant.InterfaceKey, "testInterface")
		invoker.GetURL().SubURL = url3()
	}

	rt_link_external_libraries(runtime)

	err := runtime.Set("invokers", test_invokers)
	if err != nil {
		panic(err)
	}
	err = runtime.Set("invocation", test_invocation)
	if err != nil {
		panic(err)
	}
	err = runtime.Set("context", test_context)
	if err != nil {
		panic(err)
	}

	re_init_res_recv(runtime)

	// run go func in javaScript
	_ = runtime.Set(`check`, func(str string, args ...interface{}) {
		/*
			here set fmt.print to check func support
		*/
		//fmt.Printf("support %s\n", str)
		//fmt.Print(args...)
		//fmt.Print("\n")
	})

	/*
		//may not require

		invokers[0].GetURL().URLEqual(url *URL) bool
		invokers[0].GetURL().ReplaceParams(param url.Values)
		invokers[0].GetURL().RangeParams(f func(key string, value string) bool)
		invokers[0].GetURL().GetParams() url.Values
		invokers[0].GetURL().SetParams(m url.Values)
		invokers[0].GetURL().MergeURL(anotherUrl *URL) *URL
		invokers[0].GetURL().Clone() *URL
		invokers[0].GetURL().CloneExceptParams(excludeParams *gxset.HashSet) *URL
		invokers[0].GetURL().Compare(comp common.Comparator) int

		invocation.SetInvoker(invoker protocol.Invoker)
		invocation.CallBack()
		invocation.SetCallBack(c interface{})

	*/
	// 在 JavaScript 中调用 Go 的 struct 的方法
	_, err = runtime.RunString(`var console = require('console')
function route(invokers, invocation, context) {
    var result = [];
    for (var i = 0; i < invokers.length; i++) {
        // get all field 
        check(
            'invokers[i].GetURL().Path',
            invokers[i].GetURL().Path
        )
        check(
            'invokers[i].GetURL().Address()',
            invokers[i].GetURL().Address()
        )
        check(
            'invokers[i].GetURL().Protocol',
            invokers[i].GetURL().Protocol
        )
        check(
            'invokers[i].GetURL().Location',
            invokers[i].GetURL().Location
        )
        check(
            'invokers[i].GetURL().Ip',
            invokers[i].GetURL().Ip
        )
        check(
            'invokers[i].GetURL().Port',
            invokers[i].GetURL().Port
        )
        check(
            'invokers[i].GetURL().PrimitiveURL',
            invokers[i].GetURL().PrimitiveURL
        )
        check(
            'invokers[i].GetURL().Username',
            invokers[i].GetURL().Username
        )
        check(
            'invokers[i].GetURL().Password',
            invokers[i].GetURL().Password
        )
        check(
            'invokers[i].GetURL().Methods',
            invokers[i].GetURL().Methods
        )
        check(
            'invokers[i].GetURL().SubURL',
            invokers[i].GetURL().SubURL
        )
        // do all call 
        check(
            'invokers[i].GetURL()',
            invokers[i].GetURL()
        )
        check(
            'invokers[i].GetURL().JavaClassName()',
            invokers[i].GetURL().JavaClassName()
        )
        check(
            'invokers[i].GetURL().Group()',
            invokers[i].GetURL().Group()
        )
        check(
            'invokers[i].GetURL().Interface()',
            invokers[i].GetURL().Interface()
        )
        check(
            'invokers[i].GetURL().Version()',
            invokers[i].GetURL().Version()
        )
        check(
            'invokers[i].GetURL().Address()',
            invokers[i].GetURL().Address()
        )
        check(
            'invokers[i].GetURL().String()',
            invokers[i].GetURL().String()
        )
        check(
            'invokers[i].GetURL().Key()',
            invokers[i].GetURL().Key()
        )
        check(
            'invokers[i].GetURL().GetCacheInvokerMapKey()',
            invokers[i].GetURL().GetCacheInvokerMapKey()
        )
        check(
            'invokers[i].GetURL().ServiceKey()',
            invokers[i].GetURL().ServiceKey()
        )
        check(
            'invokers[i].GetURL().ColonSeparatedKey()',
            invokers[i].GetURL().ColonSeparatedKey()
        )
        check(
            'invokers[i].GetURL().EncodedServiceKey()',
            invokers[i].GetURL().EncodedServiceKey()
        )
        check(
            'invokers[i].GetURL().Service()',
            invokers[i].GetURL().Service()
        )
        check(
            'invokers[i].GetURL().AddParam("key-string", "value-string")',
            invokers[i].GetURL().AddParam("key-string", "value-string")
        )
        check(
            'invokers[i].GetURL().AddParamAvoidNil("key-string", "value-string")',
            invokers[i].GetURL().AddParamAvoidNil("key-string", "value-string")
        )
        check(
            'invokers[i].GetURL().SetParam("key-string", "value-string")',
            invokers[i].GetURL().SetParam("key-string", "value-string")
        )
        check(
            'invokers[i].GetURL().SetAttribute("key-string", "value-interface{}")',
            invokers[i].GetURL().SetAttribute("key-string", "value-interface{}")
        )
        check(
            'invokers[i].GetURL().GetAttribute("key-string")',
            invokers[i].GetURL().GetAttribute("key-string")
        )
        check(
            'invokers[i].GetURL().DelParam("key-string")',
            invokers[i].GetURL().DelParam("key-string")
        )
        check(
            'invokers[i].GetURL().GetParam("key-string", "default-value")',
            invokers[i].GetURL().GetParam("key-string", "default-value")
        )
        check(
            'invokers[i].GetURL().GetNonDefaultParam("key-string")',
            invokers[i].GetURL().GetNonDefaultParam("key-string")
        )
        check(
            'invokers[i].GetURL().GetParamAndDecoded("key-string")',
            invokers[i].GetURL().GetParamAndDecoded("key-string")
        )
        check(
            'invokers[i].GetURL().GetRawParam("key-string")',
            invokers[i].GetURL().GetRawParam("key-string")
        )
        check(
            'invokers[i].GetURL().GetParamBool("key-string", true)',
            invokers[i].GetURL().GetParamBool("key-string", true)
        )
        check(
            'invokers[i].GetURL().GetParamInt("key-string", 64)',
            invokers[i].GetURL().GetParamInt("key-string", 64)
        )
        check(
            'invokers[i].GetURL().GetParamInt32("key-string", 64)',
            invokers[i].GetURL().GetParamInt32("key-string", 64)
        )
        check(
            'invokers[i].GetURL().GetParamByIntValue("key-string",64 )',
            invokers[i].GetURL().GetParamByIntValue("key-string", 64)
        )
        check(
            'invokers[i].GetURL().GetMethodParamInt("GetUser", "key-string", 64)',
            invokers[i].GetURL().GetMethodParamInt("GetUser", "key-string", 64)
        )
        check(
            'invokers[i].GetURL().GetMethodParamIntValue("GetUser", "key-string", 65)',
            invokers[i].GetURL().GetMethodParamIntValue("GetUser", "key-string", 65)
        )
        check(
            'invokers[i].GetURL().GetMethodParamInt64("GetUser", " key-string", 64)',
            invokers[i].GetURL().GetMethodParamInt64("GetUser", " key-string", 64)
        )
        check(
            'invokers[i].GetURL().GetMethodParam("GetUser", "key-string", "d string")',
            invokers[i].GetURL().GetMethodParam("GetUser", "key-string", "d string")
        )
        check(
            'invokers[i].GetURL().GetMethodParamBool("GetUser", "key-string", false)',
            invokers[i].GetURL().GetMethodParamBool("GetUser", "key-string", false)
        )
        check(
            'invokers[i].GetURL().ToMap()',
            invokers[i].GetURL().ToMap()
        )
        check(
            'invokers[i].GetURL().CloneWithParams(["a", "b", "c", "d"])',
            invokers[i].GetURL().CloneWithParams(["a", "b", "c", "d"])
        )
        check(
            'invokers[i].GetURL().GetParamDuration("s string", "d string")',
            invokers[i].GetURL().GetParamDuration("s string", "d string")
        )
		// test invocation
        check(
            'invocation.MethodName()',
            invocation.MethodName()
        )
        check(
            'invocation.ActualMethodName()',
            invocation.ActualMethodName()
        )
        check(
            'invocation.IsGenericInvocation()',
            invocation.IsGenericInvocation()
        )
        check(
            'invocation.ParameterTypes()',
            invocation.ParameterTypes()
        )
        check(
            'invocation.ParameterTypeNames()',
            invocation.ParameterTypeNames()
        )
        check(
            'invocation.ParameterValues()',
            invocation.ParameterValues()
        )
        check(
            'invocation.ParameterRawValues()',
            invocation.ParameterRawValues()
        )
        check(
            'invocation.Arguments()',
            invocation.Arguments()
        )
        check(
            'invocation.Reply()',
            invocation.Reply()
        )
        check(
            'invocation.SetReply("interface key")',
            invocation.SetReply("interface key")
        )
        check(
            'invocation.Attachments()',
            invocation.Attachments()
        )
        check(
            'invocation.GetAttachmentInterface("key-string")',
            invocation.GetAttachmentInterface("key-string")
        )
        check(
            'invocation.Attributes()',
            invocation.Attributes()
        )
        check(
            'invocation.Invoker()',
            invocation.Invoker()
        )
        check(
            'invocation.ServiceKey()',
            invocation.ServiceKey()
        )
        check(
            'invocation.SetAttachment("key-string", "value-interface{}")',
            invocation.SetAttachment("key-string", "value-interface{}")
        )
        check(
            'invocation.GetAttachment("key-string")',
            invocation.GetAttachment("key-string")
        )
        check(
            'invocation.GetAttachmentWithDefaultValue("key-string", "defaultValue-string")',
            invocation.GetAttachmentWithDefaultValue("key-string", "defaultValue-string")
        )
        check(
            'invocation.SetAttribute("key-string", "value-interface{}")',
            invocation.SetAttribute("key-string", "value-interface{}")
        )
        check(
            'invocation.GetAttribute("key-string")',
            invocation.GetAttribute("key-string")
        )
        check(
            'invocation.GetAttributeWithDefaultValue("key-string", "defaultValue-interface{}")',
            invocation.GetAttributeWithDefaultValue("key-string", "defaultValue-interface{}")
        )
        check(
            'invocation.GetAttachmentAsContext()',
            invocation.GetAttachmentAsContext()
        )
    }
    return result;
}

__go_program_get_result = route(invokers,
    invocation, context)
`)
	assert.Nil(t, err)
}

func TestFuncWithCompile(t *testing.T) {
	pg, err := goja.Compile("routeJs", jsScriptPrefix+Func_Script, true)
	if err != nil {
		panic(err)
	}
	rt := goja.New()
	rt_link_external_libraries(rt)
	rt_init_args(rt)
	re_init_res_recv(rt)
	res, err := rt.RunProgram(pg)
	if err != nil {
		panic(err)
	}
	assert.Equal(t, 2, len(res.Export().([]interface{})))
	assert.Equal(t, localIp, (*(res.Export().([]interface{})[0]).(*protocol.BaseInvoker)).GetURL().Ip)
}

func TestFuncWithCompileConcurrent(t *testing.T) {
	pg, err := goja.Compile("routeJs", jsScriptPrefix+Func_Script, true)
	if err != nil {
		panic(err)
	}
	wg := sync.WaitGroup{}
	wg.Add(100)
	for i := 0; i < 100; i++ {
		go func() {
			defer wg.Done()
			rt := goja.New()
			rt_link_external_libraries(rt)
			rt_init_args(rt)
			re_init_res_recv(rt)
			res, err := rt.RunProgram(pg)
			if err != nil {
				panic(err)
			}
			assert.Equal(t, 2, len(res.Export().([]interface{})))
			assert.Equal(t, localIp, (*(res.Export().([]interface{})[0]).(*protocol.BaseInvoker)).GetURL().Ip)
		}()
	}
	wg.Wait()
}

func TestFuncWithCompileAndRunRepeatedly(t *testing.T) {
	pg, err := goja.Compile("routeJs", jsScriptPrefix+`(
function route(invokers,invocation,context) {
	var result = [];
	for (var i = 0; i < invokers.length; i++) {
		if ("127.0.0.1" === invokers[i].GetURL().Ip) {
			if (invokers[i].GetURL().Port !== "20000"){
				invokers[i].GetURL().Ip = "`+localIp+`"
				invokers[i].GetURL().Port = "20004"
				result.push(invokers[i]);
			}
	    }
	}
	return result;
})(invokers,invocation,context)`, true)
	if err != nil {
		panic(err)
	}
	rt := goja.New()
	_ = rt.Set(`println`, func(args ...interface{}) {
		//fmt.Println(args...)
	})
	for i := 0; i < 100; i++ {
		rt_link_external_libraries(rt)
		rt_init_args(rt)
		re_init_res_recv(rt)
		res, err := rt.RunProgram(pg)
		if err != nil {
			panic(err)
		}
		assert.Equal(t, 2, len(res.Export().([]interface{})))
		assert.Equal(t, localIp, (*(res.Export().([]interface{})[0]).(*protocol.BaseInvoker)).GetURL().Ip)
		assert.Equal(t, "20004", (*(res.Export().([]interface{})[0]).(*protocol.BaseInvoker)).GetURL().Port)
	}
}

func setRunScriptEnv() *goja.Runtime {
	runtime := goja.New()
	rt_link_external_libraries(runtime)
	srcInvoker1, srcInvoker2, srcInvoker3 := protocol.NewBaseInvoker(url1()), protocol.NewBaseInvoker(url2()), protocol.NewBaseInvoker(url3())

	getArgs := func() ([]protocol.Invoker, *invocation.RPCInvocation, context.Context) {
		return []protocol.Invoker{
				newScriptInvokerImpl(srcInvoker1), newScriptInvokerImpl(srcInvoker2), newScriptInvokerImpl(srcInvoker3),
			}, invocation.NewRPCInvocation("GetUser", nil, map[string]interface{}{
				"attachmentKey": []string{"attachmentValue"},
			}), context.TODO()
	}

	invokers, invocation, context := getArgs()
	err := runtime.Set("invokers", invokers)
	if err != nil {
		panic(err)
	}
	err = runtime.Set("invocation", invocation)
	if err != nil {
		panic(err)
	}
	err = runtime.Set("context", context)
	if err != nil {
		panic(err)
	}
	re_init_res_recv(runtime)
	return runtime
}

func TestRunScriptInPanic(t *testing.T) {
	willPanic := func(errScript string) {
		rt := setRunScriptEnv()
		pg, err := goja.Compile(``, errScript, true)
		assert.Nil(t, err)
		res, err := func() (res interface{}, err error) {
			defer func(err *error) {
				panicReason := recover()
				if panicReason != nil {
					switch panicReason.(type) {
					case string:
						*err = fmt.Errorf("panic: %s", panicReason.(string))
					case error:
						*err = panicReason.(error)
					default:
						*err = fmt.Errorf("panic occurred: failed to retrieve panic information. Expected string or error, got %T", panicReason)
					}
				}
			}(&err)
			res, err = rt.RunProgram(pg)
			testData := res.(*goja.Object).Export()
			assert.Equal(t, reflect.ValueOf([]interface{}{}), reflect.ValueOf(testData))
			return
		}()
		assert.NotNil(t, err)
		assert.Nil(t, res)
	}
	wontPanic := func(errScript string) {
		rt := setRunScriptEnv()
		pg, err := goja.Compile(``, errScript, true)
		assert.Nil(t, err)
		res, err := rt.RunProgram(pg)
		assert.Nil(t, err)
		assert.NotNil(t, res)
		data := res.Export()
		assert.NotNil(t, data)
	}
	const scriptInvokePanic = `(function route(invokers, invocation, context) {
    var result = [];
    for (var i = 0; i < invokers.length; i++) {
        if ("127.0.0.1" === invokers[i].GetURL().Ip) {
            if (invokers[i].GetURL().Port !== "20000") {
                invokers[i].GetURL().Ip = "anotherIP"
                // Invoke() call should go panic 
                invokers[i].Invoke(context, invocation)
                result.push(invokers[i]);
            }
        }
    }
    return result;
}(invokers, invocation, context));
`
	willPanic(scriptInvokePanic)
	const scriptNodeDestroyPanic = `(function route(invokers, invocation, context) {
    var result = [];
    for (var i = 0; i < invokers.length; i++) {
        if ("127.0.0.1" === invokers[i].GetURL().Ip) {
            if (invokers[i].GetURL().Port !== "20000") {
                invokers[i].GetURL().Ip = "anotherIP"
                // Destroy() call should go panic 
                invokers[i].Destroy()
                result.push(invokers[i]);
            }
        }
    }
    return result;
}(invokers, invocation, context));
`
	willPanic(scriptNodeDestroyPanic)

	// Wrong Call will go panic , recover , and return an error
	const scriptCallWrongFunc = `(function route(invokers, invocation, context) {
    var result = [];
    for (var i = 0; i < invokers.length; i++) {
        if ("127.0.0.1" === invokers[i].GetURL().Ip) {
						// err here 
            if (invokers[i].GetURLS(" Err Here ").Port !== "20000") {
                invokers[i].GetURLS().Ip = "anotherIP"
                result.push(invokers[i]);
            }
        }
    }
    return result;
}(invokers, invocation, context));
`
	willPanic(scriptCallWrongFunc)

	// these case means
	// error input is program acceptable, but it will make an undefined error
	const scriptCallWrongArgs1 = `(function route(invokers, invocation, context) {
    var result = [];
    for (var i = 0; i < invokers.length; i++) {
        if ("127.0.0.1" === invokers[i].GetURL().Ip) {
            if (invokers[i].GetURL(" Err Here ").Port !== "20000") {
//---------------------------------^  here wont go err , it will trans to ()
                invokers[i].GetURL().Ip = "anotherIP"
                result.push(invokers[i]);
            }
        }
    }
    return result;
}(invokers, invocation, context));
`
	wontPanic(scriptCallWrongArgs1)

	const scriptCallWrongArgs2 = `(function route(invokers, invocation, context) {
    var result = [];
    for (var i = 0; i < invokers.length; i++) {
        if ("127.0.0.1" === invokers[i].GetURL().Ip) {
            if (invokers[i].GetURL(" Err Here ").Port !== "20000") {
                invokers[i].GetURL().AddParam( null , "key string", "value string")
//---------------------------------------------^  here wont go err , it will trans to ("" , "key string" , "value string")
                result.push(invokers[i]);
            }
        }
    }
    return result;
}(invokers, invocation, context));
`
	wontPanic(scriptCallWrongArgs2)

	const scriptCallWrongArgs3 = `(function route(invokers, invocation, context) {
    var result = [];
    for (var i = 0; i < invokers.length; i++) {
        if ("127.0.0.1" === invokers[i].GetURL().Ip) {
            if (invokers[i].GetURL(" Err Here ").Port !== "20000") {
                invokers[i].GetURL().AddParam( invocation , "key string", "value string")
//---------------------------------------------^  here wont go err , it will trans to ("[object Object]" , "key string" , "value string")
                result.push(invokers[i]);
            }
        }
    }
    return result;
}(invokers, invocation, context));
`
	wontPanic(scriptCallWrongArgs3)
}
