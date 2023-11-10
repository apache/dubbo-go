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

	cliOpts *ClientOptions
	refOpts map[string]*ReferenceOptions
}

type ClientInfo struct {
	InterfaceName    string
	MethodNames      []string
	ClientInjectFunc func(dubboCliRaw interface{}, cli *Client)
	Meta             map[string]interface{}
}

func (cli *Client) call(ctx context.Context, invoker protocol.Invoker, paramsRawVals []interface{}, interfaceName, methodName, callType string, opts ...CallOption) (protocol.Result, error) {
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
	return invoker.Invoke(ctx, inv), nil

}

func (cli *Client) CallUnary(ctx context.Context, invoker protocol.Invoker, req, resp interface{}, interfaceName, methodName string, opts ...CallOption) error {
	res, err := cli.call(ctx, invoker, []interface{}{req, resp}, interfaceName, methodName, constant.CallUnary, opts...)
	if err != nil {
		return err
	}
	return res.Error()
}

func (cli *Client) CallClientStream(ctx context.Context, invoker protocol.Invoker, interfaceName, methodName string, opts ...CallOption) (interface{}, error) {
	res, err := cli.call(ctx, invoker, nil, interfaceName, methodName, constant.CallClientStream, opts...)
	if err != nil {
		return nil, err
	}
	return res.Result(), res.Error()
}

func (cli *Client) CallServerStream(ctx context.Context, invoker protocol.Invoker, req interface{}, interfaceName, methodName string, opts ...CallOption) (interface{}, error) {
	res, err := cli.call(ctx, invoker, []interface{}{req}, interfaceName, methodName, constant.CallServerStream, opts...)
	if err != nil {
		return nil, err
	}
	return res.Result(), res.Error()
}

func (cli *Client) CallBidiStream(ctx context.Context, invoker protocol.Invoker, interfaceName, methodName string, opts ...CallOption) (interface{}, error) {
	res, err := cli.call(ctx, invoker, nil, interfaceName, methodName, constant.CallBidiStream, opts...)
	if err != nil {
		return nil, err
	}
	return res.Result(), res.Error()
}

func (cli *Client) Init(info *ClientInfo, opts ...ReferenceOption) (protocol.Invoker, error) {
	if info == nil {
		return nil, errors.New("ClientInfo is nil")
	}

	newRefOptions := defaultReferenceOptions()
	err := newRefOptions.init(cli, opts...)
	if err != nil {
		return nil, err
	}

	cli.refOpts[newRefOptions.Reference.ServiceKey()] = newRefOptions

	newRefOptions.ReferWithInfo(info)

	return newRefOptions.invoker, nil
}

func generateInvocation(methodName string, paramsRawVals []interface{}, callType string, opts *CallOptions) (protocol.Invocation, error) {
	inv := invocation_impl.NewRPCInvocationWithOptions(
		invocation_impl.WithMethodName(methodName),
		invocation_impl.WithAttachment(constant.TimeoutKey, opts.RequestTimeout),
		invocation_impl.WithAttachment(constant.RetriesKey, opts.Retries),
		invocation_impl.WithParameterRawValues(paramsRawVals),
	)
	inv.SetAttribute(constant.CallTypeKey, callType)

	return inv, nil
}

func NewClient(opts ...ClientOption) (*Client, error) {
	newCliOpts := defaultClientOptions()
	if err := newCliOpts.init(opts...); err != nil {
		return nil, err
	}
	return &Client{
		cliOpts: newCliOpts,
	}, nil
}
