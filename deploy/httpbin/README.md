# 一、测试环境
## 1. install istio

https://istio.io/latest/docs/setup/getting-started/

## 2. 测试环境

![image](./img.png)


## 3. 创建命名空间：dubbo

```shell

$ kubectl create ns dubbo
$ kubectl apply -f dubboclient.yaml -n dubbo
$ kubectl apply -f dubbohttpbin.yaml -n dubbo
$ kubectl apply -f sleep.yaml -n dubbo

```

## 4. 创建命名空间：legacy

```shell

$ kubectl create ns legacy
$ kubectl apply -f dubboclient-legacy.yaml -n legacy
$ kubectl apply -f sleep-legacy.yaml -n legacy

```

## 5. 检查 PeerAuthentication, RequestAuthentication, AuthorizationPolicy 

```shell
kubectl get peerauthentication -n dubbo
kubectl get requestauthentication -n dubbo
kubectl get authorizationpolicy -n dubbo
```


# 二、 测试 mtls & PeerAuthentication

## 1. 定义 sleep pod name
```shell
DUBBO_SLEEP_POD=`kubectl get pod -l app=sleep -n dubbo -o jsonpath={.items..metadata.name}`
LEGACY_SLEEP_POD=`kubectl get pod -l app=sleep -n legacy -o jsonpath={.items..metadata.name}`
```

## 2. 应用 PERMISSIVE mode

```shell
kubectl apply -n dubbo -f - <<EOF
apiVersion: security.istio.io/v1beta1
kind: PeerAuthentication
metadata:
  name: default
spec:
  mtls:
    mode: PERMISSIVE
EOF
```

测试联通性

```shell
$ for from in "dubbo" "legacy"; do kubectl exec "$(kubectl get pod -l app=sleep -n ${from} -o jsonpath={.items..metadata.name})" -c sleep -n ${from} -- curl http://dubboclient.${from}.svc:9090/greet -s -o /dev/null -w "sleep.${from} to dubboclient.${from}: %{http_code}\n"; done

sleep.dubbo to dubboclient.dubbo: 200
sleep.legacy to dubboclient.legacy: 200

```

测试数据返回情况

1. dubbo namespace

```shell
kubectl exec "${DUBBO_SLEEP_POD}" -c sleep -n dubbo -- curl http://dubboclient.dubbo.svc:9090/greet -s -H "x-hello-header: hello" | jq

{
  "request": {
    "args": {},
    "form": {},
    "headers": {
      "accept": "*/*",
      "user-agent": "curl/7.81.0",
      "x-hello-header": "hello"
    },
    "method": "GET",
    "origin": "",
    "url": "/greet",
    "envs": {},
    "host_name": "dubboclient-6f4c544c8c-cnclb",
    "body": ""
  },
  "response": {
    "greeting": "hello world !",
    "attachments": {
      ":authority": "10.10.188.40:8001",
      ":method": "POST",
      ":mtls-mode": "PERMISSIVE",
      ":path": "/greet.GreetService/Greet",
      ":scheme": "https",
      ":source.principal": "spiffe://cluster.local/ns/dubbo/sa/dubboclient",
      ":spiffe": "spiffe://cluster.local/ns/dubbo/sa/dubboclient",
      "INSTANCE_IP": "10.10.188.40",
      "POD_NAME": "httpbin-6b89db567f-p5zsg",
      "accept": "*/*",
      "accept-encoding": "identity",
      "content-type": "application/grpc+proto",
      "grpc-accept-encoding": "gzip",
      "grpc-timeout": "2999991u",
      "interface": "greet.GreetService",
      "retries": "",
      "te": "trailers",
      "timeout": "",
      "user-agent": "curl/7.81.0",
      "x-hello-header": "hello"
    }
  }
}

```

2. legacy namespace

```shell
kubectl exec "${LEGACY_SLEEP_POD}" -c sleep -n legacy -- curl http://dubboclient.legacy.svc:9090/greet -s -H "x-hello-header: hello" | jq

{
  "request": {
    "args": {},
    "form": {},
    "headers": {
      "accept": "*/*",
      "user-agent": "curl/7.81.0",
      "x-hello-header": "hello"
    },
    "method": "GET",
    "origin": "",
    "url": "/greet",
    "envs": {},
    "host_name": "dubboclient-74f8cfcd74-rw5fj",
    "body": ""
  },
  "response": {
    "greeting": "hello world !",
    "attachments": {
      ":method": "POST",
      ":mtls-mode": "PERMISSIVE",
      ":path": "/greet.GreetService/Greet",
      ":scheme": "http",
      ":source.principal": "",
      "INSTANCE_IP": "10.10.188.40",
      "POD_NAME": "httpbin-6b89db567f-p5zsg",
      "accept": "*/*",
      "accept-encoding": "identity",
      "content-type": "application/grpc+proto",
      "grpc-accept-encoding": "gzip",
      "grpc-timeout": "2999954u",
      "interface": "greet.GreetService",
      "retries": "",
      "te": "trailers",
      "timeout": "",
      "user-agent": "curl/7.81.0",
      "x-hello-header": "hello"
    }
  }
}


```

PERMISSIVE MODE 下 dubbo 和 legacy namespace 都可以访问， dubbo namespace 下走 mTLS 认证传输， legacy namespace 下走明文传输。

## 3. 应用 STRICT Mode

```shell
kubectl apply -n dubbo -f - <<EOF
apiVersion: security.istio.io/v1beta1
kind: PeerAuthentication
metadata:
  name: default
spec:
  mtls:
    mode: STRICT
EOF

```

STRICT MODE 测试数据返回情况

1. dubbo namespace

```shell
kubectl exec "${DUBBO_SLEEP_POD}" -c sleep -n dubbo -- curl http://dubboclient.dubbo.svc:9090/greet -s -H "x-hello-header: hello" | jq

{
  "request": {
    "args": {},
    "form": {},
    "headers": {
      "accept": "*/*",
      "user-agent": "curl/7.81.0",
      "x-hello-header": "hello"
    },
    "method": "GET",
    "origin": "",
    "url": "/greet",
    "envs": {},
    "host_name": "dubboclient-6f4c544c8c-cnclb",
    "body": ""
  },
  "response": {
    "greeting": "hello world !",
    "attachments": {
      ":authority": "10.10.188.40:8001",
      ":method": "POST",
      ":mtls-mode": "STRICT",
      ":path": "/greet.GreetService/Greet",
      ":scheme": "https",
      ":source.principal": "spiffe://cluster.local/ns/dubbo/sa/dubboclient",
      ":spiffe": "spiffe://cluster.local/ns/dubbo/sa/dubboclient",
      "INSTANCE_IP": "10.10.188.40",
      "POD_NAME": "httpbin-6b89db567f-p5zsg",
      "accept": "*/*",
      "accept-encoding": "identity",
      "content-type": "application/grpc+proto",
      "grpc-accept-encoding": "gzip",
      "grpc-timeout": "2999985u",
      "interface": "greet.GreetService",
      "retries": "",
      "te": "trailers",
      "timeout": "",
      "user-agent": "curl/7.81.0",
      "x-hello-header": "hello"
    }
  }
}

```

2. legacy namespace

```shell
kubectl exec "${LEGACY_SLEEP_POD}" -c sleep -n legacy -- curl http://dubboclient.legacy.svc:9090/greet -s -H "x-hello-header: hello" | jq

Last error is unknown: http request on mtls STRICT mode is forbidden.: unknown: http request on mtls STRICT mode is forbidden"

```


STRICT MODE 下 dubbo namespace 可以访问,而 legacy namespace不可以访问， dubbo namespace 下走 mTLS 认证传输.

## 4. 应用 DISABLE Mode

```shell
kubectl apply -n dubbo -f - <<EOF
apiVersion: security.istio.io/v1beta1
kind: PeerAuthentication
metadata:
  name: default
spec:
  mtls:
    mode: DISABLE
EOF
```

DISABLE MODE 测试数据返回情况

1. dubbo namespace

```shell
kubectl exec "${DUBBO_SLEEP_POD}" -c sleep -n dubbo -- curl http://dubboclient.dubbo.svc:9090/greet -s -H "x-hello-header: hello" | jq

{
  "request": {
    "args": {},
    "form": {},
    "headers": {
      "accept": "*/*",
      "user-agent": "curl/7.81.0",
      "x-hello-header": "hello"
    },
    "method": "GET",
    "origin": "",
    "url": "/greet",
    "envs": {},
    "host_name": "dubboclient-6f4c544c8c-cnclb",
    "body": ""
  },
  "response": {
    "greeting": "hello world !",
    "attachments": {
      ":method": "POST",
      ":mtls-mode": "DISABLE",
      ":path": "/greet.GreetService/Greet",
      ":scheme": "http",
      ":source.principal": "",
      "INSTANCE_IP": "10.10.188.40",
      "POD_NAME": "httpbin-6b89db567f-p5zsg",
      "accept": "*/*",
      "accept-encoding": "identity",
      "content-type": "application/grpc+proto",
      "grpc-accept-encoding": "gzip",
      "grpc-timeout": "2999991u",
      "interface": "greet.GreetService",
      "retries": "",
      "te": "trailers",
      "timeout": "",
      "user-agent": "curl/7.81.0",
      "x-hello-header": "hello"
    }
  }
}


```

2. legacy namespace

```shell
kubectl exec "${LEGACY_SLEEP_POD}" -c sleep -n legacy -- curl http://dubboclient.legacy.svc:9090/greet -s -H "x-hello-header: hello" | jq

{
  "request": {
    "args": {},
    "form": {},
    "headers": {
      "accept": "*/*",
      "user-agent": "curl/7.81.0",
      "x-hello-header": "hello"
    },
    "method": "GET",
    "origin": "",
    "url": "/greet",
    "envs": {},
    "host_name": "dubboclient-74f8cfcd74-rw5fj",
    "body": ""
  },
  "response": {
    "greeting": "hello world !",
    "attachments": {
      ":method": "POST",
      ":mtls-mode": "DISABLE",
      ":path": "/greet.GreetService/Greet",
      ":scheme": "http",
      ":source.principal": "",
      "INSTANCE_IP": "10.10.188.40",
      "POD_NAME": "httpbin-6b89db567f-p5zsg",
      "accept": "*/*",
      "accept-encoding": "identity",
      "content-type": "application/grpc+proto",
      "grpc-accept-encoding": "gzip",
      "grpc-timeout": "2999991u",
      "interface": "greet.GreetService",
      "retries": "",
      "te": "trailers",
      "timeout": "",
      "user-agent": "curl/7.81.0",
      "x-hello-header": "hello"
    }
  }
}

```

DISABLE MODE 下 dubbo namespace 和 legacy Namespace 都可以访问,而且都是走明文传输。


# 三、测试 RequestAuthentication

```shell
DUBBO_SLEEP_POD=`kubectl get pod -l app=sleep -n dubbo -o jsonpath={.items..metadata.name}`
LEGACY_SLEEP_POD=`kubectl get pod -l app=sleep -n legacy -o jsonpath={.items..metadata.name}`


```
## 1. token & keyset 

```shell
aud: dev       token1: eyJhbGciOiJSUzI1NiIsImtpZCI6ImtleTEiLCJ0eXAiOiJKV1QifQ.eyJhdWQiOlsiZGV2Il0sImV4cCI6MjAyNzU4NDc0MiwiaWF0IjoxNzEyMjI0NzQyLCJpc3MiOiJkdWJiby5hcGFjaGUub3JnIiwic3ViIjoic3BpZmZlOi8vY2x1c3Rlci5sb2NhbC9ucy9kdWJiby9zYS9kdWJib2NsaWVudCJ9.CRpc1EMSl6h1AthyaONdZxdrj5_bYb-LBN4QIIfapLebMPUxJ3i4ynUas9Mwo6I0ldcYE9flFtjprWuozIw-J6D6KeGqQUUVukejivRcWkTWeA0s4fxmievtZGofq1jXa40xjiC8ab8BgHBHQ6yufGOdEzaO028gPclApmRM8dsrnaeB_uKLLwrWd2b6Ha-FAyjLMJtauEXyAWs4cxoxnEXdfUGxMfKmWGzye4NBeHMyTfzwM0Fp4W54sUbDvknWJ9Tyo1XZez7FwwisF6KftvESO-XIIKwRqfbVNyl_cUXlTJySgvnoZJZ8yXZsd3c3bqn-bbJHSv8dSAEAwgCEEw
aud: test      token2: eyJhbGciOiJSUzI1NiIsImtpZCI6ImtleTEiLCJ0eXAiOiJKV1QifQ.eyJhdWQiOlsidGVzdCJdLCJleHAiOjIwMjc1ODQ3NDIsImlhdCI6MTcxMjIyNDc0MiwiaXNzIjoiZHViYm8uYXBhY2hlLm9yZyIsInN1YiI6InNwaWZmZTovL2NsdXN0ZXIubG9jYWwvbnMvZHViYm8vc2EvaHR0cGJpbiJ9.WM4aUUnXu_M3Ay2rteTjXxfcf9JHtGd8yBSJ1jDkptDWC0ThOVrwRWCHpCeGsywEC-GEh_cKiwEoExNmvfc-f5B80Lqqb19iyWdJ9rDRXm2dqLOvvILfZxeIZe4s_eBL3-mQEuSAJks1xb8e9uDh2rqQWcqkqxH1q3_kOLF1J117xJPmUXjQxxc3zC_9ezbWaMTPn4rAcWU_aTJgZElVyKnvgsv7dmmhAUy6Y_NIdT9kUxw7cEpixk-1wiF2ITarMzujBPV5G651YX8cRux-D52eQO7BI3JB5y0RUzXrG06ts4P7aIF8ELZ9CTF-HHCTVetv93wUQtooftAJr_eU5g
aud: marketing token3: eyJhbGciOiJSUzI1NiIsImtpZCI6ImtleTEiLCJ0eXAiOiJKV1QifQ.eyJhdWQiOlsibWFya2V0aW5nIl0sImV4cCI6MjAyNzU4NDc0MiwiaWF0IjoxNzEyMjI0NzQyLCJpc3MiOiJkdWJiby5hcGFjaGUub3JnIiwic3ViIjoic3BpZmZlOi8vY2x1c3Rlci5sb2NhbC9ucy9kdWJiby9zYS9tYXJrZXRpbmcifQ.OUBvOrcim_wuJaL_kRtZ1tRzhLlaaM6S-WNg8w_k3-zp9bYBmqp44hQ04Rza9ssIoENvLsiIMTZZL67qlM61kqrn8cBLR-HdDEUmspFLb-0NrSOk0LVfoG7h0B3ODFNC6C6BJ8EIL3BRbOqQnymasrvug3p0Swjf40UII__oe0G7mx7Lg1zhMuyMIiMyaO1-hWwXzLom0f7h45k33r8yq7mQ6W8_opyo2XsSBM-KAprWg3rg0Ukl5-XDDjzZ6T83ipAspGYooVrh0sj8OpKbTEBq6nMlCisfAzZWkz6Yhp4yTZGXfgxt74UhlA4TfuHjT0QzNVyUMsra8W1hgxdYKQ
keyset: {"keys":[{"alg":"RS256","e":"AQAB","kid":"key1","kty":"RSA","n":"2UdRi0J3cg30LzItM2a_NtgjmGs9i5sq-S6Tw92-_zC3iMBFcSk60G49ockIqgXVJkhcP9aWSnsak7IkPastWmgNevLt-wWMSJOs1Aihpd1UsFyLoHYVGlsz9jSc-kcXTjqUjTFYamivWHasW34npvy2D2pW3eDMCHQJScbiM5KyLnAeFKc9nryPwNHmw8wmFekEPs5wlTM5WkrrljD_kPoqOqX6T2cLgdPOd9rymyVJ7UbKZjGK8Jwa17s0itj941AdJ36GNmgeqG74EiLeS_I1Mwm-zAzxy84rAtqZ5rsxwvrplyZG6rwDD1pUGr4tGthyM8J9EZwFE3HzOoePfQ"}]}
```

## 2. 应用 PERMISSIVE mode

```shell

kubectl apply -n dubbo -f - <<EOF
apiVersion: security.istio.io/v1beta1
kind: PeerAuthentication
metadata:
  name: default
spec:
  mtls:
    mode: PERMISSIVE
EOF

```

## 3. 应用 RequestAuthentication 

```shell
kubectl apply -n dubbo -f - <<EOF
apiVersion: security.istio.io/v1beta1
kind: RequestAuthentication
metadata:
  name: default
spec:
  jwtRules:
    - issuer: dubbo.apache.org
      audiences:
        - dev
        - test
      fromHeaders:
        - name: "Authorization"
          prefix: "Bearer "
      jwks: |
        {
          "keys": [
            {
              "alg": "RS256",
              "e": "AQAB",
              "kid": "key1",
              "kty": "RSA",
              "n": "2UdRi0J3cg30LzItM2a_NtgjmGs9i5sq-S6Tw92-_zC3iMBFcSk60G49ockIqgXVJkhcP9aWSnsak7IkPastWmgNevLt-wWMSJOs1Aihpd1UsFyLoHYVGlsz9jSc-kcXTjqUjTFYamivWHasW34npvy2D2pW3eDMCHQJScbiM5KyLnAeFKc9nryPwNHmw8wmFekEPs5wlTM5WkrrljD_kPoqOqX6T2cLgdPOd9rymyVJ7UbKZjGK8Jwa17s0itj941AdJ36GNmgeqG74EiLeS_I1Mwm-zAzxy84rAtqZ5rsxwvrplyZG6rwDD1pUGr4tGthyM8J9EZwFE3HzOoePfQ"
            }
          ]
        }
EOF
```

## 4. 用 token1 请求

```shell

kubectl exec "${DUBBO_SLEEP_POD}" -c sleep -n dubbo \
 -- curl http://dubboclient.dubbo.svc:9090/greet -s -H "Authorization: Bearer eyJhbGciOiJSUzI1NiIsImtpZCI6ImtleTEiLCJ0eXAiOiJKV1QifQ.eyJhdWQiOlsiZGV2Il0sImV4cCI6MjAyNzU4NDc0MiwiaWF0IjoxNzEyMjI0NzQyLCJpc3MiOiJkdWJiby5hcGFjaGUub3JnIiwic3ViIjoic3BpZmZlOi8vY2x1c3Rlci5sb2NhbC9ucy9kdWJiby9zYS9kdWJib2NsaWVudCJ9.CRpc1EMSl6h1AthyaONdZxdrj5_bYb-LBN4QIIfapLebMPUxJ3i4ynUas9Mwo6I0ldcYE9flFtjprWuozIw-J6D6KeGqQUUVukejivRcWkTWeA0s4fxmievtZGofq1jXa40xjiC8ab8BgHBHQ6yufGOdEzaO028gPclApmRM8dsrnaeB_uKLLwrWd2b6Ha-FAyjLMJtauEXyAWs4cxoxnEXdfUGxMfKmWGzye4NBeHMyTfzwM0Fp4W54sUbDvknWJ9Tyo1XZez7FwwisF6KftvESO-XIIKwRqfbVNyl_cUXlTJySgvnoZJZ8yXZsd3c3bqn-bbJHSv8dSAEAwgCEEw" | jq


{
  "request": {
    "args": {},
    "form": {},
    "headers": {
      "accept": "*/*",
      "authorization": "Bearer eyJhbGciOiJSUzI1NiIsImtpZCI6ImtleTEiLCJ0eXAiOiJKV1QifQ.eyJhdWQiOlsiZGV2Il0sImV4cCI6MjAyNzU4NDc0MiwiaWF0IjoxNzEyMjI0NzQyLCJpc3MiOiJkdWJiby5hcGFjaGUub3JnIiwic3ViIjoic3BpZmZlOi8vY2x1c3Rlci5sb2NhbC9ucy9kdWJiby9zYS9kdWJib2NsaWVudCJ9.CRpc1EMSl6h1AthyaONdZxdrj5_bYb-LBN4QIIfapLebMPUxJ3i4ynUas9Mwo6I0ldcYE9flFtjprWuozIw-J6D6KeGqQUUVukejivRcWkTWeA0s4fxmievtZGofq1jXa40xjiC8ab8BgHBHQ6yufGOdEzaO028gPclApmRM8dsrnaeB_uKLLwrWd2b6Ha-FAyjLMJtauEXyAWs4cxoxnEXdfUGxMfKmWGzye4NBeHMyTfzwM0Fp4W54sUbDvknWJ9Tyo1XZez7FwwisF6KftvESO-XIIKwRqfbVNyl_cUXlTJySgvnoZJZ8yXZsd3c3bqn-bbJHSv8dSAEAwgCEEw",
      "user-agent": "curl/7.81.0"
    },
    "method": "GET",
    "origin": "",
    "url": "/greet",
    "envs": {},
    "host_name": "dubboclient-6f4c544c8c-cnclb",
    "body": ""
  },
  "response": {
    "greeting": "hello world !",
    "attachments": {
      ":auth": "{\"iss\":\"dubbo.apache.org\",\"exp\":\"2034-04-02T09:59:02Z\",\"sub\":\"spiffe://cluster.local/ns/dubbo/sa/dubboclient\",\"aud\":[\"dev\"],\"iat\":\"2024-04-04T09:59:02Z\",\"nbf\":\"0001-01-01T00:00:00Z\"}",
      ":authority": "10.10.188.40:8001",
      ":method": "POST",
      ":mtls-mode": "PERMISSIVE",
      ":path": "/greet.GreetService/Greet",
      ":request.auth.audiences": "dev",
      ":request.auth.claims.aud": "dev",
      ":request.auth.claims.iss": "dubbo.apache.org",
      ":request.auth.claims.jti": "",
      ":request.auth.claims.sub": "spiffe://cluster.local/ns/dubbo/sa/dubboclient",
      ":request.auth.principal": "dubbo.apache.org/spiffe://cluster.local/ns/dubbo/sa/dubboclient",
      ":scheme": "https",
      ":source.principal": "spiffe://cluster.local/ns/dubbo/sa/dubboclient",
      ":spiffe": "spiffe://cluster.local/ns/dubbo/sa/dubboclient",
      "INSTANCE_IP": "10.10.188.40",
      "POD_NAME": "httpbin-6b89db567f-p5zsg",
      "accept": "*/*",
      "accept-encoding": "identity",
      "content-type": "application/grpc+proto",
      "grpc-accept-encoding": "gzip",
      "grpc-timeout": "2999987u",
      "interface": "greet.GreetService",
      "retries": "",
      "te": "trailers",
      "timeout": "",
      "user-agent": "curl/7.81.0"
    }
  }
}

```

用 token1 请求回复可以看到jwt token claim 内容已经解析出来，
- 在 attachments 增加一个 key 为 :auth, 值为 jwt token claims JSON: "{\"iss\":\"dubbo.apache.org\",\"exp\":\"2034-04-02T09:59:02Z\",\"sub\":\"spiffe://cluster.local/ns/dubbo/sa/dubboclient\",\"aud\":[\"dev\"],\"iat\":\"2024-04-04T09:59:02Z\",\"nbf\":\"0001-01-01T00:00:00Z\"}"
- 在 attachments 增加以 :request.auth.claims.aud, :request.auth.claims.iss, :request.auth.claims.jti, :request.auth.claims.sub, :request.auth.principal, :request.auth.audiences

## 5. 用错误 token 请求

```shell
kubectl exec "${DUBBO_SLEEP_POD}" -c sleep -n dubbo \
 -- curl http://dubboclient.dubbo.svc:9090/greet -s -H "Authorization: Bearer WrongJwtToken" | jq


Last error is unknown: jwt token verify fail.: unknown: jwt token verify fail"

```
错误 jwt token 请求返回错误内容

## 6. 用不同的 audiences 请求， token3 的 audiences为 marketing

```shell
kubectl exec "${DUBBO_SLEEP_POD}" -c sleep -n dubbo \
 -- curl http://dubboclient.dubbo.svc:9090/greet -s -H "Authorization: Bearer eyJhbGciOiJSUzI1NiIsImtpZCI6ImtleTEiLCJ0eXAiOiJKV1QifQ.eyJhdWQiOlsibWFya2V0aW5nIl0sImV4cCI6MjAyNzU4NDc0MiwiaWF0IjoxNzEyMjI0NzQyLCJpc3MiOiJkdWJiby5hcGFjaGUub3JnIiwic3ViIjoic3BpZmZlOi8vY2x1c3Rlci5sb2NhbC9ucy9kdWJiby9zYS9tYXJrZXRpbmcifQ.OUBvOrcim_wuJaL_kRtZ1tRzhLlaaM6S-WNg8w_k3-zp9bYBmqp44hQ04Rza9ssIoENvLsiIMTZZL67qlM61kqrn8cBLR-HdDEUmspFLb-0NrSOk0LVfoG7h0B3ODFNC6C6BJ8EIL3BRbOqQnymasrvug3p0Swjf40UII__oe0G7mx7Lg1zhMuyMIiMyaO1-hWwXzLom0f7h45k33r8yq7mQ6W8_opyo2XsSBM-KAprWg3rg0Ukl5-XDDjzZ6T83ipAspGYooVrh0sj8OpKbTEBq6nMlCisfAzZWkz6Yhp4yTZGXfgxt74UhlA4TfuHjT0QzNVyUMsra8W1hgxdYKQ" | jq


Last error is unknown: jwt token verify fail.: unknown: jwt token verify fail"
```

requestauthentication 只允许 audiences为 dev 或者 test请求， token3 audiences为 marketing, 被拒绝请求

# 四、测试 AuthenticationPolicy

## 1. 应用 PERMISSIVE mode

```shell
kubectl apply -n dubbo -f - <<EOF
apiVersion: security.istio.io/v1beta1
kind: PeerAuthentication
metadata:
  name: default
spec:
  mtls:
    mode: PERMISSIVE
EOF
```
## 2. 应用 Request Authentication

```shell
kubectl apply -n dubbo -f - <<EOF
apiVersion: security.istio.io/v1beta1
kind: RequestAuthentication
metadata:
  name: default
spec:
  jwtRules:
    - issuer: dubbo.apache.org
      audiences:
        - dev
        - test
      fromHeaders:
        - name: "Authorization"
          prefix: "Bearer "
      jwks: |
        {
          "keys": [
            {
              "alg": "RS256",
              "e": "AQAB",
              "kid": "key1",
              "kty": "RSA",
              "n": "2UdRi0J3cg30LzItM2a_NtgjmGs9i5sq-S6Tw92-_zC3iMBFcSk60G49ockIqgXVJkhcP9aWSnsak7IkPastWmgNevLt-wWMSJOs1Aihpd1UsFyLoHYVGlsz9jSc-kcXTjqUjTFYamivWHasW34npvy2D2pW3eDMCHQJScbiM5KyLnAeFKc9nryPwNHmw8wmFekEPs5wlTM5WkrrljD_kPoqOqX6T2cLgdPOd9rymyVJ7UbKZjGK8Jwa17s0itj941AdJ36GNmgeqG74EiLeS_I1Mwm-zAzxy84rAtqZ5rsxwvrplyZG6rwDD1pUGr4tGthyM8J9EZwFE3HzOoePfQ"
            }
          ]
        }
EOF
```

## 3. 应用 Deny All POST Request

```shell
kubectl apply -n dubbo -f  - <<EOF
apiVersion: security.istio.io/v1
kind: AuthorizationPolicy
metadata:
  name: httpbin
spec:
  action: DENY
  rules:
  - to:
    - operation:
        methods: ["POST"]
EOF
```


用 token1 请求

```shell
kubectl exec "${DUBBO_SLEEP_POD}" -c sleep -n dubbo \
-- curl http://dubboclient.dubbo.svc:9090/greet -s -H "Authorization: Bearer eyJhbGciOiJSUzI1NiIsImtpZCI6ImtleTEiLCJ0eXAiOiJKV1QifQ.eyJhdWQiOlsiZGV2Il0sImV4cCI6MjAyNzU4NDc0MiwiaWF0IjoxNzEyMjI0NzQyLCJpc3MiOiJkdWJiby5hcGFjaGUub3JnIiwic3ViIjoic3BpZmZlOi8vY2x1c3Rlci5sb2NhbC9ucy9kdWJiby9zYS9kdWJib2NsaWVudCJ9.CRpc1EMSl6h1AthyaONdZxdrj5_bYb-LBN4QIIfapLebMPUxJ3i4ynUas9Mwo6I0ldcYE9flFtjprWuozIw-J6D6KeGqQUUVukejivRcWkTWeA0s4fxmievtZGofq1jXa40xjiC8ab8BgHBHQ6yufGOdEzaO028gPclApmRM8dsrnaeB_uKLLwrWd2b6Ha-FAyjLMJtauEXyAWs4cxoxnEXdfUGxMfKmWGzye4NBeHMyTfzwM0Fp4W54sUbDvknWJ9Tyo1XZez7FwwisF6KftvESO-XIIKwRqfbVNyl_cUXlTJySgvnoZJZ8yXZsd3c3bqn-bbJHSv8dSAEAwgCEEw" | jq

unknown: request deny by rbac filter policy: ns[dubbo]-policy[httpbin]-rule[0]


kubectl exec "${LEGACY_SLEEP_POD}" -c sleep -n legacy \
-- curl http://dubboclient.legacy.svc:9090/greet -s -H "Authorization: Bearer eyJhbGciOiJSUzI1NiIsImtpZCI6ImtleTEiLCJ0eXAiOiJKV1QifQ.eyJhdWQiOlsiZGV2Il0sImV4cCI6MjAyNzU4NDc0MiwiaWF0IjoxNzEyMjI0NzQyLCJpc3MiOiJkdWJiby5hcGFjaGUub3JnIiwic3ViIjoic3BpZmZlOi8vY2x1c3Rlci5sb2NhbC9ucy9kdWJiby9zYS9kdWJib2NsaWVudCJ9.CRpc1EMSl6h1AthyaONdZxdrj5_bYb-LBN4QIIfapLebMPUxJ3i4ynUas9Mwo6I0ldcYE9flFtjprWuozIw-J6D6KeGqQUUVukejivRcWkTWeA0s4fxmievtZGofq1jXa40xjiC8ab8BgHBHQ6yufGOdEzaO028gPclApmRM8dsrnaeB_uKLLwrWd2b6Ha-FAyjLMJtauEXyAWs4cxoxnEXdfUGxMfKmWGzye4NBeHMyTfzwM0Fp4W54sUbDvknWJ9Tyo1XZez7FwwisF6KftvESO-XIIKwRqfbVNyl_cUXlTJySgvnoZJZ8yXZsd3c3bqn-bbJHSv8dSAEAwgCEEw" | jq

unknown: request deny by rbac filter policy: ns[dubbo]-policy[httpbin]-rule[0]

```


## 4. 应用只允许 dubbo namespace 访问

```shell
kubectl apply -n dubbo -f  - <<EOF
apiVersion: security.istio.io/v1
kind: AuthorizationPolicy
metadata:
  name: httpbin
spec:
  action: ALLOW
  rules:
  - from:
    - source:
        namespaces: ["dubbo"]
    to:
    - operation:
        methods: ["POST"]
EOF
```


用 token1 请求

```shell
kubectl exec "${DUBBO_SLEEP_POD}" -c sleep -n dubbo \
-- curl http://dubboclient.dubbo.svc:9090/greet -s -H "Authorization: Bearer eyJhbGciOiJSUzI1NiIsImtpZCI6ImtleTEiLCJ0eXAiOiJKV1QifQ.eyJhdWQiOlsiZGV2Il0sImV4cCI6MjAyNzU4NDc0MiwiaWF0IjoxNzEyMjI0NzQyLCJpc3MiOiJkdWJiby5hcGFjaGUub3JnIiwic3ViIjoic3BpZmZlOi8vY2x1c3Rlci5sb2NhbC9ucy9kdWJiby9zYS9kdWJib2NsaWVudCJ9.CRpc1EMSl6h1AthyaONdZxdrj5_bYb-LBN4QIIfapLebMPUxJ3i4ynUas9Mwo6I0ldcYE9flFtjprWuozIw-J6D6KeGqQUUVukejivRcWkTWeA0s4fxmievtZGofq1jXa40xjiC8ab8BgHBHQ6yufGOdEzaO028gPclApmRM8dsrnaeB_uKLLwrWd2b6Ha-FAyjLMJtauEXyAWs4cxoxnEXdfUGxMfKmWGzye4NBeHMyTfzwM0Fp4W54sUbDvknWJ9Tyo1XZez7FwwisF6KftvESO-XIIKwRqfbVNyl_cUXlTJySgvnoZJZ8yXZsd3c3bqn-bbJHSv8dSAEAwgCEEw" | jq

{
  "request": {
    "args": {},
    "form": {},
    "headers": {
      "accept": "*/*",
      "authorization": "Bearer eyJhbGciOiJSUzI1NiIsImtpZCI6ImtleTEiLCJ0eXAiOiJKV1QifQ.eyJhdWQiOlsiZGV2Il0sImV4cCI6MjAyNzU4NDc0MiwiaWF0IjoxNzEyMjI0NzQyLCJpc3MiOiJkdWJiby5hcGFjaGUub3JnIiwic3ViIjoic3BpZmZlOi8vY2x1c3Rlci5sb2NhbC9ucy9kdWJiby9zYS9kdWJib2NsaWVudCJ9.CRpc1EMSl6h1AthyaONdZxdrj5_bYb-LBN4QIIfapLebMPUxJ3i4ynUas9Mwo6I0ldcYE9flFtjprWuozIw-J6D6KeGqQUUVukejivRcWkTWeA0s4fxmievtZGofq1jXa40xjiC8ab8BgHBHQ6yufGOdEzaO028gPclApmRM8dsrnaeB_uKLLwrWd2b6Ha-FAyjLMJtauEXyAWs4cxoxnEXdfUGxMfKmWGzye4NBeHMyTfzwM0Fp4W54sUbDvknWJ9Tyo1XZez7FwwisF6KftvESO-XIIKwRqfbVNyl_cUXlTJySgvnoZJZ8yXZsd3c3bqn-bbJHSv8dSAEAwgCEEw",
      "user-agent": "curl/7.81.0"
    },
    "method": "GET",
    "origin": "",
    "url": "/greet",
    "envs": {},
    "host_name": "dubboclient-cfb77d874-svqtf",
    "body": ""
  },
  "response": {
    "greeting": "hello world !",
    "attachments": {
      ":auth": "{\"iss\":\"dubbo.apache.org\",\"exp\":\"2034-04-02T09:59:02Z\",\"sub\":\"spiffe://cluster.local/ns/dubbo/sa/dubboclient\",\"aud\":[\"dev\"],\"iat\":\"2024-04-04T09:59:02Z\",\"nbf\":\"0001-01-01T00:00:00Z\"}",
      ":authority": "10.10.188.39:8001",
      ":method": "POST",
      ":mtls-mode": "PERMISSIVE",
      ":path": "/greet.GreetService/Greet",
      ":request.auth.audiences": "dev",
      ":request.auth.claims.aud": "dev",
      ":request.auth.claims.iss": "dubbo.apache.org",
      ":request.auth.claims.jti": "",
      ":request.auth.claims.sub": "spiffe://cluster.local/ns/dubbo/sa/dubboclient",
      ":request.auth.principal": "dubbo.apache.org/spiffe://cluster.local/ns/dubbo/sa/dubboclient",
      ":scheme": "https",
      ":source.principal": "spiffe://cluster.local/ns/dubbo/sa/dubboclient",
      ":spiffe": "spiffe://cluster.local/ns/dubbo/sa/dubboclient",
      "INSTANCE_IP": "10.10.188.39",
      "POD_NAME": "httpbin-7679f75886-qbcds",
      "accept": "*/*",
      "accept-encoding": "identity",
      "content-type": "application/grpc+proto",
      "grpc-accept-encoding": "gzip",
      "grpc-timeout": "2999992u",
      "interface": "greet.GreetService",
      "retries": "",
      "te": "trailers",
      "timeout": "",
      "user-agent": "curl/7.81.0"
    }
  }
}



kubectl exec "${LEGACY_SLEEP_POD}" -c sleep -n legacy \
-- curl http://dubboclient.legacy.svc:9090/greet -s -H "Authorization: Bearer eyJhbGciOiJSUzI1NiIsImtpZCI6ImtleTEiLCJ0eXAiOiJKV1QifQ.eyJhdWQiOlsiZGV2Il0sImV4cCI6MjAyNzU4NDc0MiwiaWF0IjoxNzEyMjI0NzQyLCJpc3MiOiJkdWJiby5hcGFjaGUub3JnIiwic3ViIjoic3BpZmZlOi8vY2x1c3Rlci5sb2NhbC9ucy9kdWJiby9zYS9kdWJib2NsaWVudCJ9.CRpc1EMSl6h1AthyaONdZxdrj5_bYb-LBN4QIIfapLebMPUxJ3i4ynUas9Mwo6I0ldcYE9flFtjprWuozIw-J6D6KeGqQUUVukejivRcWkTWeA0s4fxmievtZGofq1jXa40xjiC8ab8BgHBHQ6yufGOdEzaO028gPclApmRM8dsrnaeB_uKLLwrWd2b6Ha-FAyjLMJtauEXyAWs4cxoxnEXdfUGxMfKmWGzye4NBeHMyTfzwM0Fp4W54sUbDvknWJ9Tyo1XZez7FwwisF6KftvESO-XIIKwRqfbVNyl_cUXlTJySgvnoZJZ8yXZsd3c3bqn-bbJHSv8dSAEAwgCEEw" | jq

Last error is unknown: request deny by rbac filter policy: .: unknown: request deny by rbac filter policy: "

```




## 5. 应用只允许 dubbo namespace 同时 jwt token audience 为 dev 访问

```shell
kubectl apply -n dubbo -f  - <<EOF
apiVersion: security.istio.io/v1
kind: AuthorizationPolicy
metadata:
  name: httpbin
spec:
  action: ALLOW
  rules:
  - from:
    - source:
        namespaces: ["dubbo"]
    to:
    - operation:
        methods: ["POST"]
    when:
    - key: request.auth.claims[aud]
      values: ["dev"]
EOF
```



用 token1 请求

```shell
kubectl exec "${DUBBO_SLEEP_POD}" -c sleep -n dubbo \
-- curl http://dubboclient.dubbo.svc:9090/greet -s -H "Authorization: Bearer eyJhbGciOiJSUzI1NiIsImtpZCI6ImtleTEiLCJ0eXAiOiJKV1QifQ.eyJhdWQiOlsiZGV2Il0sImV4cCI6MjAyNzU4NDc0MiwiaWF0IjoxNzEyMjI0NzQyLCJpc3MiOiJkdWJiby5hcGFjaGUub3JnIiwic3ViIjoic3BpZmZlOi8vY2x1c3Rlci5sb2NhbC9ucy9kdWJiby9zYS9kdWJib2NsaWVudCJ9.CRpc1EMSl6h1AthyaONdZxdrj5_bYb-LBN4QIIfapLebMPUxJ3i4ynUas9Mwo6I0ldcYE9flFtjprWuozIw-J6D6KeGqQUUVukejivRcWkTWeA0s4fxmievtZGofq1jXa40xjiC8ab8BgHBHQ6yufGOdEzaO028gPclApmRM8dsrnaeB_uKLLwrWd2b6Ha-FAyjLMJtauEXyAWs4cxoxnEXdfUGxMfKmWGzye4NBeHMyTfzwM0Fp4W54sUbDvknWJ9Tyo1XZez7FwwisF6KftvESO-XIIKwRqfbVNyl_cUXlTJySgvnoZJZ8yXZsd3c3bqn-bbJHSv8dSAEAwgCEEw" | jq

{
  "request": {
    "args": {},
    "form": {},
    "headers": {
      "accept": "*/*",
      "authorization": "Bearer eyJhbGciOiJSUzI1NiIsImtpZCI6ImtleTEiLCJ0eXAiOiJKV1QifQ.eyJhdWQiOlsiZGV2Il0sImV4cCI6MjAyNzU4NDc0MiwiaWF0IjoxNzEyMjI0NzQyLCJpc3MiOiJkdWJiby5hcGFjaGUub3JnIiwic3ViIjoic3BpZmZlOi8vY2x1c3Rlci5sb2NhbC9ucy9kdWJiby9zYS9kdWJib2NsaWVudCJ9.CRpc1EMSl6h1AthyaONdZxdrj5_bYb-LBN4QIIfapLebMPUxJ3i4ynUas9Mwo6I0ldcYE9flFtjprWuozIw-J6D6KeGqQUUVukejivRcWkTWeA0s4fxmievtZGofq1jXa40xjiC8ab8BgHBHQ6yufGOdEzaO028gPclApmRM8dsrnaeB_uKLLwrWd2b6Ha-FAyjLMJtauEXyAWs4cxoxnEXdfUGxMfKmWGzye4NBeHMyTfzwM0Fp4W54sUbDvknWJ9Tyo1XZez7FwwisF6KftvESO-XIIKwRqfbVNyl_cUXlTJySgvnoZJZ8yXZsd3c3bqn-bbJHSv8dSAEAwgCEEw",
      "user-agent": "curl/7.81.0"
    },
    "method": "GET",
    "origin": "",
    "url": "/greet",
    "envs": {},
    "host_name": "dubboclient-cfb77d874-svqtf",
    "body": ""
  },
  "response": {
    "greeting": "hello world !",
    "attachments": {
      ":auth": "{\"iss\":\"dubbo.apache.org\",\"exp\":\"2034-04-02T09:59:02Z\",\"sub\":\"spiffe://cluster.local/ns/dubbo/sa/dubboclient\",\"aud\":[\"dev\"],\"iat\":\"2024-04-04T09:59:02Z\",\"nbf\":\"0001-01-01T00:00:00Z\"}",
      ":authority": "10.10.188.39:8001",
      ":method": "POST",
      ":mtls-mode": "PERMISSIVE",
      ":path": "/greet.GreetService/Greet",
      ":request.auth.audiences": "dev",
      ":request.auth.claims.aud": "dev",
      ":request.auth.claims.iss": "dubbo.apache.org",
      ":request.auth.claims.jti": "",
      ":request.auth.claims.sub": "spiffe://cluster.local/ns/dubbo/sa/dubboclient",
      ":request.auth.principal": "dubbo.apache.org/spiffe://cluster.local/ns/dubbo/sa/dubboclient",
      ":scheme": "https",
      ":source.principal": "spiffe://cluster.local/ns/dubbo/sa/dubboclient",
      ":spiffe": "spiffe://cluster.local/ns/dubbo/sa/dubboclient",
      "INSTANCE_IP": "10.10.188.39",
      "POD_NAME": "httpbin-7679f75886-qbcds",
      "accept": "*/*",
      "accept-encoding": "identity",
      "content-type": "application/grpc+proto",
      "grpc-accept-encoding": "gzip",
      "grpc-timeout": "2999982u",
      "interface": "greet.GreetService",
      "retries": "",
      "te": "trailers",
      "timeout": "",
      "user-agent": "curl/7.81.0"
    }
  }
}

用 token2 请求

kubectl exec "${DUBBO_SLEEP_POD}" -c sleep -n dubbo \
-- curl http://dubboclient.dubbo.svc:9090/greet -s -H "Authorization: Bearer eyJhbGciOiJSUzI1NiIsImtpZCI6ImtleTEiLCJ0eXAiOiJKV1QifQ.eyJhdWQiOlsidGVzdCJdLCJleHAiOjIwMjc1ODQ3NDIsImlhdCI6MTcxMjIyNDc0MiwiaXNzIjoiZHViYm8uYXBhY2hlLm9yZyIsInN1YiI6InNwaWZmZTovL2NsdXN0ZXIubG9jYWwvbnMvZHViYm8vc2EvaHR0cGJpbiJ9.WM4aUUnXu_M3Ay2rteTjXxfcf9JHtGd8yBSJ1jDkptDWC0ThOVrwRWCHpCeGsywEC-GEh_cKiwEoExNmvfc-f5B80Lqqb19iyWdJ9rDRXm2dqLOvvILfZxeIZe4s_eBL3-mQEuSAJks1xb8e9uDh2rqQWcqkqxH1q3_kOLF1J117xJPmUXjQxxc3zC_9ezbWaMTPn4rAcWU_aTJgZElVyKnvgsv7dmmhAUy6Y_NIdT9kUxw7cEpixk-1wiF2ITarMzujBPV5G651YX8cRux-D52eQO7BI3JB5y0RUzXrG06ts4P7aIF8ELZ9CTF-HHCTVetv93wUQtooftAJr_eU5g" | jq

Last error is unknown: request deny by rbac filter policy: .: unknown: request deny by rbac filter policy: "

```
