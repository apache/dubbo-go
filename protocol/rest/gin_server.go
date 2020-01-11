package rest

import (
	"context"
	"github.com/apache/dubbo-go/common"
	"github.com/apache/dubbo-go/common/logger"
	"github.com/gin-gonic/gin"
	"net/http"
	"time"
)

type GinServer struct {
	engine *gin.Engine
	srv    *http.Server
}

func (gs *GinServer) Start(url common.URL) {
	srv := &http.Server{
		Addr:    ":8080",
		Handler: gs.engine,
	}
	srv.ListenAndServe()
}

func (gs *GinServer) Deploy(service interface{}, config interface{}) {
	//TODO gin http 部署接口
	// handler中处理：
	// http server收到http request，转交给server；
	// server根据RestService metadata找到Service，并将请求反序列化，构造好Service调用的上下文；
	// RequestMapper这时候将不会处理什么事情，而是直接转交给Service的实现。
	//
	// gs.engine.Handle(method, relativePath, handler)
}

func (gs *GinServer) Undeploy(url common.URL) {
	//TODO gin http 删除接口
}

func (gs *GinServer) Destroy() {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := gs.srv.Shutdown(ctx); err != nil {
		logger.Errorf("Server Shutdown:", err)
	}
	logger.Errorf("Server exiting")
}
