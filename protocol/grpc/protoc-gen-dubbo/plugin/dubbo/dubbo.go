/*
 * Licensed to the Apache Software Foundation (ASF) under one or more
 * contributor license agreements.  See the NOTICE file distributed with
 * this work for additional information regarding copyright ownership.
 * The ASF licenses this file to You under the Apache License, Version 2.0
 * (the "License"); you may not use this file except in compliance with
 * the License.  You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package dubbo

import (
	"fmt"
	"strconv"
	"strings"
)

import (
	pb "github.com/golang/protobuf/protoc-gen-go/descriptor"
	"github.com/golang/protobuf/protoc-gen-go/generator"
)

// Paths for packages used by code generated in this file,
// relative to the import_prefix of the generator.Generator.
const (
	contextPkgPath = "context"
	grpcPkgPath    = "google.golang.org/grpc"
)

func init() {
	generator.RegisterPlugin(new(dubboGrpc))
}

// grpc is an implementation of the Go protocol buffer compiler's
// plugin architecture.  It generates bindings for gRPC-dubbo support.
type dubboGrpc struct {
	gen *generator.Generator
}

// Name returns the name of this plugin, "grpc".
func (g *dubboGrpc) Name() string {
	return "dubbo"
}

// The names for packages imported in the generated code.
// They may vary from the final path component of the import path
// if the name is used by other packages.
var (
	contextPkg string
	grpcPkg    string
)

// Init initializes the plugin.
func (g *dubboGrpc) Init(gen *generator.Generator) {
	g.gen = gen
}

// Given a type name defined in a .proto, return its object.
// Also record that we're using it, to guarantee the associated import.
func (g *dubboGrpc) objectNamed(name string) generator.Object {
	g.gen.RecordTypeUse(name)
	return g.gen.ObjectNamed(name)
}

// Given a type name defined in a .proto, return its name as we will print it.
func (g *dubboGrpc) typeName(str string) string {
	return g.gen.TypeName(g.objectNamed(str))
}

// P forwards to g.gen.P.
func (g *dubboGrpc) P(args ...interface{}) { g.gen.P(args...) }

// Generate generates code for the services in the given file.
// be consistent with grpc plugin
func (g *dubboGrpc) Generate(file *generator.FileDescriptor) {
	if len(file.FileDescriptorProto.Service) == 0 {
		return
	}

	contextPkg = string(g.gen.AddImport(contextPkgPath))
	grpcPkg = string(g.gen.AddImport(grpcPkgPath))

	for i, service := range file.FileDescriptorProto.Service {
		g.generateService(file, service, i)
	}
}

// GenerateImports generates the import declaration for this file.
func (g *dubboGrpc) GenerateImports(file *generator.FileDescriptor) {
	g.P("import (")
	g.P(`"dubbo.apache.org/dubbo-go/v3/protocol/invocation"`)
	g.P(`"dubbo.apache.org/dubbo-go/v3/protocol"`)
	g.P(` ) `)
}

func unexport(s string) string { return strings.ToLower(s[:1]) + s[1:] }

// deprecationComment is the standard comment added to deprecated
// messages, fields, enums, and enum values.
var deprecationComment = "// Deprecated: Do not use."

// generateService generates all the code for the named service.
func (g *dubboGrpc) generateService(file *generator.FileDescriptor, service *pb.ServiceDescriptorProto, index int) {
	path := fmt.Sprintf("6,%d", index) // 6 means service.

	origServName := service.GetName()
	fullServName := origServName
	if pkg := file.GetPackage(); pkg != "" {
		fullServName = pkg + "." + fullServName
	}
	servName := generator.CamelCase(origServName)
	deprecated := service.GetOptions().GetDeprecated()

	g.P()
	g.P(fmt.Sprintf(`// %sClientImpl is the client API for %s service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://godoc.org/google.golang.org/grpc#ClientConn.NewStream.`, servName, servName))

	// Client interface.
	if deprecated {
		g.P("//")
		g.P(deprecationComment)
	}
	dubboSrvName := servName + "ClientImpl"
	g.P("type ", dubboSrvName, " struct {")
	for i, method := range service.Method {
		g.gen.PrintComments(fmt.Sprintf("%s,2,%d", path, i)) // 2 means method in a service.
		if method.GetOptions().GetDeprecated() {
			g.P("//")
			g.P(deprecationComment)
		}
		g.P(g.generateClientSignature(servName, method))
	}
	g.P("}")
	g.P()

	// NewClient factory.
	if deprecated {
		g.P(deprecationComment)
	}

	// add Reference method
	//func (u *GrpcGreeterImpl) Reference() string {
	//	return "GrpcGreeterImpl"
	//}
	g.P("func (c *", dubboSrvName, ") ", " Reference() string ", "{")
	g.P(`return "`, unexport(servName), `Impl"`)
	g.P("}")
	g.P()

	// add GetDubboStub method
	// func (u *GrpcGreeterImpl) GetDubboStub(cc *grpc.ClientConn) GreeterClient {
	//	return NewGreeterClient(cc)
	//}
	g.P("func (c *", dubboSrvName, ") ", " GetDubboStub(cc *grpc.ClientConn) ", servName, "Client {")
	g.P(`return New`, servName, `Client(cc)`)
	g.P("}")
	g.P()

	g.P(`
// DubboGrpcService is gRPC service
type DubboGrpcService interface {
	// SetProxyImpl sets proxy.
	SetProxyImpl(impl protocol.Invoker)
	// GetProxyImpl gets proxy.
	GetProxyImpl() protocol.Invoker
	// ServiceDesc gets an RPC service's specification.
	ServiceDesc() *grpc.ServiceDesc
}
`)

	// Server interface.
	serverType := servName + "ProviderBase"
	g.P("type ", serverType, " struct {")
	g.P("proxyImpl protocol.Invoker")
	g.P("}")
	g.P()

	// add set method
	//func (g *GreeterProviderBase) SetProxyImpl(impl protocol.Invoker) {
	//	g.proxyImpl = impl
	//}
	g.P("func (s *", serverType, ") SetProxyImpl(impl protocol.Invoker) {")
	g.P(`s.proxyImpl = impl`)
	g.P("}")
	g.P()

	// return get method
	g.P("func (s *", serverType, ") GetProxyImpl() protocol.Invoker {")
	g.P(`return s.proxyImpl`)
	g.P("}")
	g.P()

	// return reference
	g.P("func (c *", serverType, ") ", " Reference() string ", "{")
	g.P(`return "`, unexport(servName), `Impl"`)
	g.P("}")
	g.P()

	// add handler
	var handlerNames []string
	for _, method := range service.Method {
		hname := g.generateServerMethod(servName, fullServName, method)
		handlerNames = append(handlerNames, hname)
	}

	grpcserverType := servName + "Server"
	// return service desc
	g.P("func (s *", serverType, ") ServiceDesc() *grpc.ServiceDesc {")
	g.P(`return &grpc.ServiceDesc{`)
	g.P("ServiceName: ", strconv.Quote(fullServName), ",")
	g.P("HandlerType: (*", grpcserverType, ")(nil),")
	g.P("Methods: []", grpcPkg, ".MethodDesc{")
	for i, method := range service.Method {
		if method.GetServerStreaming() || method.GetClientStreaming() {
			continue
		}
		g.P("{")
		g.P("MethodName: ", strconv.Quote(method.GetName()), ",")
		g.P("Handler: ", handlerNames[i], ",")
		g.P("},")
	}
	g.P("},")
	g.P("Streams: []", grpcPkg, ".StreamDesc{")
	for i, method := range service.Method {
		if !method.GetClientStreaming() && !method.GetServerStreaming() {
			continue
		}
		g.P("{")
		g.P("StreamName: ", strconv.Quote(method.GetName()), ",")
		g.P("Handler: ", handlerNames[i], ",")
		if method.GetServerStreaming() {
			g.P("ServerStreams: true,")
		}
		if method.GetClientStreaming() {
			g.P("ClientStreams: true,")
		}
		g.P("},")
	}
	g.P("},")
	g.P("Metadata: \"", file.GetName(), "\",")
	g.P("}")
	g.P("}")
	g.P()
}

// generateClientSignature returns the client-side signature for a method.
func (g *dubboGrpc) generateClientSignature(servName string, method *pb.MethodDescriptorProto) string {
	origMethName := method.GetName()
	methName := generator.CamelCase(origMethName)
	//if reservedClientName[methName] {
	//  methName += "_"
	//}
	reqArg := ", in *" + g.typeName(method.GetInputType())
	if method.GetClientStreaming() {
		reqArg = ""
	}
	respName := "out *" + g.typeName(method.GetOutputType())
	if method.GetServerStreaming() || method.GetClientStreaming() {
		respName = servName + "_" + generator.CamelCase(origMethName) + "Client"
		return fmt.Sprintf("%s func(ctx %s.Context%s) (%s, error)", methName, contextPkg, reqArg, respName)
	}
	return fmt.Sprintf("%s func(ctx %s.Context%s, %s) error", methName, contextPkg, reqArg, respName)
}

//func (g *dubboGrpc) generateClientMethod(servName, fullServName, serviceDescVar string, method *pb.MethodDescriptorProto, descExpr string) {
//}

func (g *dubboGrpc) generateServerMethod(servName, fullServName string, method *pb.MethodDescriptorProto) string {
	methName := generator.CamelCase(method.GetName())
	hname := fmt.Sprintf("_DUBBO_%s_%s_Handler", servName, methName)
	inType := g.typeName(method.GetInputType())

	if !method.GetServerStreaming() && !method.GetClientStreaming() {
		g.P("func ", hname, "(srv interface{}, ctx ", contextPkg, ".Context, dec func(interface{}) error, interceptor ", grpcPkg, ".UnaryServerInterceptor) (interface{}, error) {")
		g.P("in := new(", inType, ")")
		g.P("if err := dec(in); err != nil { return nil, err }")

		g.P("base := srv.(DubboGrpcService)")
		g.P("args := []interface{}{}")
		g.P("args = append(args, in)")
		g.P(`invo := invocation.NewRPCInvocation("`, methName, `", args, nil)`)

		g.P("if interceptor == nil {")
		g.P("result := base.GetProxyImpl().Invoke(ctx, invo)")
		g.P("return result.Result(), result.Error()")
		g.P("}")

		g.P("info := &", grpcPkg, ".UnaryServerInfo{")
		g.P("Server: srv,")
		g.P("FullMethod: ", strconv.Quote(fmt.Sprintf("/%s/%s", fullServName, methName)), ",")
		g.P("}")

		g.P("handler := func(ctx ", contextPkg, ".Context, req interface{}) (interface{}, error) {")
		g.P("result := base.GetProxyImpl().Invoke(ctx, invo)")
		g.P("return result.Result(), result.Error()")
		g.P("}")

		g.P("return interceptor(ctx, in, info, handler)")
		g.P("}")
		g.P()
		return hname
	}
	streamType := unexport(servName) + methName + "Server"
	g.P("func ", hname, "(srv interface{}, stream ", grpcPkg, ".ServerStream) error {")
	g.P("_, ok := srv.(DubboGrpcService)")
	g.P(`invo := invocation.NewRPCInvocation("`, methName, `", nil, nil)`)
	g.P("if !ok {")
	g.P("fmt.Println(invo)")
	g.P("}")
	if !method.GetClientStreaming() {
		g.P("m := new(", inType, ")")
		g.P("if err := stream.RecvMsg(m); err != nil { return err }")
		g.P("return srv.(", servName, "Server).", methName, "(m, &", streamType, "{stream})")
	} else {
		g.P("return srv.(", servName, "Server).", methName, "(&", streamType, "{stream})")
	}
	g.P("}")
	g.P()

	return hname
}
