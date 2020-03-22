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

package remoting

import (
	"time"

	"github.com/dubbogo/getty"

	perrors "github.com/pkg/errors"
)

type (
	// GettySessionParam ...
	GettySessionParam struct {
		CompressEncoding bool   `default:"false" yaml:"compress_encoding" json:"compress_encoding,omitempty"`
		TcpNoDelay       bool   `default:"true" yaml:"tcp_no_delay" json:"tcp_no_delay,omitempty"`
		TcpKeepAlive     bool   `default:"true" yaml:"tcp_keep_alive" json:"tcp_keep_alive,omitempty"`
		KeepAlivePeriod  string `default:"180s" yaml:"keep_alive_period" json:"keep_alive_period,omitempty"`
		KeepAlivePeriodD time.Duration
		TcpRBufSize      int    `default:"262144" yaml:"tcp_r_buf_size" json:"tcp_r_buf_size,omitempty"`
		TcpWBufSize      int    `default:"65536" yaml:"tcp_w_buf_size" json:"tcp_w_buf_size,omitempty"`
		PkgWQSize        int    `default:"1024" yaml:"pkg_wq_size" json:"pkg_wq_size,omitempty"`
		TcpReadTimeout   string `default:"1s" yaml:"tcp_read_timeout" json:"tcp_read_timeout,omitempty"`
		TcpReadTimeoutD  time.Duration
		TcpWriteTimeout  string `default:"5s" yaml:"tcp_write_timeout" json:"tcp_write_timeout,omitempty"`
		TcpWriteTimeoutD time.Duration
		WaitTimeout      string `default:"7s" yaml:"wait_timeout" json:"wait_timeout,omitempty"`
		WaitTimeoutD     time.Duration
		MaxMsgLen        int    `default:"1024" yaml:"max_msg_len" json:"max_msg_len,omitempty"`
		SessionName      string `default:"rpc" yaml:"session_name" json:"session_name,omitempty"`
	}

	// ServerConfig
	//Config holds supported types by the multiconfig package
	ServerConfig struct {
		// session
		SessionTimeout  string `default:"60s" yaml:"session_timeout" json:"session_timeout,omitempty"`
		SessionTimeoutD time.Duration
		SessionNumber   int `default:"1000" yaml:"session_number" json:"session_number,omitempty"`

		// grpool
		GrPoolSize  int `default:"0" yaml:"gr_pool_size" json:"gr_pool_size,omitempty"`
		QueueLen    int `default:"0" yaml:"queue_len" json:"queue_len,omitempty"`
		QueueNumber int `default:"0" yaml:"queue_number" json:"queue_number,omitempty"`

		// session tcp parameters
		GettySessionParam GettySessionParam `required:"true" yaml:"getty_session_param" json:"getty_session_param,omitempty"`

		// serialization
		Serialization string `default:"hessian2" yaml:"serialization" json:"serialization"`
	}

	// ClientConfig
	//Config holds supported types by the multiconfig package
	ClientConfig struct {
		ReconnectInterval int `default:"0" yaml:"reconnect_interval" json:"reconnect_interval,omitempty"`

		// session Pool
		ConnectionNum int `default:"16" yaml:"connection_number" json:"connection_number,omitempty"`

		// heartbeat
		HeartbeatPeriod  string `default:"15s" yaml:"heartbeat_period" json:"heartbeat_period,omitempty"`
		HeartbeatPeriodD time.Duration

		// session
		SessionTimeout  string `default:"60s" yaml:"session_timeout" json:"session_timeout,omitempty"`
		SessionTimeoutD time.Duration

		// Connection Pool
		PoolSize int `default:"2" yaml:"pool_size" json:"pool_size,omitempty"`
		PoolTTL  int `default:"180" yaml:"pool_ttl" json:"pool_ttl,omitempty"`

		// grpool
		GrPoolSize  int `default:"0" yaml:"gr_pool_size" json:"gr_pool_size,omitempty"`
		QueueLen    int `default:"0" yaml:"queue_len" json:"queue_len,omitempty"`
		QueueNumber int `default:"0" yaml:"queue_number" json:"queue_number,omitempty"`

		// session tcp parameters
		GettySessionParam GettySessionParam `required:"true" yaml:"getty_session_param" json:"getty_session_param,omitempty"`

		// serialization
		Serialization string `default:"hessian2" yaml:"serialization" json:"serialization"`
	}
)

// GetDefaultClientConfig ...
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
		Serialization:     "hessian2",
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

// GetDefaultServerConfig ...
func GetDefaultServerConfig() ServerConfig {
	return ServerConfig{
		SessionTimeout: "180s",
		SessionNumber:  700,
		GrPoolSize:     120,
		QueueNumber:    6,
		QueueLen:       64,
		Serialization:  "hessian2",
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

// CheckValidity ...
func (c *GettySessionParam) CheckValidity() error {
	var err error

	if c.KeepAlivePeriodD, err = time.ParseDuration(c.KeepAlivePeriod); err != nil {
		return perrors.WithMessagef(err, "time.ParseDuration(KeepAlivePeriod{%#v})", c.KeepAlivePeriod)
	}

	if c.TcpReadTimeoutD, err = time.ParseDuration(c.TcpReadTimeout); err != nil {
		return perrors.WithMessagef(err, "time.ParseDuration(TcpReadTimeout{%#v})", c.TcpReadTimeout)
	}

	if c.TcpWriteTimeoutD, err = time.ParseDuration(c.TcpWriteTimeout); err != nil {
		return perrors.WithMessagef(err, "time.ParseDuration(TcpWriteTimeout{%#v})", c.TcpWriteTimeout)
	}

	if c.WaitTimeoutD, err = time.ParseDuration(c.WaitTimeout); err != nil {
		return perrors.WithMessagef(err, "time.ParseDuration(WaitTimeout{%#v})", c.WaitTimeout)
	}

	return nil
}

// CheckValidity ...
func (c *ClientConfig) CheckValidity() error {
	var err error

	c.ReconnectInterval = c.ReconnectInterval * 1e6

	if c.HeartbeatPeriodD, err = time.ParseDuration(c.HeartbeatPeriod); err != nil {
		return perrors.WithMessagef(err, "time.ParseDuration(HeartbeatPeroid{%#v})", c.HeartbeatPeriod)
	}

	if c.HeartbeatPeriodD >= time.Duration(getty.MaxWheelTimeSpan) {
		return perrors.WithMessagef(err, "heartbeat_period %s should be less than %s",
			c.HeartbeatPeriod, time.Duration(getty.MaxWheelTimeSpan))
	}

	if c.SessionTimeoutD, err = time.ParseDuration(c.SessionTimeout); err != nil {
		return perrors.WithMessagef(err, "time.ParseDuration(SessionTimeout{%#v})", c.SessionTimeout)
	}

	return perrors.WithStack(c.GettySessionParam.CheckValidity())
}

// CheckValidity ...
func (c *ServerConfig) CheckValidity() error {
	var err error

	if c.SessionTimeoutD, err = time.ParseDuration(c.SessionTimeout); err != nil {
		return perrors.WithMessagef(err, "time.ParseDuration(SessionTimeout{%#v})", c.SessionTimeout)
	}

	if c.SessionTimeoutD >= time.Duration(getty.MaxWheelTimeSpan) {
		return perrors.WithMessagef(err, "session_timeout %s should be less than %s",
			c.SessionTimeout, time.Duration(getty.MaxWheelTimeSpan))
	}

	return perrors.WithStack(c.GettySessionParam.CheckValidity())
}
