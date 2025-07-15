# protoc-gen-triple-openapi

`protoc-gen-triple-openapi` is a protoc plugin to generate OpenAPI v3 documentation from protobuf files that use the Triple protocol.

## Getting Started

### Installation

To install `protoc-gen-triple-openapi`, you need to have Go installed and configured. Then, you can use the following command to install the plugin:

```shell
go install github.com/apache/dubbo-go/tools/protoc-gen-triple-openapi
```

## Usage

To use `protoc-gen-triple-openapi`, you need to have a `.proto` file that defines your service. For example, you can have a file named `greet.proto` with the following content:

```proto
syntax = "proto3";

package org.apache.dubbo.samples.greet;

option go_package = "github.com/apache/dubbo-go-samples/api";

// The greeting service definition.
service GreetService {
  // Sends a greeting
  rpc Greet(GreetRequest) returns (GreetResponse) {}
}

// The request message containing the user's name.
message GreetRequest {
  string name = 1;
}

// The response message containing the greetings
message GreetResponse {
  string greeting = 1;
}
```

Then, you can use the following command to generate the OpenAPI documentation:

```shell
protoc --proto_path=. --triple-openapi_out=. greet.proto
```

This will generate a file named `greet.openapi.yaml` with the OpenAPI documentation.

## License

This project is licensed under the Apache License 2.0 - see the [LICENSE](LICENSE) file for details.

## Contributing

Contributions are welcome! Please read the [contributing guidelines](CONTRIBUTING.md) for more information.