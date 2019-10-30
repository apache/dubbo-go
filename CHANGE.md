# Release Notes

## 1.2.0

### New Features

- Move callService to invoker & support attachments<https://github.com/apache/dubbo-go/pull/193>
- Support dynamic config center which compatible with dubbo 2.6.x & 2.7.x and commit the zookeeper impl<https://github.com/apache/dubbo-go/pull/194>
- Add service token authorization support<https://github.com/apache/dubbo-go/pull/202>
- Add accessLogFilter support<https://github.com/apache/dubbo-go/pull/214>
- Delete example in dubbo-go project<https://github.com/apache/dubbo-go/pull/228>
- Add tps limit support<https://github.com/apache/dubbo-go/pull/237>
- Add execute limit support<https://github.com/apache/dubbo-go/pull/246>

### Enhancement

- Split gettyRPCClient.close and gettyRPCClientPool.remove in protocol/dubbo/pool.go<https://github.com/apache/dubbo-go/pull/186>
- Remove client from pool before closing it<https://github.com/apache/dubbo-go/pull/190>
- Enhance the logic for fetching the local address<https://github.com/apache/dubbo-go/pull/209>
- Add protocol_conf default values<https://github.com/apache/dubbo-go/pull/221>

### Bugfixes

- GettyRPCClientPool remove deadlock<https://github.com/apache/dubbo-go/pull/183/files>
- Fix failover cluster bug and url parameter retries change int to string type<https://github.com/apache/dubbo-go/pull/195>
- Fix url params unsafe map<https://github.com/apache/dubbo-go/pull/201>
- Read protocol config by map key in config yaml instead of protocol name<https://github.com/apache/dubbo-go/pull/218>
- Fix dubbo group issues #238<https://github.com/apache/dubbo-go/pull/243>/<https://github.com/apache/dubbo-go/pull/244>

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
