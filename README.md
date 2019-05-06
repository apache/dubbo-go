# dubbo-go #
---
Apache Dubbo Golang Implementation.

## License

Apache License, Version 2.0

## Feature list ##

+ 1 Transport: HTTP(√)
+ 2 Codec:  JsonRPC(√), Hessian(X)
+ 3 Service discovery：Service Register(√), Service Watch(√)
+ 4 Registry: ZooKeeper(√), Etcd(X), Redis(X)
+ 5 Strategy: Failover(√), Failfast(√)
+ 6 Load Balance: Random(√), RoundRobin(√)
+ 7 Role: Consumer(√), Provider(√)

## Code Example

The subdirectory examples shows how to use dubbo-go. Please read the examples/readme.md carefully to learn how to dispose the configuration and compile the program.


## Todo list

- [ ] Tcp Transport and Hessian2 protocol
- [ ] Network
  - [ ] Fuse
  - [ ] Rate Limit
  - [ ] Trace
  - [ ] Metrics
  - [ ] Load Balance
