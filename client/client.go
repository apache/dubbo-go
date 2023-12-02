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

// Package client provides APIs for starting RPC calls.
package client

import (
	"context"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/protocol"
	invocation_impl "dubbo.apache.org/dubbo-go/v3/protocol/invocation"
)

// ConsumerConfig
type Client struct {
	cliOpts *ClientOptions
}

type ClientInfo struct {
	InterfaceName        string
	MethodNames          []string
	ConnectionInjectFunc func(dubboCliRaw interface{}, conn *Connection)
	Meta                 map[string]interface{}
}

type ClientDefinition struct {
	Svc  interface{}
	Info *ClientInfo
}

// InterfaceName/group/version /ReferenceConfig
type Connection struct {
	refOpts *ReferenceOptions
}

func (conn *Connection) call(ctx context.Context, reqs []interface{}, resp interface{}, methodName, callType string, opts ...CallOption) (protocol.Result, error) {
	options := newDefaultCallOptions()
	for _, opt := range opts {
		opt(options)
	}
	inv, err := generateInvocation(methodName, reqs, resp, callType, options)
	if err != nil {
		return nil, err
	}
	return conn.refOpts.invoker.Invoke(ctx, inv), nil
}

func (conn *Connection) CallUnary(ctx context.Context, reqs []interface{}, resp interface{}, methodName string, opts ...CallOption) error {
	res, err := conn.call(ctx, reqs, resp, methodName, constant.CallUnary, opts...)
	if err != nil {
		return err
	}
	return res.Error()
}

func (conn *Connection) CallClientStream(ctx context.Context, methodName string, opts ...CallOption) (interface{}, error) {
	res, err := conn.call(ctx, nil, nil, methodName, constant.CallClientStream, opts...)
	if err != nil {
		return nil, err
	}
	return res.Result(), res.Error()
}

func (conn *Connection) CallServerStream(ctx context.Context, req interface{}, methodName string, opts ...CallOption) (interface{}, error) {
	res, err := conn.call(ctx, []interface{}{req}, nil, methodName, constant.CallServerStream, opts...)
	if err != nil {
		return nil, err
	}
	return res.Result(), res.Error()
}

func (conn *Connection) CallBidiStream(ctx context.Context, methodName string, opts ...CallOption) (interface{}, error) {
	res, err := conn.call(ctx, nil, nil, methodName, constant.CallBidiStream, opts...)
	if err != nil {
		return nil, err
	}
	return res.Result(), res.Error()
}

func (cli *Client) Dial(interfaceName string, opts ...ReferenceOption) (*Connection, error) {
	return cli.dial(interfaceName, nil, opts...)
}

func (cli *Client) DialWithInfo(interfaceName string, info *ClientInfo, opts ...ReferenceOption) (*Connection, error) {
	return cli.dial(interfaceName, info, opts...)
}

func (cli *Client) dial(interfaceName string, info *ClientInfo, opts ...ReferenceOption) (*Connection, error) {
	newRefOpts := defaultReferenceOptions()
	finalOpts := []ReferenceOption{
		setReference(cli.cliOpts.overallReference),
		setApplicationCompat(cli.cliOpts.applicationCompat),
		setRegistriesCompat(cli.cliOpts.registriesCompat),
		setConsumer(cli.cliOpts.Consumer),
		// this config must be set after Reference initialized
		setInterfaceName(interfaceName),
	}
	finalOpts = append(finalOpts, opts...)
	if err := newRefOpts.init(finalOpts...); err != nil {
		return nil, err
	}
	newRefOpts.ReferWithInfo(info)

	return &Connection{refOpts: newRefOpts}, nil
}
func generateInvocation(methodName string, reqs []interface{}, resp interface{}, callType string, opts *CallOptions) (protocol.Invocation, error) {
	var paramsRawVals []interface{}
	for _, req := range reqs {
		paramsRawVals = append(paramsRawVals, req)
	}
	if resp != nil {
		paramsRawVals = append(paramsRawVals, resp)
	}
	inv := invocation_impl.NewRPCInvocationWithOptions(
		invocation_impl.WithMethodName(methodName),
		invocation_impl.WithAttachment(constant.TimeoutKey, opts.RequestTimeout),
		invocation_impl.WithAttachment(constant.RetriesKey, opts.Retries),
		invocation_impl.WithArguments(reqs),
		invocation_impl.WithReply(resp),
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
