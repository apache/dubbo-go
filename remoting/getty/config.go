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
	"time"
)

import (
	perrors "github.com/pkg/errors"
)

import (
	"github.com/apache/dubbo-go/config"
)

type (
	// GettySessionParam is session configuration for getty
	GettySessionParam struct {
		CompressEncoding bool   `default:"false" yaml:"compress_encoding" json:"compress_encoding,omitempty"`
		TcpNoDelay       bool   `default:"true" yaml:"tcp_no_delay" json:"tcp_no_delay,omitempty"`
		TcpKeepAlive     bool   `default:"true" yaml:"tcp_keep_alive" json:"tcp_keep_alive,omitempty"`
		KeepAlivePeriod  string `default:"180s" yaml:"keep_alive_period" json:"keep_alive_period,omitempty"`
		keepAlivePeriod  time.Duration
		TcpRBufSize      int    `default:"262144" yaml:"tcp_r_buf_size" json:"tcp_r_buf_size,omitempty"`
		TcpWBufSize      int    `default:"65536" yaml:"tcp_w_buf_size" json:"tcp_w_buf_size,omitempty"`
		PkgWQSize        int    `default:"1024" yaml:"pkg_wq_size" json:"pkg_wq_size,omitempty"`
		TcpReadTimeout   string `default:"1s" yaml:"tcp_read_timeout" json:"tcp_read_timeout,omitempty"`
		tcpReadTimeout   time.Duration
		TcpWriteTimeout  string `default:"5s" yaml:"tcp_write_timeout" json:"tcp_write_timeout,omitempty"`
		tcpWriteTimeout  time.Duration
		WaitTimeout      string `default:"7s" yaml:"wait_timeout" json:"wait_timeout,omitempty"`
		waitTimeout      time.Duration
		MaxMsgLen        int    `default:"1024" yaml:"max_msg_len" json:"max_msg_len,omitempty"`
		SessionName      string `default:"rpc" yaml:"session_name" json:"session_name,omitempty"`
	}

	// ServerConfig holds supported types by the multiconfig package
	ServerConfig struct {
		SSLEnabled bool

		// heartbeat
		HeartbeatPeriod string `default:"60s" yaml:"heartbeat_period" json:"heartbeat_period,omitempty"`
		heartbeatPeriod time.Duration

		// heartbeat timeout
		HeartbeatTimeout string `default:"5s" yaml:"heartbeat_timeout" json:"heartbeat_timeout,omitempty"`
		heartbeatTimeout time.Duration

		// session
		SessionTimeout string `default:"60s" yaml:"session_timeout" json:"session_timeout,omitempty"`
		sessionTimeout time.Duration
		SessionNumber  int `default:"1000" yaml:"session_number" json:"session_number,omitempty"`

		// grpool
		GrPoolSize  int `default:"0" yaml:"gr_pool_size" json:"gr_pool_size,omitempty"`
		QueueLen    int `default:"0" yaml:"queue_len" json:"queue_len,omitempty"`
		QueueNumber int `default:"0" yaml:"queue_number" json:"queue_number,omitempty"`

		// session tcp parameters
		GettySessionParam GettySessionParam `required:"true" yaml:"getty_session_param" json:"getty_session_param,omitempty"`
	}

	// ClientConfig holds supported types by the multiconfig package
	ClientConfig struct {
		ReconnectInterval int `default:"0" yaml:"reconnect_interval" json:"reconnect_interval,omitempty"`

		// session pool
		ConnectionNum int `default:"16" yaml:"connection_number" json:"connection_number,omitempty"`

		// heartbeat
		HeartbeatPeriod string `default:"60s" yaml:"heartbeat_period" json:"heartbeat_period,omitempty"`
		heartbeatPeriod time.Duration

		// heartbeat timeout
		HeartbeatTimeout string `default:"5s" yaml:"heartbeat_timeout" json:"heartbeat_timeout,omitempty"`
		heartbeatTimeout time.Duration

		// session
		SessionTimeout string `default:"60s" yaml:"session_timeout" json:"session_timeout,omitempty"`
		sessionTimeout time.Duration

		// Connection Pool
		PoolSize int `default:"2" yaml:"pool_size" json:"pool_size,omitempty"`
		PoolTTL  int `default:"180" yaml:"pool_ttl" json:"pool_ttl,omitempty"`

		// grpool
		GrPoolSize  int `default:"0" yaml:"gr_pool_size" json:"gr_pool_size,omitempty"`
		QueueLen    int `default:"0" yaml:"queue_len" json:"queue_len,omitempty"`
		QueueNumber int `default:"0" yaml:"queue_number" json:"queue_number,omitempty"`

		// session tcp parameters
		GettySessionParam GettySessionParam `required:"true" yaml:"getty_session_param" json:"getty_session_param,omitempty"`
	}
)

// GetDefaultClientConfig gets client default configuration
func GetDefaultClientConfig() ClientConfig {
	return ClientConfig{
		ReconnectInterval: 0,
		ConnectionNum:     16,
		HeartbeatPeriod:   "30s",
		SessionTimeout:    "180s",
		PoolSize:          4,
		PoolTTL:           600,
		GrPoolSize:        200,
		QueueLen:          64,
		QueueNumber:       10,
		GettySessionParam: GettySessionParam{
			CompressEncoding: false,
			TcpNoDelay:       true,
			TcpKeepAlive:     true,
			KeepAlivePeriod:  "180s",
			TcpRBufSize:      262144,
			TcpWBufSize:      65536,
			PkgWQSize:        512,
			TcpReadTimeout:   "1s",
			TcpWriteTimeout:  "5s",
			WaitTimeout:      "1s",
			MaxMsgLen:        102400,
			SessionName:      "client",
		}}
}

// GetDefaultServerConfig gets server default configuration
func GetDefaultServerConfig() ServerConfig {
	return ServerConfig{
		SessionTimeout: "180s",
		SessionNumber:  700,
		GrPoolSize:     120,
		QueueNumber:    6,
		QueueLen:       64,
		GettySessionParam: GettySessionParam{
			CompressEncoding: false,
			TcpNoDelay:       true,
			TcpKeepAlive:     true,
			KeepAlivePeriod:  "180s",
			TcpRBufSize:      262144,
			TcpWBufSize:      65536,
			PkgWQSize:        512,
			TcpReadTimeout:   "1s",
			TcpWriteTimeout:  "5s",
			WaitTimeout:      "1s",
			MaxMsgLen:        102400,
			SessionName:      "server",
		},
	}
}

// CheckValidity confirm getty sessian params
func (c *GettySessionParam) CheckValidity() error {
	var err error

	if c.keepAlivePeriod, err = time.ParseDuration(c.KeepAlivePeriod); err != nil {
		return perrors.WithMessagef(err, "time.ParseDuration(KeepAlivePeriod{%#v})", c.KeepAlivePeriod)
	}

	if c.tcpReadTimeout, err = time.ParseDuration(c.TcpReadTimeout); err != nil {
		return perrors.WithMessagef(err, "time.ParseDuration(TcpReadTimeout{%#v})", c.TcpReadTimeout)
	}

	if c.tcpWriteTimeout, err = time.ParseDuration(c.TcpWriteTimeout); err != nil {
		return perrors.WithMessagef(err, "time.ParseDuration(TcpWriteTimeout{%#v})", c.TcpWriteTimeout)
	}

	if c.waitTimeout, err = time.ParseDuration(c.WaitTimeout); err != nil {
		return perrors.WithMessagef(err, "time.ParseDuration(WaitTimeout{%#v})", c.WaitTimeout)
	}

	return nil
}

// CheckValidity confirm client params.
func (c *ClientConfig) CheckValidity() error {
	var err error

	c.ReconnectInterval = c.ReconnectInterval * 1e6

	if c.heartbeatPeriod, err = time.ParseDuration(c.HeartbeatPeriod); err != nil {
		return perrors.WithMessagef(err, "time.ParseDuration(HeartbeatPeroid{%#v})", c.HeartbeatPeriod)
	}

	if c.heartbeatPeriod >= time.Duration(config.MaxWheelTimeSpan) {
		return perrors.WithMessagef(err, "heartbeat_period %s should be less than %s",
			c.HeartbeatPeriod, time.Duration(config.MaxWheelTimeSpan))
	}

	if len(c.HeartbeatTimeout) == 0 {
		c.heartbeatTimeout = 60 * time.Second
	} else if c.heartbeatTimeout, err = time.ParseDuration(c.HeartbeatTimeout); err != nil {
		return perrors.WithMessagef(err, "time.ParseDuration(HeartbeatTimeout{%#v})", c.HeartbeatTimeout)
	}

	if c.sessionTimeout, err = time.ParseDuration(c.SessionTimeout); err != nil {
		return perrors.WithMessagef(err, "time.ParseDuration(SessionTimeout{%#v})", c.SessionTimeout)
	}

	return perrors.WithStack(c.GettySessionParam.CheckValidity())
}

// CheckValidity confirm server params
func (c *ServerConfig) CheckValidity() error {
	var err error

	if len(c.HeartbeatPeriod) == 0 {
		c.heartbeatPeriod = 60 * time.Second
	} else if c.heartbeatPeriod, err = time.ParseDuration(c.HeartbeatPeriod); err != nil {
		return perrors.WithMessagef(err, "time.ParseDuration(HeartbeatPeroid{%#v})", c.HeartbeatPeriod)
	}

	if c.heartbeatPeriod >= time.Duration(config.MaxWheelTimeSpan) {
		return perrors.WithMessagef(err, "heartbeat_period %s should be less than %s",
			c.HeartbeatPeriod, time.Duration(config.MaxWheelTimeSpan))
	}

	if len(c.HeartbeatTimeout) == 0 {
		c.heartbeatTimeout = 60 * time.Second
	} else if c.heartbeatTimeout, err = time.ParseDuration(c.HeartbeatTimeout); err != nil {
		return perrors.WithMessagef(err, "time.ParseDuration(HeartbeatTimeout{%#v})", c.HeartbeatTimeout)
	}

	if c.sessionTimeout, err = time.ParseDuration(c.SessionTimeout); err != nil {
		return perrors.WithMessagef(err, "time.ParseDuration(SessionTimeout{%#v})", c.SessionTimeout)
	}

	if c.sessionTimeout >= time.Duration(config.MaxWheelTimeSpan) {
		return perrors.WithMessagef(err, "session_timeout %s should be less than %s",
			c.SessionTimeout, time.Duration(config.MaxWheelTimeSpan))
	}

	return perrors.WithStack(c.GettySessionParam.CheckValidity())
}
