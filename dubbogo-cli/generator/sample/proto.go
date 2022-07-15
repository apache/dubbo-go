package sample

const (
	protoFile = `syntax = "proto3";
package api;

option go_package = "./;api";

// The greeting service definition.
service Greeter {
  // Sends a greeting
  rpc SayHello (HelloRequest) returns (User) {}
  // Sends a greeting via stream
  rpc SayHelloStream (stream HelloRequest) returns (stream User) {}
}

// The request message containing the user's name.
message HelloRequest {
  string name = 1;
}

// The response message containing the greetings
message User {
  string name = 1;
  string id = 2;
  int32 age = 3;
}`
)

func init() {
	fileMap[protoFile] = &fileGenerator{
		path:    "./api",
		file:    "samples_api.proto",
		context: license + protoFile,
	}
}
