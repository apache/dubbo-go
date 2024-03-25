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

package istio

import (
	"dubbo.apache.org/dubbo-go/v3/istio/bootstrap"
	"dubbo.apache.org/dubbo-go/v3/istio/resources"
	"dubbo.apache.org/dubbo-go/v3/istio/utils"
	"fmt"
	"github.com/dubbogo/gost/log/logger"
	"testing"
)

func TestGetPilotAgent(t *testing.T) {
	bootstrapInfo, _ := bootstrap.GetBootStrapInfo()
	fmt.Sprintf("bootstrapinfo %v", bootstrapInfo)
	bootstrapInfo.SdsGrpcPath = "/Users/jun/GolandProjects/dubbo/dubbo-mesh/var/run/dubbomesh/workload-spiffe-uds/socket"
	bootstrapInfo.XdsGrpcPath = "/Users/jun/GolandProjects/dubbo/dubbo-mesh/var/run/dubbomesh/proxy/XDS"
	pilotAgent, _ := GetPilotAgent(PilotAgentTypeServerWorkload)
	OnRdsChangeListener := func(serviceName string, xdsVirtualHost resources.XdsVirtualHost) error {
		logger.Infof("OnRdsChangeListener serviceName %s with rds = %s", serviceName, utils.ConvertJsonString(xdsVirtualHost))
		return nil
	}
	OnEdsChangeListener := func(clusterName string, xdsCluster resources.XdsCluster, xdsClusterEndpoint resources.XdsClusterEndpoint) error {
		logger.Infof("OnEdsChangeListener clusterName %s with cluster = %s and  eds = %s", clusterName, utils.ConvertJsonString(xdsCluster), utils.ConvertJsonString(xdsClusterEndpoint))
		return nil
	}
	pilotAgent.SubscribeCds("outbound|8000||httpbin.foo.svc.cluster.local", "eds", OnEdsChangeListener)
	pilotAgent.SubscribeRds("8000", "rds", OnRdsChangeListener)
	select {}

}
