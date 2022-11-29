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
	getty "github.com/apache/dubbo-getty"

	perrors "github.com/pkg/errors"
)

import (
	"dubbo.apache.org/dubbo-go/v3/config"
)

const (
	TCPReadWriteTimeoutMinValue = time.Second * 1
)

type (
	// GettySessionParam is session configuration for getty
	GettySessionParam struct {
		CompressEncoding bool   `default:"false" yaml:"compress-encoding" json:"compress-encoding,omitempty"`
		TcpNoDelay       bool   `default:"true" yaml:"tcp-no-delay" json:"tcp-no-delay,omitempty"`
		TcpKeepAlive     bool   `default:"true" yaml:"tcp-keep-alive" json:"tcp-keep-alive,omitempty"`
		KeepAlivePeriod  string `default:"180s" yaml:"keep-alive-period" json:"keep-alive-period,omitempty"`
		keepAlivePeriod  time.Duration
		TcpRBufSize      int    `default:"262144" yaml:"tcp-r-buf-size" json:"tcp-r-buf-size,omitempty"`
		TcpWBufSize      int    `default:"65536" yaml:"tcp-w-buf-size" json:"tcp-w-buf-size,omitempty"`
		TcpReadTimeout   string `default:"1s" yaml:"tcp-read-timeout" json:"tcp-read-timeout,omitempty"`
		tcpReadTimeout   time.Duration
		TcpWriteTimeout  string `default:"5s" yaml:"tcp-write-timeout" json:"tcp-write-timeout,omitempty"`
		tcpWriteTimeout  time.Duration
		WaitTimeout      string `default:"7s" yaml:"wait-timeout" json:"wait-timeout,omitempty"`
		waitTimeout      time.Duration
		MaxMsgLen        int    `default:"1024" yaml:"max-msg-len" json:"max-msg-len,omitempty"`
		SessionName      string `default:"rpc" yaml:"session-name" json:"session-name,omitempty"`
	}

	// ServerConfig holds supported types by the multiconfig package
	ServerConfig struct {
		SSLEnabled bool
		TLSBuilder getty.TlsConfigBuilder

		// heartbeat
		HeartbeatPeriod string `default:"60s" yaml:"heartbeat-period" json:"heartbeat-period,omitempty"`
		heartbeatPeriod time.Duration

		// heartbeat timeout
		HeartbeatTimeout string `default:"5s" yaml:"heartbeat-timeout" json:"heartbeat-timeout,omitempty"`
		heartbeatTimeout time.Duration

		// session
		SessionTimeout string `default:"60s" yaml:"session-timeout" json:"session-timeout,omitempty"`
		sessionTimeout time.Duration
		SessionNumber  int `default:"1000" yaml:"session-number" json:"session-number,omitempty"`

		// gr pool
		GrPoolSize  int `default:"0" yaml:"gr-pool-size" json:"gr-pool-size,omitempty"`
		QueueLen    int `default:"0" yaml:"queue-len" json:"queue-len,omitempty"`
		QueueNumber int `default:"0" yaml:"queue-number" json:"queue-number,omitempty"`

		// session tcp parameters
		GettySessionParam GettySessionParam `required:"true" yaml:"getty-session-param" json:"getty-session-param,omitempty"`
	}

	// ClientConfig holds supported types by the multi config package
	ClientConfig struct {
		SSLEnabled bool
		TLSBuilder getty.TlsConfigBuilder

		ReconnectInterval int `default:"0" yaml:"reconnect-interval" json:"reconnect-interval,omitempty"`

		// session pool
		ConnectionNum int `default:"16" yaml:"connection-number" json:"connection-number,omitempty"`

		// heartbeat
		HeartbeatPeriod string `default:"60s" yaml:"heartbeat-period" json:"heartbeat-period,omitempty"`
		heartbeatPeriod time.Duration

		// heartbeat timeout
		HeartbeatTimeout string `default:"5s" yaml:"heartbeat-timeout" json:"heartbeat-timeout,omitempty"`
		heartbeatTimeout time.Duration

		// session
		SessionTimeout string `default:"60s" yaml:"session-timeout" json:"session-timeout,omitempty"`
		sessionTimeout time.Duration

		// gr pool
		GrPoolSize  int `default:"0" yaml:"gr-pool-size" json:"gr-pool-size,omitempty"`
		QueueLen    int `default:"0" yaml:"queue-len" json:"queue-len,omitempty"`
		QueueNumber int `default:"0" yaml:"queue-number" json:"queue-number,omitempty"`

		// session tcp parameters
		GettySessionParam GettySessionParam `required:"true" yaml:"getty-session-param" json:"getty-session-param,omitempty"`
	}
)

// GetDefaultClientConfig gets client default configuration
func GetDefaultClientConfig() *ClientConfig {
	defaultClientConfig := &ClientConfig{
		ReconnectInterval: 0,
		ConnectionNum:     16,
		HeartbeatPeriod:   "30s",
		SessionTimeout:    "180s",
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
			TcpReadTimeout:   "1s",
			TcpWriteTimeout:  "5s",
			WaitTimeout:      "1s",
			MaxMsgLen:        102400,
			SessionName:      "client",
		},
	}
	_ = defaultClientConfig.CheckValidity()
	return defaultClientConfig
}

// GetDefaultServerConfig gets server default configuration
func GetDefaultServerConfig() *ServerConfig {
	defaultServerConfig := &ServerConfig{
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
			TcpReadTimeout:   "1s",
			TcpWriteTimeout:  "5s",
			WaitTimeout:      "1s",
			MaxMsgLen:        102400,
			SessionName:      "server",
		},
	}
	_ = defaultServerConfig.CheckValidity()
	return defaultServerConfig
}

// CheckValidity confirm getty session params
func (c *GettySessionParam) CheckValidity() error {
	var err error

	if c.keepAlivePeriod, err = time.ParseDuration(c.KeepAlivePeriod); err != nil {
		return perrors.WithMessagef(err, "time.ParseDuration(KeepAlivePeriod{%#v})", c.KeepAlivePeriod)
	}

	if c.tcpReadTimeout, err = parseTcpTimeoutDuration(c.TcpReadTimeout); err != nil {
		return perrors.WithMessagef(err, "time.ParseDuration(TcpReadTimeout{%#v})", c.TcpReadTimeout)
	}

	if c.tcpWriteTimeout, err = parseTcpTimeoutDuration(c.TcpWriteTimeout); err != nil {
		return perrors.WithMessagef(err, "time.ParseDuration(TcpWriteTimeout{%#v})", c.TcpWriteTimeout)
	}

	if c.waitTimeout, err = time.ParseDuration(c.WaitTimeout); err != nil {
		return perrors.WithMessagef(err, "time.ParseDuration(WaitTimeout{%#v})", c.WaitTimeout)
	}

	return nil
}

func parseTcpTimeoutDuration(timeStr string) (time.Duration, error) {
	result, err := time.ParseDuration(timeStr)
	if err != nil {
		return 0, err
	}
	if result < TCPReadWriteTimeoutMinValue {
		return TCPReadWriteTimeoutMinValue, nil
	}
	return result, nil
}

// CheckValidity confirm client params.
func (c *ClientConfig) CheckValidity() error {
	var err error

	c.ReconnectInterval = c.ReconnectInterval * 1e6

	if c.heartbeatPeriod, err = time.ParseDuration(c.HeartbeatPeriod); err != nil {
		return perrors.WithMessagef(err, "time.ParseDuration(HeartbeatPeroid{%#v})", c.HeartbeatPeriod)
	}

	if c.heartbeatPeriod >= time.Duration(config.MaxWheelTimeSpan) {
		return perrors.WithMessagef(err, "heartbeat-period %s should be less than %s",
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
		return perrors.WithMessagef(err, "heartbeat-period %s should be less than %s",
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
		return perrors.WithMessagef(err, "session-timeout %s should be less than %s",
			c.SessionTimeout, time.Duration(config.MaxWheelTimeSpan))
	}

	return perrors.WithStack(c.GettySessionParam.CheckValidity())
}
