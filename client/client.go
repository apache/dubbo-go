package client

import (
	"context"
)

import (
	"github.com/pkg/errors"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/protocol"
	invocation_impl "dubbo.apache.org/dubbo-go/v3/protocol/invocation"
)

type Client struct {
	invoker protocol.Invoker
	info    *ClientInfo
	cfg     *ReferenceConfig
}

type ClientInfo struct {
	InterfaceName    string
	MethodNames      []string
	ClientInjectFunc func(dubboCliRaw interface{}, cli *Client)
	Meta             map[string]interface{}
}

func (cli *Client) call(ctx context.Context, paramsRawVals []interface{}, interfaceName, methodName, callType string, opts ...CallOption) (protocol.Result, error) {
	// get a default CallOptions
	// apply CallOption
	options := newDefaultCallOptions()
	for _, opt := range opts {
		opt(options)
	}

	inv, err := generateInvocation(methodName, paramsRawVals, callType, options)
	if err != nil {
		return nil, err
	}
	// todo: move timeout into context or invocation
	return cli.invoker.Invoke(ctx, inv), nil

}

func (cli *Client) CallUnary(ctx context.Context, req, resp interface{}, interfaceName, methodName string, opts ...CallOption) error {
	res, err := cli.call(ctx, []interface{}{req, resp}, interfaceName, methodName, constant.CallUnary, opts...)
	if err != nil {
		return err
	}
	return res.Error()
}

func (cli *Client) CallClientStream(ctx context.Context, interfaceName, methodName string, opts ...CallOption) (interface{}, error) {
	res, err := cli.call(ctx, nil, interfaceName, methodName, constant.CallClientStream, opts...)
	if err != nil {
		return nil, err
	}
	return res.Result(), res.Error()
}

func (cli *Client) CallServerStream(ctx context.Context, req interface{}, interfaceName, methodName string, opts ...CallOption) (interface{}, error) {
	res, err := cli.call(ctx, []interface{}{req}, interfaceName, methodName, constant.CallServerStream, opts...)
	if err != nil {
		return nil, err
	}
	return res.Result(), res.Error()
}

func (cli *Client) CallBidiStream(ctx context.Context, interfaceName, methodName string, opts ...CallOption) (interface{}, error) {
	res, err := cli.call(ctx, nil, interfaceName, methodName, constant.CallBidiStream, opts...)
	if err != nil {
		return nil, err
	}
	return res.Result(), res.Error()
}

func (cli *Client) Init(info *ClientInfo) error {
	if info == nil {
		return errors.New("ClientInfo is nil")
	}
	cli.cfg.ReferWithInfo(info)
	cli.invoker = cli.cfg.invoker

	return nil
}

func generateInvocation(methodName string, paramsRawVals []interface{}, callType string, opts *CallOptions) (protocol.Invocation, error) {
	inv := invocation_impl.NewRPCInvocationWithOptions(
		invocation_impl.WithMethodName(methodName),
		// todo: process opts
		invocation_impl.WithParameterRawValues(paramsRawVals),
	)
	inv.SetAttribute(constant.CallTypeKey, callType)

	return inv, nil
}

func NewClient(opts ...ReferenceOption) (*Client, error) {
	// get default RootConfigs
	//rootCfg := config.NewRootConfigBuilder().Build()
	//rootCfg.Init()
	// todo: create a default ReferenceConfig
	newRefCfg := &ReferenceConfig{}
	if err := newRefCfg.Init(opts...); err != nil {
		return nil, err
	}
	return &Client{
		cfg: newRefCfg,
	}, nil
}
