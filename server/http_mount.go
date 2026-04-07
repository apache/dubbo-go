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
	"fmt"
	"net/http"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/common/extension"
	"dubbo.apache.org/dubbo-go/v3/protocol/base"
)

func (s *Server) MountHTTPHandler(handler http.Handler) error {
	if handler == nil {
		return fmt.Errorf("mounted HTTP handler must not be nil")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	s.mountedHTTPHandler = handler
	return nil
}

func (s *Server) mountedHTTPHandlerSnapshot() http.Handler {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.mountedHTTPHandler
}

func (s *Server) mountHTTPHandlers() error {
	handler := s.mountedHTTPHandlerSnapshot()
	if handler == nil {
		return nil
	}

	protocolConfigs := loadProtocol(s.cfg.Provider.ProtocolIDs, s.cfg.Protocols)
	mounted := false

	for _, protocolConf := range protocolConfigs {
		if protocolConf == nil || protocolConf.Name != constant.TriProtocol {
			continue
		}
		if protocolConf.Port == "" {

		}

		proto := extension.GetProtocol(protocolConf.Name)
		mountable, ok := proto.(base.HTTPHandlerMountable)
		if !ok {

		}

		u := common.NewURLWithOptions(
			common.WithProtocol(protocolConf.Name),
			common.WithIp(protocolConf.Ip),
			common.WithPort(protocolConf.Port),
			common.WithAttribute(constant.TripleConfigKey, protocolConf.TripleConfig),
			common.WithAttribute(constant.TLSConfigKey, s.cfg.TLS),
			common.WithAttribute(constant.MountedHTTPHandlerKey, handler),
		)
		if err := mountable.MountHTTPHandler(u, handler); err != nil {
			return err
		}
		mounted = true
	}

	if !mounted {

	}
	return nil
}
