# Release Notes

## 1.1.0

### New Features

- Support Java bigdecimal<https://github.com/apache/dubbo-go/pull/126>；
- Support all JDK exceptions<https://github.com/apache/dubbo-go/pull/120>；
- Support multi-version of service<https://github.com/apache/dubbo-go/pull/119>；
- Allow user set custom params for registry<https://github.com/apache/dubbo-go/pull/117>；
- Support zookeeper config center<https://github.com/apache/dubbo-go/pull/99>；
- Failsafe/Failback  Cluster Strategy<https://github.com/apache/dubbo-go/pull/136>;

### Enhancement

- Use time wheel instead of time.After to defeat timer object memory leakage<https://github.com/apache/dubbo-go/pull/130> ；

### Bugfixes

- Preventing dead loop when got zookeeper unregister event<https://github.com/apache/dubbo-go/pull/129>；
- Delete ineffassign<https://github.com/apache/dubbo-go/pull/127>；
- Add wg.Done() for mockDataListener<https://github.com/apache/dubbo-go/pull/118>；
- Delete wrong spelling words<https://github.com/apache/dubbo-go/pull/107>；
- Use sync.Map to defeat from gettyClientPool deadlock<https://github.com/apache/dubbo-go/pull/106>；
- Handle panic when function args list is empty<https://github.com/apache/dubbo-go/pull/98>；
- url.Values is not safe map<https://github.com/apache/dubbo-go/pull/172>;
