package jsonrpc

import (
	"bytes"
	"context"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"reflect"
	"strings"
	"sync"
	"unicode"
	"unicode/utf8"
)

import (
	log "github.com/AlexStocks/log4go"
	jerrors "github.com/juju/errors"
)

var (
	// A value sent as a placeholder for the server's response value when the server
	// receives an invalid request. It is never decoded by the client since the Response
	// contains an error when it is used.
	invalidRequest = struct{}{}

	// Precompute the reflect type for error. Can't use error directly
	// because Typeof takes an empty interface value. This is annoying.
	typeOfError = reflect.TypeOf((*error)(nil)).Elem()
)

type serviceMethod struct {
	method    reflect.Method // receiver method
	ctxType   reflect.Type   // type of the request context
	argsType  reflect.Type   // type of the request argument
	replyType reflect.Type   // type of the response argument
}

func (m *serviceMethod) suiteContext(ctx context.Context) reflect.Value {
	if contextv := reflect.ValueOf(ctx); contextv.IsValid() {
		return contextv
	}
	return reflect.Zero(m.ctxType)
}

type svc struct {
	name     string                    // name of service
	rcvr     reflect.Value             // receiver of methods for the service
	rcvrType reflect.Type              // type of the receiver
	methods  map[string]*serviceMethod // registered methods, function name -> reflect.function
}

type serviceMap struct {
	mutex      sync.Mutex      // protects the serviceMap
	serviceMap map[string]*svc // service name -> service
}

func initServer() *serviceMap {
	return &serviceMap{
		serviceMap: make(map[string]*svc),
	}
}

// isExported returns true of a string is an exported (upper case) name.
func isExported(name string) bool {
	rune, _ := utf8.DecodeRuneInString(name)
	return unicode.IsUpper(rune)
}

// isExportedOrBuiltin returns true if a type is exported or a builtin.
func isExportedOrBuiltin(t reflect.Type) bool {
	for t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	// PkgPath will be non-empty even for an exported type,
	// so we need to check the type name as well.
	return isExported(t.Name()) || t.PkgPath() == ""
}

func suiteMethod(method reflect.Method) *serviceMethod {
	mtype := method.Type
	mname := method.Name

	// Method must be exported.
	if method.PkgPath != "" {
		return nil
	}

	var replyType, argType, ctxType reflect.Type
	switch mtype.NumIn() {
	case 3:
		argType = mtype.In(1)
		replyType = mtype.In(2)
	case 4:
		ctxType = mtype.In(1)
		argType = mtype.In(2)
		replyType = mtype.In(3)
	default:
		log.Error("method %s of mtype %v has wrong number of in parameters %d; needs exactly 3/4",
			mname, mtype, mtype.NumIn())
		return nil
	}
	// First arg need not be a pointer.
	if !isExportedOrBuiltin(argType) {
		log.Error("argument type of method %q is not exported %v", mname, argType)
		return nil
	}
	// Second arg must be a pointer.
	if replyType.Kind() != reflect.Ptr {
		log.Error("reply type of method %q is not a pointer %v", mname, replyType)
		return nil
	}
	// Reply type must be exported.
	if !isExportedOrBuiltin(replyType) {
		log.Error("reply type of method %s not exported{%v}", mname, replyType)
		return nil
	}
	// Method needs one out.
	if mtype.NumOut() != 1 {
		log.Error("method %q has %d out parameters; needs exactly 1", mname, mtype.NumOut())
		return nil
	}
	// The return type of the method must be error.
	if returnType := mtype.Out(0); returnType != typeOfError {
		log.Error("return type %s of method %q is not error", returnType, mname)
		return nil
	}

	return &serviceMethod{method: method, argsType: argType, replyType: replyType, ctxType: ctxType}
}

func (server *serviceMap) register(rcvr Handler) (string, error) {
	server.mutex.Lock()
	defer server.mutex.Unlock()
	if server.serviceMap == nil {
		server.serviceMap = make(map[string]*svc)
	}

	s := new(svc)
	s.rcvrType = reflect.TypeOf(rcvr)
	s.rcvr = reflect.ValueOf(rcvr)
	sname := reflect.Indirect(s.rcvr).Type().Name()
	if sname == "" {
		s := "no service name for type " + s.rcvrType.String()
		log.Error(s)
		return "", jerrors.New(s)
	}
	if !isExported(sname) {
		s := "type " + sname + " is not exported"
		log.Error(s)
		return "", jerrors.New(s)
	}

	sname = rcvr.Service()
	if _, dup := server.serviceMap[sname]; dup {
		return "", jerrors.New("service already defined: " + sname)
	}
	s.name = sname
	s.methods = make(map[string]*serviceMethod)

	// Install the methods
	methods := ""
	num := s.rcvrType.NumMethod()
	for m := 0; m < num; m++ {
		method := s.rcvrType.Method(m)
		if mt := suiteMethod(method); mt != nil {
			s.methods[method.Name] = mt
			methods += method.Name + ","
		}
	}

	if len(s.methods) == 0 {
		s := "type " + sname + " has no exported methods of suitable type"
		log.Error(s)
		return "", jerrors.New(s)
	}
	server.serviceMap[s.name] = s

	return strings.TrimSuffix(methods, ","), nil
}

func (server *serviceMap) serveRequest(ctx context.Context,
	header map[string]string, body []byte, conn net.Conn) error {

	// read request header
	codec := newServerCodec()
	err := codec.ReadHeader(header, body)
	if err != nil {
		if err == io.EOF || err == io.ErrUnexpectedEOF {
			return jerrors.Trace(err)
		}

		return jerrors.New("server cannot decode request: " + err.Error())
	}
	serviceName := header["Path"]
	methodName := codec.req.Method
	if len(serviceName) == 0 || len(methodName) == 0 {
		codec.ReadBody(nil)
		return jerrors.New("service/method request ill-formed: " + serviceName + "/" + methodName)
	}

	// get method
	server.mutex.Lock()
	svc := server.serviceMap[serviceName]
	server.mutex.Unlock()
	if svc == nil {
		codec.ReadBody(nil)
		return jerrors.New("cannot find svc " + serviceName)
	}
	mtype := svc.methods[methodName]
	if mtype == nil {
		codec.ReadBody(nil)
		return jerrors.New("cannot find method " + methodName + " of svc " + serviceName)
	}

	// get args
	var argv reflect.Value
	argIsValue := false
	if mtype.argsType.Kind() == reflect.Ptr {
		argv = reflect.New(mtype.argsType.Elem())
	} else {
		argv = reflect.New(mtype.argsType)
		argIsValue = true
	}
	// argv guaranteed to be a pointer now.
	if err = codec.ReadBody(argv.Interface()); err != nil {
		return jerrors.Trace(err)
	}
	if argIsValue {
		argv = argv.Elem()
	}

	replyv := reflect.New(mtype.replyType.Elem())

	//  call service.method(args)
	var errMsg string
	returnValues := mtype.method.Func.Call([]reflect.Value{
		svc.rcvr,
		mtype.suiteContext(ctx),
		reflect.ValueOf(argv.Interface()),
		reflect.ValueOf(replyv.Interface()),
	})
	// The return value for the method is an error.
	if retErr := returnValues[0].Interface(); retErr != nil {
		errMsg = retErr.(error).Error()
	}

	// write response
	code := 200
	rspReply := replyv.Interface()
	if len(errMsg) != 0 {
		code = 500
		rspReply = invalidRequest
	}
	rspStream, err := codec.Write(errMsg, rspReply)
	if err != nil {
		return jerrors.Trace(err)
	}
	rsp := &http.Response{
		StatusCode:    code,
		ProtoMajor:    1,
		ProtoMinor:    1,
		Header:        make(http.Header),
		ContentLength: int64(len(rspStream)),
		Body:          ioutil.NopCloser(bytes.NewReader(rspStream)),
	}
	delete(header, "Content-Type")
	delete(header, "Content-Length")
	delete(header, "Timeout")
	for k, v := range header {
		rsp.Header.Set(k, v)
	}

	rspBuf := bytes.NewBuffer(make([]byte, DefaultHTTPRspBufferSize))
	rspBuf.Reset()
	if err = rsp.Write(rspBuf); err != nil {
		log.Warn("rsp.Write(rsp:%#v) = error:%s", rsp, err)
		return nil
	}
	if _, err = rspBuf.WriteTo(conn); err != nil {
		log.Warn("rspBuf.WriteTo(conn:%#v) = error:%s", conn, err)
	}
	return nil
}
