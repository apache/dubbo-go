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

// Package logger is unified facade provided by Dubbo to work with different logger frameworks, eg, Zapper, Logrus.
package logger

import (
	"fmt"
)

import (
	"github.com/sirupsen/logrus"
	"go.uber.org/zap"
)
import (
	corezap "dubbo.apache.org/dubbo-go/v3/logger/core/zap"
	"github.com/dubbogo/gost/log/logger"
)

func SetLoggerLevel(level string) error {
	l := logger.GetLogger()
	if _, ok := l.(*zap.SugaredLogger); ok {
		_, err := zap.ParseAtomicLevel(level)
		if err != nil {
			return fmt.Errorf("failed to parse log level: %v", err)
		}
		if err := corezap.SetLevel(level); err != nil {
			return fmt.Errorf("failed to set zap logger level: %v", err)
		}

		return nil
	}

	if ll, ok := l.(*logrus.Logger); ok {
		lv, err := logrus.ParseLevel(level)
		if err != nil {
			return fmt.Errorf("failed to parse log level: %v", err)
		}
		ll.SetLevel(lv)
		return nil
	}
	return fmt.Errorf("unsupported logger type: %T", l)
}
