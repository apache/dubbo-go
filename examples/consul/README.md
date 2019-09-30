# consul

Examples for consul registry. Before running examples below, make sure that consul has been started.

## requirement

- consul
- go 1.12
- java 8
- maven 3.6.1

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
$ cd examples/consul/java-server
$ mvn clean package
$ java -jar target/java-server-1.0.0.jar
```

## java-client

```
$ cd examples/consul/java-client
$ mvn clean package
$ java -jar target/java-client-1.0.0.jar
```