# consul

Examples for consul registry. Before running examples below, make sure that consul has been start.

## go-server

```
$ cd examples/consul/go-server
$ export CONF_PROVIDER_FILE_PATH="config/server.yml"
$ export APP_LOG_CONF_FILE="config/log.yml"
$ go run .
```

## go-client

```
$ cd examples/consul/go-client
$ export CONF_CONSUMER_FILE_PATH="config/client.yml"
$ export APP_LOG_CONF_FILE="config/log.yml"
$ go run .
```

## java-server

```
$ cd example/consul/java-server
$ gradle build
$ java -jar build/libs/java-server-1.0-SNAPSHOT.jar
```

## java-client

```
$ cd examples/consul/java-client
$ gradle build
$ java -jar build/libs/java-client-1.0-SNAPSHOT.jar
```