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

package polaris

import (
	"context"
)

import (
	"github.com/polarismesh/polaris-go/pkg/model"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/protocol"
)

var (
	_ model.Instance   = (*polarisInvoker)(nil)
	_ protocol.Invoker = (*polarisInvoker)(nil)
)

func newPolarisInvoker(invoker protocol.Invoker) *polarisInvoker {
	p := &polarisInvoker{invoker: invoker}
	return p
}

type polarisInvoker struct {
	invoker protocol.Invoker
	ins     model.Instance
}

// GetInstanceKey 获取实例四元组标识
func (p *polarisInvoker) GetInstanceKey() model.InstanceKey {
	return p.ins.GetInstanceKey()
}

// GetNamespace 实例所在命名空间
func (p *polarisInvoker) GetNamespace() string {
	return p.ins.GetNamespace()
}

// GetService 实例所在服务名
func (p *polarisInvoker) GetService() string {
	return p.ins.GetService()
}

// GetId 服务实例唯一标识
func (p *polarisInvoker) GetId() string {
	return p.ins.GetId()
}

// GetHost 实例的域名/IP信息
func (p *polarisInvoker) GetHost() string {
	return p.ins.GetHost()
}

// GetPort 实例的监听端口
func (p *polarisInvoker) GetPort() uint32 {
	return p.ins.GetPort()
}

// GetVpcId 实例的vpcId
func (p *polarisInvoker) GetVpcId() string {
	return p.ins.GetVpcId()
}

// GetProtocol 服务实例的协议
func (p *polarisInvoker) GetProtocol() string {
	return p.ins.GetProtocol()
}

// GetVersion 实例版本号
func (p *polarisInvoker) GetVersion() string {
	return p.ins.GetVersion()
}

// GetWeight 实例静态权重值
func (p *polarisInvoker) GetWeight() int {
	return p.ins.GetWeight()
}

// GetPriority 实例优先级信息
func (p *polarisInvoker) GetPriority() uint32 {
	return p.ins.GetPriority()
}

// GetMetadata 实例元数据信息
func (p *polarisInvoker) GetMetadata() map[string]string {
	return p.ins.GetMetadata()
}

// GetLogicSet 实例逻辑分区
func (p *polarisInvoker) GetLogicSet() string {
	return p.ins.GetLogicSet()
}

// GetCircuitBreakerStatus 实例的断路器状态，包括：
// 打开（被熔断）、半开（探测恢复）、关闭（正常运行）
func (p *polarisInvoker) GetCircuitBreakerStatus() model.CircuitBreakerStatus {
	return p.ins.GetCircuitBreakerStatus()
}

// IsHealthy 实例是否健康，基于服务端返回的健康数据
func (p *polarisInvoker) IsHealthy() bool {
	return p.ins.IsHealthy()
}

// IsIsolated 实例是否已经被手动隔离
func (p *polarisInvoker) IsIsolated() bool {
	return p.ins.IsIsolated()
}

// IsEnableHealthCheck 实例是否启动了健康检查
func (p *polarisInvoker) IsEnableHealthCheck() bool {
	return p.ins.IsEnableHealthCheck()
}

// GetRegion 实例所属的大区信息
func (p *polarisInvoker) GetRegion() string {
	return p.ins.GetRegion()
}

// GetZone 实例所属的地方信息
func (p *polarisInvoker) GetZone() string {
	return p.ins.GetRegion()
}

// GetIDC .
// Deprecated，建议使用GetCampus方法
func (p *polarisInvoker) GetIDC() string {
	return p.ins.GetIDC()
}

// GetCampus 实例所属的园区信息
func (p *polarisInvoker) GetCampus() string {
	return p.ins.GetCampus()
}

// GetRevision .获取实例的修订版本信息
// 与上一次比较，用于确认服务实例是否发生变更
func (p *polarisInvoker) GetRevision() string {
	return p.ins.GetRevision()
}

func (p *polarisInvoker) Invoke(ctx context.Context, invoaction protocol.Invocation) protocol.Result {
	return p.invoker.Invoke(ctx, invoaction)
}

func (p *polarisInvoker) GetURL() *common.URL {
	return p.invoker.GetURL()
}

func (p *polarisInvoker) IsAvailable() bool {
	return p.invoker.IsAvailable()
}

func (p *polarisInvoker) Destroy() {
	p.invoker.Destroy()
}
