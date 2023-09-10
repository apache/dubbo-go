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

package server

import (
	"context"
	"dubbo.apache.org/dubbo-go/v3/protocol"
)

type Server struct {
	invoker protocol.Invoker
	info    *ServiceInfo

	cfg *ServerOptions
}

// ServiceInfo is meta info of a service
type ServiceInfo struct {
	InterfaceName string
	ServiceType   interface{}
	Methods       []MethodInfo
	Meta          map[string]interface{}
}

// Register assemble invoker chains like ProviderConfig.Load, init a service per call
func (s *Server) Register(handler interface{}, info *ServiceInfo, opts ...ServerOption) error {
	for _, opt := range opts {
		opt(s.cfg)
	}

	// ProviderConfig.Load
	// url
	return nil
}

type MethodInfo struct {
	Name           string
	Type           string
	ReqInitFunc    func() interface{}
	StreamInitFunc func(baseStream interface{}) interface{}
	MethodFunc     func(ctx context.Context, args []interface{}, handler interface{}) (interface{}, error)
	Meta           map[string]interface{}
}

func NewServer(opts ...ServerOption) (*Server, error) {
	newSrvOpts := defaultServerOptions()
	if err := newSrvOpts.init(opts...); err != nil {
		return nil, err
	}
	return &Server{
		cfg: newSrvOpts,
	}, nil
}
