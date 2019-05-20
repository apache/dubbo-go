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
```bash
cd dubbo/go-server
#linux, mac windows represent the os
#release, dev and test represent the environment
sh ./assembly/linux/release.sh
```

go client
```bash
cd dubbo/go-client
#linux, mac windows represent the os
#release, dev and test represent the environment
sh ./assembly/linux/release.sh
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
> It must not listen on IP 127.0.0.1 when called by java-client.
> You should change IP in dubbo/go-server/target/linux/user_info_server-0.3.1-20190517-0930-release/conf/server.yml
```bash
cd dubbo/go-server/target/linux/user_info_server-0.3.1-20190517-0930-release
sh ./bin/load.sh start
```

go client
```bash
cd dubbo/go-client/target/linux/user_info_client-0.3.1-20190517-0921-release
sh ./bin/load_user_info_client.sh start
```

## jsonrpc
Similar to dubbo
