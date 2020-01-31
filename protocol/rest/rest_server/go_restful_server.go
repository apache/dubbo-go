package rest_server

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"reflect"
	"strconv"
	"strings"
	"time"
)

import (
	"github.com/apache/dubbo-go/common"
	"github.com/apache/dubbo-go/common/constant"
	"github.com/apache/dubbo-go/common/extension"
	"github.com/apache/dubbo-go/common/logger"
	"github.com/apache/dubbo-go/protocol"
	"github.com/apache/dubbo-go/protocol/invocation"
	"github.com/apache/dubbo-go/protocol/rest/rest_interface"
	"github.com/emicklei/go-restful/v3"
	perrors "github.com/pkg/errors"
)

func init() {
	extension.SetRestServer(constant.DEFAULT_REST_SERVER, GetNewGoRestfulServer)
}

type GoRestfulServer struct {
	srv       *http.Server
	container *restful.Container
}

func NewGoRestfulServer() *GoRestfulServer {
	return &GoRestfulServer{}
}

func (grs *GoRestfulServer) Start(url common.URL) {
	grs.container = restful.NewContainer()
	grs.srv = &http.Server{
		Addr:    url.Location,
		Handler: grs.container,
	}
	ln, err := net.Listen("tcp", url.Location)
	if err != nil {
		panic(perrors.New(fmt.Sprintf("Restful Server start error:%v", err)))
	}
	go grs.srv.Serve(ln)
}

func (grs *GoRestfulServer) Deploy(invoker protocol.Invoker, restMethodConfig map[string]*rest_interface.RestMethodConfig) {
	svc := common.ServiceMap.GetService(invoker.GetUrl().Protocol, strings.TrimPrefix(invoker.GetUrl().Path, "/"))
	for methodName, config := range restMethodConfig {
		// get method
		method := svc.Method()[methodName]
		types := method.ArgsType()
		f := func(req *restful.Request, resp *restful.Response) {
			var (
				err  error
				args []interface{}
			)
			args = getArgsFromRequest(req, types, config)
			result := invoker.Invoke(context.Background(), invocation.NewRPCInvocation(methodName, args, make(map[string]string, 0)))
			if result.Error() != nil {
				err = resp.WriteError(http.StatusInternalServerError, result.Error())
				if err != nil {
					logger.Errorf("[Go Restful] WriteError error:%v", err)
				}
				return
			}
			err = resp.WriteEntity(result.Result())
			if err != nil {
				logger.Error("[Go Restful] WriteEntity error:%v", err)
			}
		}
		ws := new(restful.WebService)
		ws.Path(config.Path).
			Produces(config.Produces).
			Consumes(config.Consumes).
			Route(ws.Method(config.MethodType).To(f))
		grs.container.Add(ws)
	}

}

func (grs *GoRestfulServer) UnDeploy(restMethodConfig map[string]*rest_interface.RestMethodConfig) {
	for _, config := range restMethodConfig {
		ws := new(restful.WebService)
		ws.Path(config.Path)
		err := grs.container.Remove(ws)
		if err != nil {
			logger.Warnf("[Go restful] Remove web service error:%v", err)
		}
	}
}

func (grs *GoRestfulServer) Destroy() {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := grs.srv.Shutdown(ctx); err != nil {
		logger.Errorf("Server Shutdown:", err)
	}
	logger.Errorf("Server exiting")
}

func getArgsFromRequest(req *restful.Request, types []reflect.Type, config *rest_interface.RestMethodConfig) []interface{} {
	args := make([]interface{}, len(types))
	for i, t := range types {
		args[i] = reflect.Zero(t).Interface()
	}
	var (
		err   error
		param interface{}
		i64   int64
	)
	for k, v := range config.PathParamsMap {
		if k < 0 || k >= len(types) {
			logger.Errorf("[Go restful] Path param parse error, the args:%v doesn't exist", k)
			continue
		}
		t := types[k]
		kind := t.Kind()
		if kind == reflect.Ptr {
			t = t.Elem()
		}
		if kind == reflect.Int {
			param, err = strconv.Atoi(req.PathParameter(v))
		} else if kind == reflect.Int32 {
			i64, err = strconv.ParseInt(req.PathParameter(v), 10, 32)
			if err == nil {
				param = int32(i64)
			}
		} else if kind == reflect.Int64 {
			param, err = strconv.ParseInt(req.PathParameter(v), 10, 64)
		} else if kind != reflect.String {
			logger.Warnf("[Go restful] Path param parse error, the args:%v of type isn't int or string", k)
			continue
		}
		if err != nil {
			logger.Errorf("[Go restful] Path param parse error, error is %v", err)
			continue
		}
		args[k] = param
	}
	for k, v := range config.QueryParamsMap {
		if k < 0 || k > len(types) {
			logger.Errorf("[Go restful] Query param parse error, the args:%v doesn't exist", k)
			continue
		}
		t := types[k]
		kind := t.Kind()
		if kind == reflect.Ptr {
			t = t.Elem()
		}
		if kind == reflect.Slice {
			param = req.QueryParameters(v)
		} else if kind == reflect.String {
			param = req.QueryParameter(v)
		} else if kind == reflect.Int {
			param, err = strconv.Atoi(req.QueryParameter(v))
		} else if kind == reflect.Int32 {
			i64, err = strconv.ParseInt(req.QueryParameter(v), 10, 32)
			if err == nil {
				param = int32(i64)
			}
		} else if kind == reflect.Int64 {
			param, err = strconv.ParseInt(req.QueryParameter(v), 10, 64)
		} else {
			logger.Errorf("[Go restful] Query param parse error, the args:%v of type isn't int or string or slice", k)
			continue
		}
		if err != nil {
			logger.Errorf("[Go restful] Query param parse error, error is %v", err)
			continue
		}
		args[k] = param
	}

	if config.Body > 0 && config.Body < len(types) {
		t := types[config.Body]
		err := req.ReadEntity(reflect.New(t))
		if err != nil {
			logger.Errorf("[Go restful] Read body entity error:%v", err)
		}
	}
	for k, v := range config.HeadersMap {
		param := req.HeaderParameter(v)
		if k < 0 || k >= len(types) {
			logger.Errorf("[Go restful] Header param parse error, the args:%v doesn't exist", k)
			continue
		}
		t := types[k]
		if t.Kind() == reflect.Ptr {
			t = t.Elem()
		}
		if t.Kind() == reflect.String {
			args[k] = param
		} else {
			logger.Errorf("[Go restful] Header param parse error, the args:%v of type isn't string", k)
		}
	}

	return args
}

func GetNewGoRestfulServer() rest_interface.RestServer {
	return NewGoRestfulServer()
}
