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

package dubbo

import (
	"fmt"
	"net"
)

import (
	"github.com/dubbogo/getty"
	"github.com/dubbogo/gost/sync"
	"gopkg.in/yaml.v2"
)

import (
	"github.com/apache/dubbo-go/common"
	"github.com/apache/dubbo-go/common/logger"
	"github.com/apache/dubbo-go/config"
)

var (
	srvConf   *ServerConfig
	srvGrpool *gxsync.TaskPool
)

func init() {

	// load clientconfig from provider_config
	protocolConf := config.GetProviderConfig().ProtocolConf
	if protocolConf == nil {
		logger.Warnf("protocol_conf is nil")
		return
	}
	dubboConf := protocolConf.(map[interface{}]interface{})[DUBBO]
	if dubboConf == nil {
		logger.Warnf("dubboConf is nil")
		return
	}

	dubboConfByte, err := yaml.Marshal(dubboConf)
	if err != nil {
		panic(err)
	}
	conf := &ServerConfig{}
	err = yaml.Unmarshal(dubboConfByte, conf)
	if err != nil {
		panic(err)
	}

	if err := conf.CheckValidity(); err != nil {
		panic(err)
	}

	srvConf = conf
	SetServerGrpool()
}

func SetServerConfig(s ServerConfig) {
	srvConf = &s
	err := srvConf.CheckValidity()
	if err != nil {
		logger.Warnf("[ServerConfig CheckValidity] error: %v", err)
		return
	}
	SetServerGrpool()
}

func GetServerConfig() ServerConfig {
	return *srvConf
}

func SetServerGrpool() {
	if srvConf.GrPoolSize > 1 {
		srvGrpool = gxsync.NewTaskPool(gxsync.WithTaskPoolTaskPoolSize(srvConf.GrPoolSize), gxsync.WithTaskPoolTaskQueueLength(srvConf.QueueLen),
			gxsync.WithTaskPoolTaskQueueNumber(srvConf.QueueNumber))
	}
}

type Server struct {
	conf       ServerConfig
	tcpServer  getty.Server
	rpcHandler *RpcServerHandler
}

func NewServer() *Server {

	s := &Server{
		conf: *srvConf,
	}

	s.rpcHandler = NewRpcServerHandler(s.conf.SessionNumber, s.conf.sessionTimeout)

	return s
}

func (s *Server) newSession(session getty.Session) error {
	var (
		ok      bool
		tcpConn *net.TCPConn
	)
	conf := s.conf

	if conf.GettySessionParam.CompressEncoding {
		session.SetCompressType(getty.CompressZip)
	}

	if tcpConn, ok = session.Conn().(*net.TCPConn); !ok {
		panic(fmt.Sprintf("%s, session.conn{%#v} is not tcp connection\n", session.Stat(), session.Conn()))
	}

	tcpConn.SetNoDelay(conf.GettySessionParam.TcpNoDelay)
	tcpConn.SetKeepAlive(conf.GettySessionParam.TcpKeepAlive)
	if conf.GettySessionParam.TcpKeepAlive {
		tcpConn.SetKeepAlivePeriod(conf.GettySessionParam.keepAlivePeriod)
	}
	tcpConn.SetReadBuffer(conf.GettySessionParam.TcpRBufSize)
	tcpConn.SetWriteBuffer(conf.GettySessionParam.TcpWBufSize)

	session.SetName(conf.GettySessionParam.SessionName)
	session.SetMaxMsgLen(conf.GettySessionParam.MaxMsgLen)
	session.SetPkgHandler(rpcServerPkgHandler)
	session.SetEventListener(s.rpcHandler)
	session.SetWQLen(conf.GettySessionParam.PkgWQSize)
	session.SetReadTimeout(conf.GettySessionParam.tcpReadTimeout)
	session.SetWriteTimeout(conf.GettySessionParam.tcpWriteTimeout)
	session.SetCronPeriod((int)(conf.sessionTimeout.Nanoseconds() / 1e6))
	session.SetWaitTime(conf.GettySessionParam.waitTimeout)
	logger.Debugf("app accepts new session:%s\n", session.Stat())

	session.SetTaskPool(srvGrpool)

	return nil
}

func (s *Server) Start(url common.URL) {
	var (
		addr      string
		tcpServer getty.Server
	)

	addr = url.Location
	tcpServer = getty.NewTCPServer(
		getty.WithLocalAddress(addr),
	)
	tcpServer.RunEventLoop(s.newSession)
	logger.Debugf("s bind addr{%s} ok!", addr)
	s.tcpServer = tcpServer

}

func (s *Server) Stop() {
	s.tcpServer.Close()
}
