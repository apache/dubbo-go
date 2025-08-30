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

	getty "github.com/apache/dubbo-getty"
	"github.com/dubbogo/gost/log/logger"

	gxsync "github.com/dubbogo/gost/sync"

	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/config"
	"dubbo.apache.org/dubbo-go/v3/global"
	"dubbo.apache.org/dubbo-go/v3/protocol/invocation"
	"dubbo.apache.org/dubbo-go/v3/protocol/result"
	"dubbo.apache.org/dubbo-go/v3/remoting"
	perrors "github.com/pkg/errors"
	"gopkg.in/yaml.v2"

	dubbotls "dubbo.apache.org/dubbo-go/v3/tls"
)

var (
	srvConf = GetDefaultServerConfig()
)

func initServer(url *common.URL) {
	if url.Protocol == "" {
		return
	}

	// load server config from rootConfig.Protocols
	// default use dubbo
	if config.GetApplicationConfig() == nil {
		return
	}
	if config.GetRootConfig().Protocols == nil {
		return
	}

	protocolConf := config.GetRootConfig().Protocols[url.Protocol]
	if protocolConf == nil {
		logger.Debug("use default getty server config")
		return
	} else {
		//server tls config
		tlsConfig := config.GetRootConfig().TLSConfig
		if tlsConfig != nil {
			srvConf.SSLEnabled = true
			srvConf.TLSBuilder = &getty.ServerTlsConfigBuilder{
				ServerKeyCertChainPath:        tlsConfig.TLSCertFile,
				ServerPrivateKeyPath:          tlsConfig.TLSKeyFile,
				ServerTrustCertCollectionPath: tlsConfig.CACertFile,
			}
			logger.Infof("Getty Server initialized the TLSConfig configuration")
		} else if tlsConfRaw, ok := url.GetAttribute(constant.TLSConfigKey); ok {
			// use global TLSConfig handle tls
			tlsConf, ok := tlsConfRaw.(*global.TLSConfig)
			if !ok {
				logger.Errorf("Getty Server initialized the TLSConfig configuration failed")
				return
			}
			if dubbotls.IsServerTLSValid(tlsConf) {
				srvConf.SSLEnabled = true
				srvConf.TLSBuilder = &getty.ServerTlsConfigBuilder{
					ServerKeyCertChainPath:        tlsConf.TLSCertFile,
					ServerPrivateKeyPath:          tlsConf.TLSKeyFile,
					ServerTrustCertCollectionPath: tlsConf.CACertFile,
				}
				logger.Infof("Getty Server initialized the TLSConfig configuration")
			}
		}
		//getty params
		gettyServerConfig := protocolConf.Params
		if gettyServerConfig == nil {
			logger.Debug("gettyServerConfig is nil")
			return
		}

		gettyServerConfigBytes, err := yaml.Marshal(gettyServerConfig)
		if err != nil {
			logger.Errorf("failed to marshal getty server config: %v", err)
			return
		}
		err = yaml.Unmarshal(gettyServerConfigBytes, srvConf)
		if err != nil {
			logger.Errorf("failed to unmarshal getty server config: %v", err)
			return
		}
	}

	if err := srvConf.CheckValidity(); err != nil {
		logger.Errorf("server config check validity failed: %v", err)
		return
	}
}

// SetServerConfig set dubbo server config.
func SetServerConfig(s ServerConfig) {
	srvConf = &s
	err := srvConf.CheckValidity()
	if err != nil {
		logger.Warnf("[ServerConfig CheckValidity] error: %v", err)
		return
	}
}

// GetServerConfig get getty server config.
func GetServerConfig() ServerConfig {
	return *srvConf
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
	initServer(url)
	s := &Server{
		conf:           *srvConf,
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
		logger.Debugf("server accepts new session:%s\n", session.Stat())
		return nil
	}
	if _, ok = session.Conn().(*net.TCPConn); !ok {
		return perrors.Errorf("%s, session.conn{%#v} is not tcp connection", session.Stat(), session.Conn())
	}

	if _, ok = session.Conn().(*tls.Conn); !ok {
		if tcpConn, ok = session.Conn().(*net.TCPConn); !ok {
			return perrors.New(fmt.Sprintf("%s, session.conn{%#v} is not tcp connection", session.Stat(), session.Conn()))
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
	logger.Debugf("server accepts new session: %s", session.Stat())
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
			getty.WithServerTlsConfigBuilder(srvConf.TLSBuilder))
		logger.Infof("Getty Server initialized the TLSConfig configuration")
	}

	serverOpts = append(serverOpts, getty.WithServerTaskPool(gxsync.NewTaskPoolSimple(s.conf.GrPoolSize)))

	tcpServer = getty.NewTCPServer(serverOpts...)
	tcpServer.RunEventLoop(s.newSession)
	logger.Debugf("s bind addr{%s} ok!", s.addr)
	s.tcpServer = tcpServer
}

// Stop dubbo server
func (s *Server) Stop() {
	s.tcpServer.Close()
}
