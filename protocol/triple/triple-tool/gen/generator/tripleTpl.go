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

package generator

import (
	"html/template"
	"log"
	"strings"
)

var (
	Tpls                   []*template.Template
	TplPreamble            *template.Template
	TplPackage             *template.Template
	TplImport              *template.Template
	TplTotal               *template.Template
	TplTypeCheck           *template.Template
	TplClientInterface     *template.Template
	TplClientInterfaceImpl *template.Template
	TplClientImpl          *template.Template
	TplMethodInfo          *template.Template
	TplHandler             *template.Template
	TplServerImpl          *template.Template
	TplServerInfo          *template.Template
	TplUnImpl              *template.Template
)

func init() {
	var err error
	TplPreamble, err = template.New("preamble").Parse(PreambleTpl)
	if err != nil {
		log.Fatal(err)
	}
	TplPackage, err = template.New("package").Parse(PackageTpl)
	if err != nil {
		log.Fatal(err)
	}
	TplImport, err = template.New("import").Parse(ImportTpl)
	if err != nil {
		log.Fatal(err)
	}
	TplTotal, err = template.New("total").Parse(TotalTpl)
	if err != nil {
		log.Fatal(err)
	}
	TplTypeCheck, err = template.New("typeCheck").Parse(TypeCheckTpl)
	if err != nil {
		log.Fatal(err)
	}
	TplClientInterface, err = template.New("clientInterface").Parse(InterfaceTpl)
	if err != nil {
		log.Fatal(err)
	}
	TplClientInterfaceImpl, err = template.New("clientInterfaceImpl").Funcs(template.FuncMap{
		"lower": func(s string) string {
			if s == "" {
				return ""
			}
			return strings.ToLower(s[:1]) + s[1:]
		},
	}).Parse(InterfaceImplTpl)
	if err != nil {
		log.Fatal(err)
	}
	TplClientImpl, err = template.New("clientImpl").Funcs(template.FuncMap{
		"lower": func(s string) string {
			if s == "" {
				return ""
			}
			return strings.ToLower(s[:1]) + s[1:]
		},
	}).Parse(ClientImplTpl)
	if err != nil {
		log.Fatal(err)
	}
	TplMethodInfo, err = template.New("methodInfo").Funcs(template.FuncMap{
		"last": func(index, length int) bool {
			return index == length-1
		},
	}).Parse(MethodInfoTpl)
	if err != nil {
		log.Fatal(err)
	}
	TplHandler, err = template.New("handler").Parse(HandlerTpl)
	if err != nil {
		log.Fatal(err)
	}
	TplServerImpl, err = template.New("serverImpl").Funcs(template.FuncMap{
		"lower": func(s string) string {
			if s == "" {
				return ""
			}
			return strings.ToLower(s[:1]) + s[1:]
		},
	}).Parse(ServerImplTpl)
	if err != nil {
		log.Fatal(err)
	}
	TplServerInfo, err = template.New("serverInfo").Funcs(template.FuncMap{
		"lower": func(s string) string {
			if s == "" {
				return ""
			}
			return strings.ToLower(s[:1]) + s[1:]
		},
	}).Parse(ServiceInfoTpl)
	if err != nil {
		log.Fatal(err)
	}
	TplUnImpl, err = template.New("unImpl").Parse(UnImplServiceTpl)
	if err != nil {
		log.Fatal(err)
	}

	Tpls = append(Tpls, TplPreamble)
	Tpls = append(Tpls, TplPackage)
	Tpls = append(Tpls, TplImport)
	Tpls = append(Tpls, TplTotal)
	Tpls = append(Tpls, TplTypeCheck)
	Tpls = append(Tpls, TplClientInterface)
	Tpls = append(Tpls, TplClientInterfaceImpl)
	Tpls = append(Tpls, TplClientImpl)
	Tpls = append(Tpls, TplMethodInfo)
	Tpls = append(Tpls, TplHandler)
	Tpls = append(Tpls, TplServerImpl)
	Tpls = append(Tpls, TplServerInfo)
	//Tpls = append(Tpls, TplUnImpl)
}

const PreambleTpl = `// Code generated by protoc-gen-triple. DO NOT EDIT.
//
// Source: {{.Source}}
`

const PackageTpl = `package {{.Package}}triple`

const ImportTpl = `

import (
	context "context"
	{{if .IsStream}}"net/http"{{end}}
)

import (
	client "dubbo.apache.org/dubbo-go/v3/client"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	triple_protocol "dubbo.apache.org/dubbo-go/v3/protocol/triple/triple_protocol"
	"dubbo.apache.org/dubbo-go/v3/server"
)

import (
	proto "{{.Import}}"
)

`

const TotalTpl = `// This is a compile-time assertion to ensure that this generated file and the Triple package
// are compatible. If you get a compiler error that this constant is not defined, this code was
// generated with a version of Triple newer than the one compiled into your binary. You can fix the
// problem by either regenerating this code with an older version of Triple or updating the Triple
// version compiled into your binary.
const _ = triple_protocol.IsAtLeastVersion0_1_0
{{$t := .}}{{range $s := .Services}}
const (
	// {{$s.ServiceName}}Name is the fully-qualified name of the {{$s.ServiceName}} service.
	{{$s.ServiceName}}Name = "{{$t.ProtoPackage}}.{{$s.ServiceName}}"
)

// These constants are the fully-qualified names of the RPCs defined in this package. They're
// exposed at runtime as procedure and as the final two segments of the HTTP route.
//
// Note that these are different from the fully-qualified method names used by
// google.golang.org/protobuf/reflect/protoreflect. To convert from these constants to
// reflection-formatted method names, remove the leading slash and convert the remaining slash to a
// period.
const (
{{range $s.Methods}}	// {{$s.ServiceName}}{{.MethodName}}Procedure is the fully-qualified name of the {{$s.ServiceName}}'s {{.MethodName}} RPC.
	{{$s.ServiceName}}{{.MethodName}}Procedure = "/{{$t.ProtoPackage}}.{{$s.ServiceName}}/{{.MethodName}}"
{{end}}){{end}}

`

const TypeCheckTpl = `var ({{$t := .}}{{range $s := .Services}}
	_ {{.ServiceName}} = (*{{.ServiceName}}Impl)(nil)	
	{{range $s.Methods}}{{if or .StreamsReturn .StreamsRequest}}
	_ {{$s.ServiceName}}_{{.MethodName}}Client = (*{{$s.ServiceName}}{{.MethodName}}Client)(nil){{end}}{{end}}
	{{range $s.Methods}}{{if or .StreamsReturn .StreamsRequest}}
	_ {{$s.ServiceName}}_{{.MethodName}}Server = (*{{$s.ServiceName}}{{.MethodName}}Server)(nil){{end}}{{end}}{{end}}
)	

`

const InterfaceTpl = `// {{$t := .}}{{range $s := .Services}}{{.ServiceName}} is a client for the {{$t.ProtoPackage}}.{{$s.ServiceName}} service.
type {{$s.ServiceName}} interface { {{- range $s.Methods}}
	{{.MethodName}}(ctx context.Context{{if .StreamsRequest}}{{else}}, req *proto.{{.RequestType}}{{end}}, opts ...client.CallOption) {{if or .StreamsReturn .StreamsRequest}}({{$s.ServiceName}}_{{.MethodName}}Client, error){{else}}(*proto.{{.ReturnType}}, error){{end}}{{end}}
}{{end}}

`

const InterfaceImplTpl = `{{$t := .}}{{range $s := .Services}}// New{{.ServiceName}} constructs a client for the {{$t.Package}}.{{.ServiceName}} service. 
func New{{.ServiceName}}(cli *client.Client) ({{.ServiceName}}, error) {
	if err := cli.Init(&{{.ServiceName}}_ClientInfo); err != nil {
		return nil, err
	}
	return &{{.ServiceName}}Impl{
		cli: cli,
	}, nil
}

// {{.ServiceName}}Impl implements {{.ServiceName}}.
type {{.ServiceName}}Impl struct {
	cli *client.Client
}
{{range .Methods}}{{if .StreamsRequest}}{{if .StreamsReturn}}
func (c *{{$s.ServiceName}}Impl) {{.MethodName}}(ctx context.Context, opts ...client.CallOption) ({{$s.ServiceName}}_{{.MethodName}}Client, error) {
	stream, err := c.cli.CallBidiStream(ctx, "{{$t.ProtoPackage}}.{{$s.ServiceName}}", "{{.MethodName}}", opts...)
	if err != nil {
		return nil, err
	}
	rawStream := stream.(*triple_protocol.BidiStreamForClient)
	return &{{$s.ServiceName}}{{.MethodName}}Client{rawStream}, nil
}
{{else}}
func (c *{{$s.ServiceName}}Impl) {{.MethodName}}(ctx context.Context, opts ...client.CallOption) ({{$s.ServiceName}}_{{.MethodName}}Client, error) {
	stream, err := c.cli.CallClientStream(ctx, "{{$t.ProtoPackage}}.{{$s.ServiceName}}", "{{.MethodName}}", opts...)
	if err != nil {
		return nil, err
	}
	rawStream := stream.(*triple_protocol.ClientStreamForClient)
	return &{{$s.ServiceName}}{{.MethodName}}Client{rawStream}, nil
}
{{end}}{{else}}{{if .StreamsReturn}}
func (c *{{$s.ServiceName}}Impl) {{.MethodName}}(ctx context.Context, req *proto.{{.RequestType}}, opts ...client.CallOption) ({{$s.ServiceName}}_{{.MethodName}}Client, error) {
	stream, err := c.cli.CallServerStream(ctx, req, "{{$t.ProtoPackage}}.{{$s.ServiceName}}", "{{.MethodName}}", opts...)
	if err != nil {
		return nil, err
	}
	rawStream := stream.(*triple_protocol.ServerStreamForClient)
	return &{{$s.ServiceName}}{{.MethodName}}Client{rawStream}, nil
}
{{else}}
func (c *{{$s.ServiceName}}Impl) {{.MethodName}}(ctx context.Context, req *proto.{{.RequestType}}, opts ...client.CallOption) (*proto.{{.ReturnType}}, error) {
	resp := new(proto.{{.ReturnType}})
	if err := c.cli.CallUnary(ctx, req, resp, "{{$t.ProtoPackage}}.{{$s.ServiceName}}", "{{.MethodName}}", opts...); err != nil {
		return nil, err
	}
	return resp, nil
}
{{end}}{{end}}{{end}}{{end}}
`

const ClientImplTpl = `{{$t := .}}{{range $s := .Services}}{{range .Methods}}{{if .StreamsRequest}}{{if .StreamsReturn}}
type {{$s.ServiceName}}_{{.MethodName}}Client interface {
	Spec() triple_protocol.Spec
	Peer() triple_protocol.Peer
	Send(*proto.{{.RequestType}}) error
	RequestHeader() http.Header
	CloseRequest() error
	Recv() (*proto.{{.ReturnType}}, error)
	ResponseHeader() http.Header
	ResponseTrailer() http.Header
	CloseResponse() error
}

type {{$s.ServiceName}}{{.MethodName}}Client struct {
	*triple_protocol.BidiStreamForClient
}

func (cli *{{$s.ServiceName}}{{.MethodName}}Client) Send(msg *proto.{{.RequestType}}) error {
	return cli.BidiStreamForClient.Send(msg)
}

func (cli *{{$s.ServiceName}}{{.MethodName}}Client) Recv() (*proto.{{.ReturnType}}, error) {
	msg := new(proto.{{.ReturnType}})
	if err := cli.BidiStreamForClient.Receive(msg); err != nil {
		return nil, err
	}
	return msg, nil
}
{{else}}
type {{$s.ServiceName}}_{{.MethodName}}Client interface {
	Spec() triple_protocol.Spec
	Peer() triple_protocol.Peer
	Send(*proto.{{.RequestType}}) error
	RequestHeader() http.Header
	CloseAndRecv() (*proto.{{.ReturnType}}, error)
	Conn() (triple_protocol.StreamingClientConn, error)
}

type {{$s.ServiceName}}{{.MethodName}}Client struct {
	*triple_protocol.ClientStreamForClient
}

func (cli *{{$s.ServiceName}}{{.MethodName}}Client) Send(msg *proto.{{.RequestType}}) error {
	return cli.ClientStreamForClient.Send(msg)
}

func (cli *{{$s.ServiceName}}{{.MethodName}}Client) CloseAndRecv() (*proto.{{.ReturnType}}, error) {
	msg := new(proto.{{.ReturnType}})
	resp := triple_protocol.NewResponse(msg)
	if err := cli.ClientStreamForClient.CloseAndReceive(resp); err != nil {
		return nil, err
	}
	return msg, nil
}

func (cli *{{$s.ServiceName}}{{.MethodName}}Client) Conn() (triple_protocol.StreamingClientConn, error) {
	return cli.ClientStreamForClient.Conn()
}
{{end}}{{else}}{{if .StreamsReturn}}
type {{$s.ServiceName}}_{{.MethodName}}Client interface {
	Recv() bool
	ResponseHeader() http.Header
	ResponseTrailer() http.Header
	Msg() *proto.{{.ReturnType}}
	Err() error
	Conn() (triple_protocol.StreamingClientConn, error)
	Close() error
}

type {{$s.ServiceName}}{{.MethodName}}Client struct {
	*triple_protocol.ServerStreamForClient
}

func (cli *{{$s.ServiceName}}{{.MethodName}}Client) Recv() bool {
	msg := new(proto.{{.ReturnType}})
	return cli.ServerStreamForClient.Receive(msg)
}

func (cli *{{$s.ServiceName}}{{.MethodName}}Client) Msg() *proto.{{.ReturnType}} {
	msg := cli.ServerStreamForClient.Msg()
	if msg == nil {
		return new(proto.{{.ReturnType}})
	}
	return msg.(*proto.{{.ReturnType}})
}

func (cli *{{$s.ServiceName}}{{.MethodName}}Client) Conn() (triple_protocol.StreamingClientConn, error) {
	return cli.ServerStreamForClient.Conn()
}{{end}}{{end}}{{end}}{{end}}

`

const MethodInfoTpl = `{{$t := .}}{{range $i, $s := .Services}}var {{.ServiceName}}_ClientInfo = client.ClientInfo{
	InterfaceName: "{{$t.Package}}.{{.ServiceName}}",
	MethodNames:   []string{ {{- range $j, $m := .Methods}}"{{.MethodName}}"{{if last $j (len $s.Methods)}}{{else}},{{end}}{{end -}} },
	ClientInjectFunc: func(dubboCliRaw interface{}, cli *client.Client) {
		dubboCli := dubboCliRaw.({{$s.ServiceName}}Impl)
		dubboCli.cli = cli
	},
}{{end}}

`

const HandlerTpl = `{{$t := .}}{{range $s := .Services}}// {{.ServiceName}}Handler is an implementation of the {{$t.ProtoPackage}}.{{.ServiceName}} service.
type {{.ServiceName}}Handler interface { {{- range $s.Methods}}
	{{.MethodName}}(context.Context, {{if .StreamsRequest}}{{$s.ServiceName}}_{{.MethodName}}Server{{else}}*proto.{{.RequestType}}{{if .StreamsReturn}}, {{$s.ServiceName}}_{{.MethodName}}Server{{end}}{{end}}) {{if .StreamsReturn}}error{{else}}(*proto.{{.ReturnType}}, error){{end}}{{end}}
}

func Register{{.ServiceName}}Handler(srv *server.Server, hdlr {{.ServiceName}}Handler, opts ...server.ServiceOption) error {
	return srv.Register(hdlr, &{{.ServiceName}}_ServiceInfo, opts...)
}{{end}}
`

const ServerImplTpl = `{{$t := .}}{{range $s := .Services}}{{range .Methods}}{{if .StreamsRequest}}{{if .StreamsReturn}}
type {{$s.ServiceName}}_{{.MethodName}}Server interface {
	Send(*proto.{{.ReturnType}}) error
	Recv() (*proto.{{.RequestType}}, error)
	Spec() triple_protocol.Spec
	Peer() triple_protocol.Peer
	RequestHeader() http.Header
	ResponseHeader() http.Header
	ResponseTrailer() http.Header
	Conn() triple_protocol.StreamingHandlerConn
}

type {{$s.ServiceName}}{{.MethodName}}Server struct {
	*triple_protocol.BidiStream
}

func (srv *{{$s.ServiceName}}{{.MethodName}}Server) Send(msg *proto.{{.ReturnType}}) error {
	return srv.BidiStream.Send(msg)
}

func (srv {{$s.ServiceName}}{{.MethodName}}Server) Recv() (*proto.{{.RequestType}}, error) {
	msg := new(proto.{{.RequestType}})
	if err := srv.BidiStream.Receive(msg); err != nil {
		return nil, err
	}
	return msg, nil
}
{{else}}
type {{$s.ServiceName}}_{{.MethodName}}Server interface {
	Spec() triple_protocol.Spec
	Peer() triple_protocol.Peer
	Recv() bool
	RequestHeader() http.Header
	Msg() *proto.{{.RequestType}}
	Err() error
	Conn() triple_protocol.StreamingHandlerConn
}

type {{$s.ServiceName}}{{.MethodName}}Server struct {
	*triple_protocol.ClientStream
}

func (srv *{{$s.ServiceName}}{{.MethodName}}Server) Recv() bool {
	msg := new(proto.{{.RequestType}})
	return srv.ClientStream.Receive(msg)
}

func (srv *{{$s.ServiceName}}{{.MethodName}}Server) Msg() *proto.{{.RequestType}} {
	msgRaw := srv.ClientStream.Msg()
	if msgRaw == nil {
		return new(proto.{{.RequestType}})
	}
	return msgRaw.(*proto.{{.RequestType}})
}
{{end}}{{else}}{{if .StreamsReturn}}
type {{$s.ServiceName}}_{{.MethodName}}Server interface {
	Send(*proto.{{.ReturnType}}) error
	ResponseHeader() http.Header
	ResponseTrailer() http.Header
	Conn() triple_protocol.StreamingHandlerConn
}

type {{$s.ServiceName}}{{.MethodName}}Server struct {
	*triple_protocol.ServerStream
}

func (g *{{$s.ServiceName}}{{.MethodName}}Server) Send(msg *proto.{{.ReturnType}}) error {
	return g.ServerStream.Send(msg)
}
{{end}}{{end}}{{end}}{{end}}
`

const ServiceInfoTpl = `{{$t := .}}{{range $s := .Services}}var {{.ServiceName}}_ServiceInfo = server.ServiceInfo{
	InterfaceName: "{{$t.ProtoPackage}}.{{.ServiceName}}",
	ServiceType:   (*{{.ServiceName}}Handler)(nil),
	Methods: []server.MethodInfo{ {{- range .Methods}}{{if .StreamsRequest}}{{if .StreamsReturn}}
		{
			Name: "{{.MethodName}}",
			Type: constant.CallBidiStream,
			StreamInitFunc: func(baseStream interface{}) interface{} {
				return &{{$s.ServiceName}}{{.MethodName}}Server{baseStream.(*triple_protocol.BidiStream)}
			},
			MethodFunc: func(ctx context.Context, args []interface{}, handler interface{}) (interface{}, error) {
				stream := args[0].({{$s.ServiceName}}_{{.MethodName}}Server)
				if err := handler.({{$s.ServiceName}}Handler).{{.MethodName}}(ctx, stream); err != nil {
					return nil, err
				}
				return nil, nil
			},
		},{{else}}
		{
			Name: "{{.MethodName}}",
			Type: constant.CallClientStream,
			StreamInitFunc: func(baseStream interface{}) interface{} {
				return &{{$s.ServiceName}}{{.MethodName}}Server{baseStream.(*triple_protocol.ClientStream)}
			},
			MethodFunc: func(ctx context.Context, args []interface{}, handler interface{}) (interface{}, error) {
				stream := args[0].({{$s.ServiceName}}_{{.MethodName}}Server)
				res, err := handler.({{$s.ServiceName}}Handler).{{.MethodName}}(ctx, stream)
				if err != nil {
					return nil, err
				}
				return triple_protocol.NewResponse(res), nil
			},
		},{{end}}{{else}}{{if .StreamsReturn}}
		{
			Name: "{{.MethodName}}",
			Type: constant.CallServerStream,
			ReqInitFunc: func() interface{} {
				return new(proto.{{.RequestType}})
			},
			StreamInitFunc: func(baseStream interface{}) interface{} {
				return &{{$s.ServiceName}}{{.MethodName}}Server{baseStream.(*triple_protocol.ServerStream)}
			},
			MethodFunc: func(ctx context.Context, args []interface{}, handler interface{}) (interface{}, error) {
				req := args[0].(*proto.{{.RequestType}})
				stream := args[1].({{$s.ServiceName}}_{{.MethodName}}Server)
				if err := handler.({{$s.ServiceName}}Handler).{{.MethodName}}(ctx, req, stream); err != nil {
					return nil, err
				}
				return nil, nil
			},
		},{{else}}
		{
			Name: "{{.MethodName}}",
			Type: constant.CallUnary,
			ReqInitFunc: func() interface{} {
				return new(proto.{{.RequestType}})
			},
			MethodFunc: func(ctx context.Context, args []interface{}, handler interface{}) (interface{}, error) {
				req := args[0].(*proto.{{.RequestType}})
				res, err := handler.({{$s.ServiceName}}Handler).{{.MethodName}}(ctx, req)
				if err != nil {
					return nil, err
				}
				return triple_protocol.NewResponse(res), nil
			},
		},{{end}}{{end}}{{end}}
	},
}{{end}}
`

const UnImplServiceTpl = `{{$t := .}}{{range $s := .Services}}// Unimplemented{{.ServiceName}}Handler returns CodeUnimplemented from all methods.
type Unimplemented{{.ServiceName}}Handler struct{}
{{range .Methods}}{{if .StreamsRequest}}{{if .StreamsReturn}}
func (Unimplemented{{$s.ServiceName}}Handler) {{.MethodName}}(context.Context, *triple_protocol.BidiStream) error {
	return triple_protocol.NewError(triple_protocol.CodeUnimplemented, errors.New("{{$t.Package}}.{{$s.ServiceName}}.{{.MethodName}} is not implemented"))
}
{{else}}
func (Unimplemented{{$s.ServiceName}}Handler) {{.MethodName}}(context.Context, *triple_protocol.ClientStream) (*triple_protocol.Response, error) {
	return nil, triple_protocol.NewError(triple_protocol.CodeUnimplemented, errors.New("{{$t.Package}}.{{$s.ServiceName}}.{{.MethodName}} is not implemented"))
}
{{end}}{{else}}{{if .StreamsReturn}}
func (Unimplemented{{$s.ServiceName}}Handler) {{.MethodName}}(context.Context, *triple_protocol.Request, *triple_protocol.ServerStream) error {
	return triple_protocol.NewError(triple_protocol.CodeUnimplemented, errors.New("{{$t.Package}}.{{$s.ServiceName}}.{{.MethodName}} is not implemented"))
}
{{else}}
func (Unimplemented{{$s.ServiceName}}Handler) {{.MethodName}}(context.Context, *proto.{{.RequestType}}) (*proto.{{.ReturnType}}, error) {
	return nil, triple_protocol.NewError(triple_protocol.CodeUnimplemented, errors.New("{{$t.Package}}.{{$s.ServiceName}}.{{.MethodName}} is not implemented"))
}
{{end}}{{end}}{{end}}{{end}}
`
