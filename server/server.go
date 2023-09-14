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
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/global"
	"dubbo.apache.org/dubbo-go/v3/protocol"
	"fmt"
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

type infoInvoker struct {
	url       *common.URL
	base      *protocol.BaseInvoker
	info      *ServiceInfo
	svc       common.RPCService
	methodMap map[string]*MethodInfo
}

func (ii *infoInvoker) init() {
	url := ii.base.GetURL()
	if url.SubURL != nil {
		url = url.SubURL
	}
	ii.url = url
	methodMap := make(map[string]*MethodInfo)
	for i := range ii.info.Methods {
		methodMap[ii.info.Methods[i].Name] = &ii.info.Methods[i]
	}
	ii.methodMap = methodMap
}

func (ii *infoInvoker) GetURL() *common.URL {
	return ii.base.GetURL()
}

func (ii *infoInvoker) IsAvailable() bool {
	return ii.base.IsAvailable()
}

func (ii *infoInvoker) Destroy() {
	ii.base.Destroy()
}

func (ii *infoInvoker) Invoke(ctx context.Context, invocation protocol.Invocation) protocol.Result {
	name := invocation.MethodName()
	args := invocation.Arguments()
	result := new(protocol.RPCResult)
	if method, ok := ii.methodMap[name]; ok {
		res, err := method.MethodFunc(ctx, args, ii.svc)
		result.SetResult(res)
		result.SetError(err)
		return result
	}
	result.SetError(fmt.Errorf("no match method for %s", name))

	return result
}

func newInfoInvoker(url *common.URL, info *ServiceInfo, svc common.RPCService) protocol.Invoker {
	invoker := &infoInvoker{
		base: protocol.NewBaseInvoker(url),
		info: info,
		svc:  svc,
	}
	invoker.init()
	return invoker
}

// Register assemble invoker chains like ProviderConfig.Load, init a service per call
func (s *Server) Register(handler interface{}, info *ServiceInfo, opts ...ServiceOption) error {
	// todo(DMwangnima): record the registered service
	newSvcOpts := defaultServiceOptions()
	// todo(DMwangnima): use a more elegant way to initialize
	opts = append(opts,
		WithApplicationConfig(
			global.WithApplication_Name(s.cfg.Application.Name),
			global.WithApplication_Organization(s.cfg.Application.Organization),
			global.WithApplication_Module(s.cfg.Application.Module),
			global.WithApplication_Version(s.cfg.Application.Version),
			global.WithApplication_Owner(s.cfg.Application.Owner),
			global.WithApplication_Environment(s.cfg.Application.Environment),
		),
		WithProviderConfig(
			global.WithProvider_RegistryIDs(s.cfg.Provider.RegistryIDs),
			global.WithProvider_ProtocolIDs(s.cfg.Provider.ProtocolIDs),
			global.WithProvider_TracingKey(s.cfg.Provider.TracingKey),
		),
	)
	for key, pro := range s.cfg.Protocols {
		opts = append(opts,
			WithProtocolConfig(key,
				global.WithProtocol_Name(pro.Name),
				global.WithProtocol_Ip(pro.Ip),
				global.WithProtocol_Port(pro.Port),
				global.WithProtocol_Params(pro.Params),
				global.WithProtocol_MaxServerSendMsgSize(pro.MaxServerSendMsgSize),
				global.WithProtocol_MaxServerRecvMsgSize(pro.MaxServerRecvMsgSize),
			),
		)
	}
	for key, reg := range s.cfg.Registries {
		opts = append(opts,
			WithRegistryConfig(key,
				global.WithRegistry_Protocol(reg.Protocol),
				global.WithRegistry_Timeout(reg.Timeout),
				global.WithRegistry_Group(reg.Group),
				global.WithRegistry_Namespace(reg.Namespace),
				global.WithRegistry_TTL(reg.TTL),
				global.WithRegistry_Address(reg.Address),
				global.WithRegistry_Username(reg.Username),
				global.WithRegistry_Password(reg.Password),
				global.WithRegistry_Simplified(reg.Simplified),
				global.WithRegistry_Preferred(reg.Preferred),
				global.WithRegistry_Zone(reg.Zone),
				global.WithRegistry_Weight(reg.Weight),
				global.WithRegistry_Params(reg.Params),
				global.WithRegistry_RegistryType(reg.RegistryType),
				global.WithRegistry_UseAsMetaReport(reg.UseAsMetaReport),
				global.WithRegistry_UseAsConfigCenter(reg.UseAsConfigCenter),
			),
		)
	}
	if err := newSvcOpts.init(opts...); err != nil {
		return err
	}
	newSvcOpts.Implement(handler)
	if err := newSvcOpts.ExportWithInfo(info); err != nil {
		return err
	}

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
