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

package getty

import (
	"crypto/tls"
	"fmt"
	"net"
	"sync"
)

import (
	getty "github.com/apache/dubbo-getty"

	"github.com/dubbogo/gost/log/logger"
	gxsync "github.com/dubbogo/gost/sync"

	"go.yaml.in/yaml/v4"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	commonCfg "dubbo.apache.org/dubbo-go/v3/common/config"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/global"
	"dubbo.apache.org/dubbo-go/v3/protocol/invocation"
	"dubbo.apache.org/dubbo-go/v3/protocol/result"
	"dubbo.apache.org/dubbo-go/v3/remoting"
	dubbotls "dubbo.apache.org/dubbo-go/v3/tls"
)

var (
	srvConf     = GetDefaultServerConfig()
	srvConfLock sync.RWMutex
)

type preparedServerConfig struct {
	tlsBuilder *getty.ServerTlsConfigBuilder
	hasParams  bool
	params     []byte
	marshalErr error
}

func initServer(url *common.URL) (conf ServerConfig) {
	var prepared preparedServerConfig
	defer func() {
		conf = applyServerConfig(prepared)
	}()
	prepareServerConfig(url, &prepared)
	return
}

func serverProtocolConfig(url *common.URL) *global.ProtocolConfig {
	if url.Protocol == "" {
		return nil
	}

	// Ensure application config is available via URL attribute
	commonCfg.EnsureApplicationAttribute(url)

	protocolConfRaw, ok := url.GetAttribute(constant.ProtocolConfigKey)
	if !ok || protocolConfRaw == nil {
		logger.Warn("[Remoting][Getty] protocolConfig not found in URL attributes")
		return nil
	}
	protocolConfMap, ok := protocolConfRaw.(map[string]*global.ProtocolConfig)
	if !ok || protocolConfMap == nil {
		logger.Warn("[Remoting][Getty] protocolConfig assert failed or is nil")
		return nil
	}

	protocolConf := protocolConfMap[url.Protocol]
	if protocolConf == nil {
		logger.Debug("[Remoting][Getty] use default getty server config")
	}
	return protocolConf
}

func prepareServerConfig(url *common.URL, prepared *preparedServerConfig) {
	protocolConf := serverProtocolConfig(url)
	if protocolConf == nil {
		return
	}

	// server tls config
	if tlsConfRaw, ok := url.GetAttribute(constant.TLSConfigKey); ok {
		tlsConf, ok := tlsConfRaw.(*global.TLSConfig)
		if !ok {
			logger.Error("[Remoting][Getty] getty server initialized the TLSConfig configuration failed")
			return
		}
		if dubbotls.IsServerTLSValid(tlsConf) {
			prepared.tlsBuilder = &getty.ServerTlsConfigBuilder{
				ServerKeyCertChainPath:        tlsConf.TLSCertFile,
				ServerPrivateKeyPath:          tlsConf.TLSKeyFile,
				ServerTrustCertCollectionPath: tlsConf.CACertFile,
			}
		}
	}
	//getty params
	gettyServerConfig := protocolConf.Params
	if gettyServerConfig == nil {
		logger.Debug("[Remoting][Getty] gettyServerConfig is nil")
		return
	}

	prepared.params, prepared.marshalErr = yaml.Marshal(gettyServerConfig)
	prepared.hasParams = true
}

func applyServerConfig(prepared preparedServerConfig) ServerConfig {
	srvConfLock.Lock()
	defer srvConfLock.Unlock()

	if prepared.tlsBuilder != nil {
		srvConf.SSLEnabled = true
		srvConf.TLSBuilder = prepared.tlsBuilder
		logger.Info("[Remoting][Getty] getty server initialized the TLSConfig configuration")
	}
	if prepared.hasParams {
		if prepared.marshalErr != nil {
			panic(prepared.marshalErr)
		}
		if err := yaml.Unmarshal(prepared.params, srvConf); err != nil {
			panic(err)
		}
		if err := srvConf.CheckValidity(); err != nil {
			panic(err)
		}
	}
	return cloneServerConfig(*srvConf)
}

func cloneServerConfig(conf ServerConfig) ServerConfig {
	if builder, ok := conf.TLSBuilder.(*getty.ServerTlsConfigBuilder); ok && builder != nil {
		copied := *builder
		conf.TLSBuilder = &copied
	}
	return conf
}

// SetServerConfig set dubbo server config.
func SetServerConfig(s ServerConfig) {
	srvConfLock.Lock()
	defer srvConfLock.Unlock()

	s = cloneServerConfig(s)
	srvConf = &s
	err := srvConf.CheckValidity()
	if err != nil {
		logger.Warnf("[Remoting][Getty] serverConfig checkValidity, err=%v", err)
		return
	}
}

// GetServerConfig get getty server config.
func GetServerConfig() ServerConfig {
	srvConfLock.RLock()
	defer srvConfLock.RUnlock()

	return cloneServerConfig(*srvConf)
}

// Server define getty server
type Server struct {
	conf           ServerConfig
	addr           string
	codec          remoting.Codec
	tcpServer      getty.Server
	rpcHandler     *RpcServerHandler
	requestHandler func(*invocation.RPCInvocation) result.RPCResult
}

// NewServer create a new Server
func NewServer(url *common.URL, handlers func(*invocation.RPCInvocation) result.RPCResult) *Server {
	// init
	conf := initServer(url)
	s := &Server{
		conf:           conf,
		addr:           url.Location,
		codec:          remoting.GetCodec(url.Protocol),
		requestHandler: handlers,
	}

	s.rpcHandler = NewRpcServerHandler(s.conf.SessionNumber, s.conf.sessionTimeout, s)

	return s
}

func (s *Server) newSession(session getty.Session) error {
	var (
		ok      bool
		tcpConn *net.TCPConn
		err     error
	)
	conf := s.conf

	if conf.GettySessionParam.CompressEncoding {
		session.SetCompressType(getty.CompressZip)
	}
	if _, ok = session.Conn().(*tls.Conn); ok {
		session.SetName(conf.GettySessionParam.SessionName)
		session.SetMaxMsgLen(conf.GettySessionParam.MaxMsgLen)
		session.SetPkgHandler(NewRpcServerPackageHandler(s))
		session.SetEventListener(s.rpcHandler)
		session.SetReadTimeout(conf.GettySessionParam.tcpReadTimeout)
		session.SetWriteTimeout(conf.GettySessionParam.tcpWriteTimeout)
		session.SetCronPeriod((int)(conf.heartbeatPeriod.Nanoseconds() / 1e6))
		session.SetWaitTime(conf.GettySessionParam.waitTimeout)
		logger.Debugf("[Remoting][Getty] server accepts new session=%s", session.Stat())
		return nil
	}
	if _, ok = session.Conn().(*net.TCPConn); !ok {
		panic(fmt.Sprintf("%s, session.conn{%#v} is not tcp connection\n", session.Stat(), session.Conn()))
	}

	if _, ok = session.Conn().(*tls.Conn); !ok {
		if tcpConn, ok = session.Conn().(*net.TCPConn); !ok {
			return fmt.Errorf("%s, session.conn{%#v} is not tcp connection", session.Stat(), session.Conn())
		}

		if err = tcpConn.SetNoDelay(conf.GettySessionParam.TcpNoDelay); err != nil {
			return err
		}
		if err = tcpConn.SetKeepAlive(conf.GettySessionParam.TcpKeepAlive); err != nil {
			return err
		}
		if conf.GettySessionParam.TcpKeepAlive {
			if err = tcpConn.SetKeepAlivePeriod(conf.GettySessionParam.keepAlivePeriod); err != nil {
				return err
			}
		}
		if err = tcpConn.SetReadBuffer(conf.GettySessionParam.TcpRBufSize); err != nil {
			return err
		}
		if err = tcpConn.SetWriteBuffer(conf.GettySessionParam.TcpWBufSize); err != nil {
			return err
		}
	}

	session.SetName(conf.GettySessionParam.SessionName)
	session.SetMaxMsgLen(conf.GettySessionParam.MaxMsgLen)
	session.SetPkgHandler(NewRpcServerPackageHandler(s))
	session.SetEventListener(s.rpcHandler)
	session.SetReadTimeout(conf.GettySessionParam.tcpReadTimeout)
	session.SetWriteTimeout(conf.GettySessionParam.tcpWriteTimeout)
	session.SetCronPeriod((int)(conf.heartbeatPeriod.Nanoseconds() / 1e6))
	session.SetWaitTime(conf.GettySessionParam.waitTimeout)
	logger.Debugf("[Remoting][Getty] server accepts new session=%s", session.Stat())
	return nil
}

// Start dubbo server.
func (s *Server) Start() {
	var (
		addr      string
		tcpServer getty.Server
	)

	addr = s.addr
	serverOpts := []getty.ServerOption{getty.WithLocalAddress(addr)}
	if s.conf.SSLEnabled {
		serverOpts = append(serverOpts, getty.WithServerSslEnabled(s.conf.SSLEnabled),
			getty.WithServerTlsConfigBuilder(s.conf.TLSBuilder))
		logger.Info("[Remoting][Getty] getty server initialized the TLSConfig configuration")
	}

	serverOpts = append(serverOpts, getty.WithServerTaskPool(gxsync.NewTaskPoolSimple(s.conf.GrPoolSize)))

	tcpServer = getty.NewTCPServer(serverOpts...)
	tcpServer.RunEventLoop(s.newSession)
	logger.Debugf("[Remoting][Getty] s bind addr=%s ok", s.addr)
	s.tcpServer = tcpServer
}

// Stop dubbo server
func (s *Server) Stop() {
	s.tcpServer.Close()
}
