package client

import (
	"context"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/config"
	"dubbo.apache.org/dubbo-go/v3/protocol"
	invocation_impl "dubbo.apache.org/dubbo-go/v3/protocol/invocation"
	"github.com/pkg/errors"
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
	rootCfg := config.NewRootConfigBuilder().Build()
	rootCfg.Init()
	newRefCfg := &ReferenceConfig{}
	for _, opt := range opts {
		opt(newRefCfg)
	}
	if err := newRefCfg.Init(rootCfg); err != nil {
		return nil, err
	}
	return &Client{
		cfg: newRefCfg,
	}, nil
}

// NewClientWithReferenceBase is used by /config/consumer_config, it assumes that refCfg has been init
func NewClientWithReferenceBase(refCfg *ReferenceConfig, opts ...ReferenceOption) (*Client, error) {
	if refCfg == nil {
		return nil, errors.New("ReferenceConfig is nil")
	}
	for _, opt := range opts {
		opt(refCfg)
	}
	return &Client{
		cfg: refCfg,
	}, nil
}

func NewConsumerConfig(opts ...ConsumerOption) (*ConsumerConfig, error) {
	rootCfg := config.NewRootConfigBuilder().Build()
	rootCfg.Init()
	newConCfg := &ConsumerConfig{}
	for _, opt := range opts {
		opt(newConCfg)
	}
	if err := newConCfg.Init(rootCfg); err != nil {
		return nil, err
	}
	rootCfg.Consumer = newConCfg
	return newConCfg, nil
}

// todo: create NewClientWithRootBase
func NewClientWithConsumerBase(conCfg *ConsumerConfig, opts ...ReferenceOption) (*Client, error) {
	newRefCfg := &ReferenceConfig{}
	for _, opt := range opts {
		opt(newRefCfg)
	}
	if err := newRefCfg.Init(conCfg.rootConfig); err != nil {
		return nil, err
	}
	return &Client{
		cfg: newRefCfg,
	}, nil
}

func NewMethodConfig(opts ...MethodOption) (*MethodConfig, error) {
	newMethodCfg := &MethodConfig{}
	for _, opt := range opts {
		opt(newMethodCfg)
	}
	if err := newMethodCfg.Init(); err != nil {
		return nil, err
	}
	return newMethodCfg, nil
}
