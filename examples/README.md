# examples

Examples of dubbo-go

## What does this contain

* helloworld

    A simplest example. It contain 'go-client', 'go-server', 'java-server' of dubbo protocol. 

* general

    A general example. It had validated zookeeper registry and different parameter lists of service. 
  And it has a comprehensive testing with dubbo/jsonrpc protocol. You can refer to it to create your first complete dubbo-go project.

* generic

    A generic reference example. It show how to use generic reference of dubbo-go.

* configcenter

    Some examples of different config center. There is only one -- zookeeper at present.

## How to build and run

> Take `helloworld` as an example

java server

```bash
cd helloworld/dubbo/java-server
sh build.sh

cd ./target
tar -zxvf user-info-server-0.2.0-assembly.tar.gz
cd ./user-info-server-0.2.0
sh ./bin/server.sh start
```

java client

```bash
cd helloworld/dubbo/java-client
sh build.sh

cd ./target
tar -zxvf user-info-client-0.2.0-assembly.tar.gz
cd ./user-info-client-0.2.0
sh ./bin/server.sh start
```

go server

* $ARCH = [linux, mac, windows] and $ENV = [dev, release, test]

```bash
cd helloworld/dubbo/go-server
sh ./assembly/$ARCH/$ENV.sh

cd ./target/linux/user_info_server-0.3.1-20190517-0930-release
# $SUFFIX is a suffix of config file,
# such as server_zookeeper.yml when $SUFFIX is "zookeeper", 
# if $SUFFIX = "", default server.yml
sh ./bin/load.sh start $SUFFIX
```

go client

* $ARCH = [linux, mac, windows] and $ENV = [dev, release, test]

```bash
cd helloworld/dubbo/go-client
sh ./assembly/$ARCH/$ENV.sh

cd ./target/linux/user_info_client-0.3.1-20190517-0921-release
# $SUFFIX is a suffix of config file,
# such as client_zookeeper.yml when $SUFFIX = zookeeper", 
# if $SUFFIX = "", config file is client.yml
sh ./bin/load_user_info_client.sh start $SUFFIX
```
