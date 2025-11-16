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
	"errors"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/metadata"
	"dubbo.apache.org/dubbo-go/v3/protocol/base"
	"dubbo.apache.org/dubbo-go/v3/protocol/invocation"
	"dubbo.apache.org/dubbo-go/v3/protocol/result"
)

// ConsumerConfig
type Client struct {
	cliOpts *ClientOptions
}

type ClientInfo struct {
	InterfaceName        string
	MethodNames          []string
	ConnectionInjectFunc func(dubboCliRaw any, conn *Connection)
	Meta                 map[string]any
}

type ClientDefinition struct {
	Svc  any
	Info *ClientInfo
	Conn *Connection
}

func (d *ClientDefinition) SetConnection(conn *Connection) {
	d.Conn = conn
}

func (d *ClientDefinition) GetConnection() (*Connection, error) {
	if d.Conn == nil {
		return nil, errors.New("you need dubbo.load() first")
	}
	return d.Conn, nil
}

// InterfaceName/group/version /ReferenceConfig
// TODO: In the Connection structure, we are only using the invoker in the refOpts.
// Considering simplifying the Connection.
// Make the structure of Connection more in line with human logic.
type Connection struct {
	refOpts *ReferenceOptions
}

func (conn *Connection) call(ctx context.Context, reqs []any, resp any, methodName, callType string, opts ...CallOption) (result.Result, error) {
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

func (conn *Connection) CallUnary(ctx context.Context, reqs []any, resp any, methodName string, opts ...CallOption) error {
	res, err := conn.call(ctx, reqs, resp, methodName, constant.CallUnary, opts...)
	if err != nil {
		return err
	}
	return res.Error()
}

func (conn *Connection) CallClientStream(ctx context.Context, methodName string, opts ...CallOption) (any, error) {
	res, err := conn.call(ctx, nil, nil, methodName, constant.CallClientStream, opts...)
	if err != nil {
		return nil, err
	}
	return res.Result(), res.Error()
}

func (conn *Connection) CallServerStream(ctx context.Context, req any, methodName string, opts ...CallOption) (any, error) {
	res, err := conn.call(ctx, []any{req}, nil, methodName, constant.CallServerStream, opts...)
	if err != nil {
		return nil, err
	}
	return res.Result(), res.Error()
}

func (conn *Connection) CallBidiStream(ctx context.Context, methodName string, opts ...CallOption) (any, error) {
	res, err := conn.call(ctx, nil, nil, methodName, constant.CallBidiStream, opts...)
	if err != nil {
		return nil, err
	}
	return res.Result(), res.Error()
}

func (cli *Client) NewService(service any, opts ...ReferenceOption) (*Connection, error) {
	if service == nil {
		return nil, errors.New("service must not be nil")
	}

	interfaceName := common.GetReference(service)

	finalOpts := []ReferenceOption{
		WithIDL(constant.NONIDL),
		// default msgpack serialization
		WithSerialization(constant.MsgpackSerialization),
	}
	finalOpts = append(finalOpts, opts...)

	return cli.DialWithService(interfaceName, service, finalOpts...)
}

func (cli *Client) Dial(interfaceName string, opts ...ReferenceOption) (*Connection, error) {
	return cli.dial(interfaceName, nil, nil, opts...)
}

func (cli *Client) DialWithService(interfaceName string, service any, opts ...ReferenceOption) (*Connection, error) {
	return cli.dial(interfaceName, nil, service, opts...)
}

func (cli *Client) DialWithInfo(interfaceName string, info *ClientInfo, opts ...ReferenceOption) (*Connection, error) {
	return cli.dial(interfaceName, info, nil, opts...)
}

func (cli *Client) DialWithDefinition(interfaceName string, definition *ClientDefinition, opts ...ReferenceOption) (*Connection, error) {
	// TODO(finalt) Temporarily solve the config_center configuration does not work
	refName := common.GetReference(definition.Svc)
	if refConfig, ok := cli.cliOpts.Consumer.References[refName]; ok {
		ref := cli.cliOpts.overallReference.Clone()
		for _, opt := range refConfig.GetOptions() {
			opt(ref)
		}
		opts = append(opts, setReference(ref))
	}

	return cli.dial(interfaceName, definition.Info, nil, opts...)
}

func (cli *Client) dial(interfaceName string, info *ClientInfo, srv any, opts ...ReferenceOption) (*Connection, error) {
	if err := metadata.InitRegistryMetadataReport(cli.cliOpts.Registries); err != nil {
		return nil, err
	}
	newRefOpts := defaultReferenceOptions()
	finalOpts := []ReferenceOption{
		setReference(cli.cliOpts.overallReference),
		setApplicationCompat(cli.cliOpts.applicationCompat),
		setRegistriesCompat(cli.cliOpts.registriesCompat),
		setRegistries(cli.cliOpts.Registries),
		setConsumer(cli.cliOpts.Consumer),
		setMetrics(cli.cliOpts.Metrics),
		setOtel(cli.cliOpts.Otel),
		setTLS(cli.cliOpts.TLS),
		// this config must be set after Reference initialized
		setInterfaceName(interfaceName),
	}

	finalOpts = append(finalOpts, opts...)
	if err := newRefOpts.init(finalOpts...); err != nil {
		return nil, err
	}

	if info != nil {
		newRefOpts.ReferWithInfo(info)
	} else if srv != nil {
		newRefOpts.ReferWithService(srv)
	} else {
		newRefOpts.Refer()
	}

	return &Connection{refOpts: newRefOpts}, nil
}

func generateInvocation(methodName string, reqs []any, resp any, callType string, opts *CallOptions) (base.Invocation, error) {
	var paramsRawVals []any

	paramsRawVals = append(paramsRawVals, reqs...)

	if resp != nil {
		paramsRawVals = append(paramsRawVals, resp)
	}
	inv := invocation.NewRPCInvocationWithOptions(
		invocation.WithMethodName(methodName),
		invocation.WithAttachment(constant.TimeoutKey, opts.RequestTimeout),
		invocation.WithAttachment(constant.RetriesKey, opts.Retries),
		invocation.WithArguments(reqs),
		invocation.WithReply(resp),
		invocation.WithParameterRawValues(paramsRawVals),
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
