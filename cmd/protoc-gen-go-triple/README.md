

# protoc-gen-go-triple

The `protoc-gen-go-triple` tool generates Go language bindings for Dubbo `service`s based on protobuf definition files.

For users of dubbo-go version 3.2.0 and above, please use `protoc-gen-go-triple` version 3.0.0 or higher. It is also recommended for other dubbo-go users to use `protoc-gen-go-triple` version 3.0.0 or higher. To generate stubs compatible with dubbo-go version 3.1.x and below, please set the following option:

```
protoc --go-triple_out=useOldVersion=true[,other options...]:.
```

## Prerequisites

Before using `protoc-gen-go-triple`, make sure you have the following prerequisites installed on your system:

- Go (version 1.17 or higher)
- Protocol Buffers (version 3.0 or higher)

## Installation

To install `protoc-gen-go-triple`, you can use the `go get` command:

```shell
go get dubbo.apache.org/dubbo-go/v3/cmd/protoc-gen-go-triple
```

Alternatively, you can clone the GitHub repository and build the binary manually:

```shell
git clone https://github.com/apache/dubbo-go.git
cd cmd/protoc-gen-go-triple
go build
```

Make sure to add the resulting binary to your system's PATH.

## Usage

To generate Triple code from your Protocol Buffer files, use the `protoc` compiler with the `protoc-gen-go-triple` plugin. Here's an example command:

```shell
protoc --go_out=. --go_opt=paths=source_relative --go-triple_out=. your_file.proto
```

Both the `--go_out` flag and `--go-triple_out` flag should be set to `.`. Please set the generated file path in the proto file using the `go_package` option.

## Example

Let's say you have a Protocol Buffer file named `greet.proto`, and you want to generate Triple Go code from it.

```proto
syntax = "proto3";

package greet;


option go_package = "dubbo.apache.org/dubbo-go/v3/protocol/triple/internal/proto/dubbo3_gen;greet";

message GreetRequest {
  string name = 1;
}

message GreetResponse {
  string greeting = 1;
}

service GreetService {
  rpc Greet(GreetRequest) returns (GreetResponse) {}
}
```

The `package` determines the interface path exposed by the Triple service after it is started, which would be:

```https
http://127.0.0.1:20000/greet.GreetService/Greet
```

The `go_package` option determines the file generation path and package name

Both parts are indispensable. The directory for the file is determined before `;`, and the package to be generated is determined after `;`.

Resulting in the following directory structure:

```
dubbo-go/protocol/triple/internal/proto/
|-triple_gen
		greet.pb.go
		greet.triple.go
```

The package for `greet.pb.go` and `greet.triple.go` are `greet`

You can use the following command to generate the Go code:

```shell
protoc --go_out=. --go_opt=paths=source_relative --go-triple_out=. greet.proto
```

This will generate the Go code for the Protobuf code in `greet.pb.go` and the Triple code in `greet.triple.go`.

