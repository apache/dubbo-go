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

package probe

import (
	"context"
	"net/http"
	"sync"
	"time"
)

import (
	"github.com/dubbogo/gost/log/logger"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/common/extension"
	"dubbo.apache.org/dubbo-go/v3/global"
)

type Config struct {
	Enabled          bool
	Port             string
	LivenessPath     string
	ReadinessPath    string
	StartupPath      string
	UseInternalState bool
}

var (
	startOnce sync.Once
)

func Init(cfg *Config) {
	if cfg == nil || !cfg.Enabled {
		return
	}
	startOnce.Do(func() {
		if cfg.UseInternalState {
			EnableInternalState(true)
			SetReady(false)
			SetStartupComplete(false)
			RegisterReadiness("internal", func(ctx context.Context) error {
				if internalReady() {
					return nil
				}
				return errNotReady
			})
			RegisterStartup("internal", func(ctx context.Context) error {
				if internalStartup() {
					return nil
				}
				return errNotStarted
			})
		}

		mux := http.NewServeMux()
		if cfg.LivenessPath != "" {
			mux.HandleFunc(cfg.LivenessPath, livenessHandler)
		}
		if cfg.ReadinessPath != "" {
			mux.HandleFunc(cfg.ReadinessPath, readinessHandler)
		}
		if cfg.StartupPath != "" {
			mux.HandleFunc(cfg.StartupPath, startupHandler)
		}
		srv := &http.Server{Addr: ":" + cfg.Port, Handler: mux}
		extension.AddCustomShutdownCallback(func() {
			SetReady(false)
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			if err := srv.Shutdown(ctx); err != nil {
				logger.Errorf("[kubernetes probe] probe server shutdown failed: %v", err)
			}
		})

		go func() {
			logger.Infof("[kubernetes probe] probe server listening on :%s", cfg.Port)
			if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
				logger.Errorf("[kubernetes probe] probe server stopped with error: %v", err)
			}
		}()
	})
}

func BuildProbeConfig(probeCfg *global.ProbeConfig) *Config {
	if probeCfg == nil || probeCfg.Enabled == nil || !*probeCfg.Enabled {
		return nil
	}

	useInternal := probeCfg.UseInternalState == nil || *probeCfg.UseInternalState
	port := probeCfg.Port
	if port == "" {
		port = constant.ProbeDefaultPort
	}
	livenessPath := probeCfg.LivenessPath
	if livenessPath == "" {
		livenessPath = constant.ProbeDefaultLivenessPath
	}
	readinessPath := probeCfg.ReadinessPath
	if readinessPath == "" {
		readinessPath = constant.ProbeDefaultReadinessPath
	}
	startupPath := probeCfg.StartupPath
	if startupPath == "" {
		startupPath = constant.ProbeDefaultStartupPath
	}

	return &Config{
		Enabled:          true,
		Port:             port,
		LivenessPath:     livenessPath,
		ReadinessPath:    readinessPath,
		StartupPath:      startupPath,
		UseInternalState: useInternal,
	}
}
