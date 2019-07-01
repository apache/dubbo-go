# examples

Examples of go-for-apache-dubbo

## dubbo

#### Build by these command

java server
```bash
cd dubbo/java-server
sh build.sh
```

java client
```bash
cd dubbo/java-client
sh build.sh
```

go server

* sh ./assembly/\[os]/\[environment].sh
```bash
cd dubbo/go-server
# $ARCH = [linux, mac, windows] and $ENV = [dev, release, test]
sh ./assembly/$ARCH/$ENV.sh
```

go client
```bash
cd dubbo/go-client
# $ARCH = [linux, mac, windows] and $ENV = [dev, release, test]
sh ./assembly/$ARCH/$ENV.sh
```

#### Run by these command:

java server
```bash
cd dubbo/java-server/target
tar -zxvf user-info-server-0.2.0-assembly.tar.gz
cd ./user-info-server-0.2.0
sh ./bin/server.sh start
```

java client
```bash
cd dubbo/java-client/target
tar -zxvf user-info-client-0.2.0-assembly.tar.gz
cd ./user-info-client-0.2.0
sh ./bin/server.sh start
```

go server
```bash
cd dubbo/go-server/target/linux/user_info_server-0.3.1-20190517-0930-release
#conf suffix appoint config file, 
#such as server_zookeeper.yml when "sh ./bin/load.sh start is zookeeper", 
#default server.yml
sh ./bin/load.sh start [conf suffix]
```

go client
```bash
cd dubbo/go-client/target/linux/user_info_client-0.3.1-20190517-0921-release
# $SUFFIX is a suffix of config file,
# such as client_zookeeper.yml when $SUFFIX = zookeeper", 
# if $SUFFIX = "", config file is client.yml
sh ./bin/load_user_info_client.sh start $SUFFIX
```

## jsonrpc
Similar to dubbo
