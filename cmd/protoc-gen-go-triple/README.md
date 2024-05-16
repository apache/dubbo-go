# protoc-gen-go-triple

This tool generates Go language bindings of `service`s in protobuf definition
files for Dubbo.

For dubbo-go 3.2.0+ version users, please use `protoc-gen-go-triple` 3.0.0 and above versions. For other dubbo-go users, it's also recommended to user `protoc-gen-go-triple` 3.0.0 and above versions. To generate stubs compatible with dubbo-go 3.1.x and below, please set the following option:

```
  protoc --go-triple_out=useOldVersion=true[,other options...]:. \
```