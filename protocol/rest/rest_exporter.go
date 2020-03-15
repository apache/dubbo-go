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

package rest

import (
	"sync"
)

import (
	"github.com/apache/dubbo-go/common"
	"github.com/apache/dubbo-go/common/constant"
	"github.com/apache/dubbo-go/common/logger"
	"github.com/apache/dubbo-go/protocol"
)

type RestExporter struct {
	protocol.BaseExporter
}

func NewRestExporter(key string, invoker protocol.Invoker, exporterMap *sync.Map) *RestExporter {
	return &RestExporter{
		BaseExporter: *protocol.NewBaseExporter(key, invoker, exporterMap),
	}
}

func (re *RestExporter) Unexport() {
	serviceId := re.GetInvoker().GetUrl().GetParam(constant.BEAN_NAME_KEY, "")
	re.BaseExporter.Unexport()
	err := common.ServiceMap.UnRegister(REST, serviceId)
	if err != nil {
		logger.Errorf("[RestExporter.Unexport] error: %v", err)
	}
	return
}
