package dubbo

import (
	"bytes"
)

import (
	"github.com/AlexStocks/getty"
	log "github.com/AlexStocks/log4go"
	"github.com/dubbogo/hessian2"
	jerrors "github.com/juju/errors"
)

////////////////////////////////////////////
// RpcClientPackageHandler
////////////////////////////////////////////

type RpcClientPackageHandler struct {
}

func NewRpcClientPackageHandler() *RpcClientPackageHandler {
	return &RpcClientPackageHandler{}
}

func (p *RpcClientPackageHandler) Read(ss getty.Session, data []byte) (interface{}, int, error) {
	pkg := &DubboPackage{}

	buf := bytes.NewBuffer(data)
	err := pkg.Unmarshal(buf, hessian.Response)
	if err != nil {
		return nil, 0, jerrors.Trace(err)
	}

	return pkg, len(data), nil
}

func (p *RpcClientPackageHandler) Write(ss getty.Session, pkg interface{}) error {
	req, ok := pkg.(*DubboPackage)
	if !ok {
		log.Error("illegal pkg:%+v\n", pkg)
		return jerrors.New("invalid rpc request")
	}

	buf, err := req.Marshal()
	if err != nil {
		log.Warn("binary.Write(req{%#v}) = err{%#v}", req, jerrors.ErrorStack(err))
		return jerrors.Trace(err)
	}

	return jerrors.Trace(ss.WriteBytes(buf.Bytes()))
}
