package dubbo

import (
	"bytes"
	"reflect"
)

import (
	"github.com/AlexStocks/getty"
	log "github.com/AlexStocks/log4go"
	jerrors "github.com/juju/errors"
)
import (
	"github.com/dubbo/go-for-apache-dubbo/config"
)

////////////////////////////////////////////
// RpcClientPackageHandler
////////////////////////////////////////////

type RpcClientPackageHandler struct {
	client *Client
}

func NewRpcClientPackageHandler(client *Client) *RpcClientPackageHandler {
	return &RpcClientPackageHandler{client: client}
}

func (p *RpcClientPackageHandler) Read(ss getty.Session, data []byte) (interface{}, int, error) {
	pkg := &DubboPackage{
		Body: p.client.pendingResponses[SequenceType(ss.GetAttribute("seq").(uint64))].reply,
	}

	buf := bytes.NewBuffer(data)
	err := pkg.Unmarshal(buf)
	if err != nil {
		pkg.Err = jerrors.Trace(err) // client will get this err
		return pkg, len(data), nil
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

////////////////////////////////////////////
// RpcServerPackageHandler
////////////////////////////////////////////

type RpcServerPackageHandler struct {
}

func NewRpcServerPackageHandler() *RpcServerPackageHandler {
	return &RpcServerPackageHandler{}
}

func (p *RpcServerPackageHandler) Read(ss getty.Session, data []byte) (interface{}, int, error) {
	pkg := &DubboPackage{
		Body: make([]interface{}, 7),
	}

	buf := bytes.NewBuffer(data)
	err := pkg.Unmarshal(buf)
	if err != nil {
		return nil, 0, jerrors.Trace(err)
	}
	// convert params of request
	req := pkg.Body.([]interface{}) // length of body should be 7
	if len(req) > 0 {
		var dubboVersion, argsTypes string
		var args []interface{}
		var attachments map[interface{}]interface{}
		if req[0] != nil {
			dubboVersion = req[0].(string)
		}
		if req[1] != nil {
			pkg.Service.Target = req[1].(string)
		}
		if req[2] != nil {
			pkg.Service.Version = req[2].(string)
		}
		if req[3] != nil {
			pkg.Service.Method = req[3].(string)
		}
		if req[4] != nil {
			argsTypes = req[4].(string)
		}
		if req[5] != nil {
			args = req[5].([]interface{})
		}
		if req[6] != nil {
			attachments = req[6].(map[interface{}]interface{})
		}
		pkg.Body = map[string]interface{}{
			"dubboVersion": dubboVersion,
			"argsTypes":    argsTypes,
			"args":         args,
			"service":      config.ServiceMap.GetService(DUBBO, pkg.Service.Target),
			"attachments":  attachments,
		}
	}

	return pkg, len(data), nil
}

func (p *RpcServerPackageHandler) Write(ss getty.Session, pkg interface{}) error {
	res, ok := pkg.(*DubboPackage)
	if !ok {
		log.Error("illegal pkg:%+v\n, it is %+v", pkg, reflect.TypeOf(pkg))
		return jerrors.New("invalid rpc response")
	}

	buf, err := res.Marshal()
	if err != nil {
		log.Warn("binary.Write(res{%#v}) = err{%#v}", res, jerrors.ErrorStack(err))
		return jerrors.Trace(err)
	}

	return jerrors.Trace(ss.WriteBytes(buf.Bytes()))
}
