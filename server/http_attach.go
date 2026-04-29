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
	"dubbo.apache.org/dubbo-go/v3/global"
	"dubbo.apache.org/dubbo-go/v3/protocol/base"
)

// AttachHTTPHandler records an existing HTTP root handler on the server before
// Serve starts. The attached handler is later hosted on the selected protocol
// listener and acts as the transport-level fallback after Triple route lookup,
// so callers should aggregate any HTTP sub-services behind their own mux/router
// before attaching.
func (s *Server) AttachHTTPHandler(handler http.Handler) error {
	if handler == nil {
		return fmt.Errorf("attached HTTP handler must not be nil")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if s.serve {
		return fmt.Errorf("attached HTTP handler must be configured before Serve")
	}
	// Server-level HTTP integration is a single root handler. Callers that need
	// multiple HTTP services should compose them behind their own mux/router.
	if s.attachedHTTPHandler != nil {
		return fmt.Errorf("an HTTP handler has already been attached")
	}

	s.attachedHTTPHandler = handler
	return nil
}

// hostAttachedHTTPHandler asks an HTTP-capable protocol to host the attached
// root handler on the same listener as framework-managed routes. Today that
// means Triple running on an explicit, user-selected port.
func (s *Server) hostAttachedHTTPHandler() error {
	s.mu.RLock()
	handler := s.attachedHTTPHandler
	s.mu.RUnlock()
	if handler == nil {
		return nil
	}

	protocolConfigs := loadProtocol(s.cfg.Provider.ProtocolIDs, s.cfg.Protocols)
	if len(protocolConfigs) == 0 {
		// WithServerProtocol populates Server.Protocols, but HTTP mounting can run
		// before any service export wires Provider.ProtocolIDs. Fall back to the
		// declared server protocols so mount-only startup still works.
		protocolConfigs = make([]*global.ProtocolConfig, 0, len(s.cfg.Protocols))
		for _, protocolConf := range s.cfg.Protocols {
			protocolConfigs = append(protocolConfigs, protocolConf)
		}
	}
	mounted := false

	for _, protocolConf := range protocolConfigs {
		if protocolConf == nil || protocolConf.Name != constant.TriProtocol {
			continue
		}
		if protocolConf.Port == "" {
			// Hosting needs a stable listener up front; unlike normal service
			// export we should not silently start on an implicit/random port here.
			return fmt.Errorf("hosting an attached HTTP handler requires an explicit triple port")
		}

		proto := extension.GetProtocol(protocolConf.Name)
		host, ok := proto.(base.HTTPHandlerHost)
		if !ok {
			return fmt.Errorf("protocol %s does not support hosting attached HTTP handlers", protocolConf.Name)
		}

		u := common.NewURLWithOptions(
			common.WithProtocol(protocolConf.Name),
			common.WithIp(protocolConf.Ip),
			common.WithPort(protocolConf.Port),
			common.WithAttribute(constant.TripleConfigKey, protocolConf.TripleConfig),
			common.WithAttribute(constant.TLSConfigKey, s.cfg.TLS),
		)
		if err := host.HostHTTPHandler(u, handler); err != nil {
			return err
		}
		mounted = true
	}

	if !mounted {
		return fmt.Errorf("attached HTTP handler requires at least one triple protocol")
	}
	return nil
}
