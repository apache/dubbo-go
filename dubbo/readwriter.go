package rpc

import (
	"bytes"
	"reflect"
)

import (
	"github.com/AlexStocks/getty"
	log "github.com/AlexStocks/log4go"
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
	pkg := &GettyPackage{
		B: NewGettyRPCResponse(),
	}

	buf := bytes.NewBuffer(data)
	length, err := pkg.Unmarshal(buf)
	if err != nil {
		if err == ErrNotEnoughStream {
			return nil, 0, nil
		}
		return nil, 0, jerrors.Trace(err)
	}

	resp := &GettyRPCResponsePackage{
		H:      pkg.H,
		header: pkg.B.GetHeader().(GettyRPCResponseHeader),
	}
	if pkg.H.Command != gettyCmdHbResponse {
		resp.body = pkg.B.GetBody()
	}
	return resp, length, nil
}

func (p *RpcClientPackageHandler) Write(ss getty.Session, pkg interface{}) error {
	req, ok := pkg.(GettyPackage)
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
