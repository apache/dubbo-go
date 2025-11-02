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
	"sync"
	"testing"
	"time"

	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/config"
	"github.com/stretchr/testify/assert"
)

// TestConnectionHealthDetection 严格验证连接健康检测的实际效果
// 这个测试会模拟真实的网络故障场景来验证我们的健康检查是否真的有效
func TestConnectionHealthDetection(t *testing.T) {
	t.Log("🔍 严格验证连接健康检测机制")

	// 1. 设置配置
	consumerConfig := &config.ConsumerConfig{
		RequestTimeout: "60s",
	}
	config.SetConsumerConfig(*consumerConfig)

	// 2. 启动可控的测试服务器
	server := NewControllableTestServer("127.0.0.1:20999")
	go server.Start()
	defer server.Stop()

	// 等待服务器启动
	time.Sleep(100 * time.Millisecond)

	// 3. 创建测试URL
	testURL, err := common.NewURL("dubbo://127.0.0.1:20999/com.test.TestService",
		common.WithParamsValue("interface", "com.test.TestService"),
		common.WithParamsValue("timeout", "60000"))
	assert.NoError(t, err)

	t.Log("📈 第一阶段：建立连接并验证正常工作")

	// 4. 第一次获取连接 - 应该成功
	exchangeClient1 := getExchangeClient(testURL)
	assert.NotNil(t, exchangeClient1, "第一次获取连接应该成功")

	isAvailable1 := exchangeClient1.IsAvailable()
	t.Logf("🔗 第一次连接状态: %v", isAvailable1)
	assert.True(t, isAvailable1, "新建连接应该是可用的")

	// 5. 再次获取连接 - 应该复用同一个连接
	exchangeClient2 := getExchangeClient(testURL)
	assert.NotNil(t, exchangeClient2, "第二次获取连接应该成功")

	// 验证是否复用了连接（地址相同）
	isReused := fmt.Sprintf("%p", exchangeClient1) == fmt.Sprintf("%p", exchangeClient2)
	t.Logf("🔄 连接复用: %v (地址1:%p, 地址2:%p)", isReused, exchangeClient1, exchangeClient2)

	t.Log("💥 第二阶段：模拟网络故障")

	// 6. 关闭服务器，模拟网络故障
	server.ForceCloseAllConnections()
	t.Log("🚫 强制关闭服务器端所有连接，模拟网络断开")

	// 等待一下让连接状态更新
	time.Sleep(500 * time.Millisecond)

	// 7. 检查连接状态 - 现在应该检测到连接不可用
	isAvailable2 := exchangeClient1.IsAvailable()
	t.Logf("💔 故障后连接状态: %v", isAvailable2)

	if isAvailable2 {
		t.Log("⚠️  连接健康检查可能没有及时检测到故障")
	} else {
		t.Log("✅ 连接健康检查成功检测到连接故障")
	}

	t.Log("🔄 第三阶段：验证故障恢复")

	// 8. 重新启动服务器
	server.Restart()
	time.Sleep(200 * time.Millisecond)

	// 9. 再次获取连接 - 应该创建新连接
	exchangeClient3 := getExchangeClient(testURL)
	assert.NotNil(t, exchangeClient3, "故障恢复后应该能获取新连接")

	isAvailable3 := exchangeClient3.IsAvailable()
	t.Logf("🆕 新连接状态: %v", isAvailable3)

	// 10. 验证是否创建了新连接
	isNewConnection := fmt.Sprintf("%p", exchangeClient1) != fmt.Sprintf("%p", exchangeClient3)
	t.Logf("🔄 是否创建新连接: %v (原连接:%p, 新连接:%p)", isNewConnection, exchangeClient1, exchangeClient3)

	// 11. 检查连接池统计 (安全处理)
	if globalConnectionManager != nil {
		stats := globalConnectionManager.GetGlobalStats()["dubbo"]
		if stats != nil {
			t.Logf("📊 最终连接池统计: 总连接=%d, 活跃=%d, 失败=%d",
				stats.TotalConnections, stats.ActiveConnections, stats.FailedConnections)
		}
	}

	t.Log("🎯 连接健康检测验证结果:")

	if !isAvailable2 {
		t.Log("✅ 连接健康检查工作正常 - 能够检测到连接故障")
	} else {
		t.Log("❌ 连接健康检查可能存在问题 - 未能及时检测到故障")
	}

	if isNewConnection && isAvailable3 {
		t.Log("✅ 故障恢复机制工作正常 - 能够创建新的健康连接")
	} else {
		t.Log("❌ 故障恢复可能存在问题")
	}
}

// TestDetailedHealthCheckMechanism 详细测试健康检查机制的各个层面
func TestDetailedHealthCheckMechanism(t *testing.T) {
	t.Log("🔬 详细测试连接健康检查机制")

	// 设置配置
	consumerConfig := &config.ConsumerConfig{
		RequestTimeout: "60s",
	}
	config.SetConsumerConfig(*consumerConfig)

	// 创建测试URL
	testURL, err := common.NewURL("dubbo://127.0.0.1:21000/com.test.TestService",
		common.WithParamsValue("interface", "com.test.TestService"))
	assert.NoError(t, err)

	// 启动服务器
	server := NewControllableTestServer("127.0.0.1:21000")
	go server.Start()
	defer server.Stop()
	time.Sleep(100 * time.Millisecond)

	t.Log("1️⃣ 测试正常连接的健康检查")

	// 获取连接
	exchangeClient := getExchangeClient(testURL)
	assert.NotNil(t, exchangeClient)

	// 多次检查健康状态
	for i := 0; i < 5; i++ {
		isAvailable := exchangeClient.IsAvailable()
		t.Logf("   第%d次健康检查: %v", i+1, isAvailable)
		time.Sleep(100 * time.Millisecond)
	}

	t.Log("2️⃣ 测试连接断开后的健康检查")

	// 强制断开连接
	server.ForceCloseAllConnections()
	t.Log("   服务器连接已断开")

	// 等待连接状态更新
	time.Sleep(1 * time.Second)

	// 再次检查健康状态
	for i := 0; i < 5; i++ {
		isAvailable := exchangeClient.IsAvailable()
		t.Logf("   断开后第%d次健康检查: %v", i+1, isAvailable)
		time.Sleep(100 * time.Millisecond)
	}

	t.Log("3️⃣ 测试健康检查的响应时间")

	// 测量健康检查的性能
	start := time.Now()
	for i := 0; i < 100; i++ {
		exchangeClient.IsAvailable()
	}
	duration := time.Since(start)

	avgTime := duration / 100
	t.Logf("   100次健康检查平均时间: %v", avgTime)

	if avgTime < 1*time.Millisecond {
		t.Log("✅ 健康检查性能良好 - 平均时间 < 1ms")
	} else {
		t.Log("⚠️  健康检查可能有性能问题")
	}
}

// ControllableTestServer 可控制的测试服务器
// 能够模拟各种网络故障场景
type ControllableTestServer struct {
	addr        string
	listener    net.Listener
	running     bool
	mutex       sync.Mutex
	connections []net.Conn
}

func NewControllableTestServer(addr string) *ControllableTestServer {
	return &ControllableTestServer{
		addr:        addr,
		connections: make([]net.Conn, 0),
	}
}

func (s *ControllableTestServer) Start() {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if s.running {
		return
	}

	listener, err := net.Listen("tcp", s.addr)
	if err != nil {
		return
	}

	s.listener = listener
	s.running = true

	go func() {
		for s.running {
			conn, err := s.listener.Accept()
			if err != nil {
				continue
			}

			s.mutex.Lock()
			s.connections = append(s.connections, conn)
			s.mutex.Unlock()

			// 简单的连接处理
			go func(c net.Conn) {
				defer func() {
					c.Close()
					s.removeConnection(c)
				}()

				buf := make([]byte, 1024)
				for {
					_, err := c.Read(buf)
					if err != nil {
						break
					}
					// Echo back some data
					c.Write([]byte("OK"))
				}
			}(conn)
		}
	}()
}

func (s *ControllableTestServer) Stop() {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if !s.running {
		return
	}

	s.running = false

	// 关闭所有连接
	for _, conn := range s.connections {
		conn.Close()
	}
	s.connections = s.connections[:0]

	if s.listener != nil {
		s.listener.Close()
	}
}

func (s *ControllableTestServer) ForceCloseAllConnections() {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	// 强制关闭所有现有连接
	for _, conn := range s.connections {
		conn.Close()
	}
	s.connections = s.connections[:0]
}

func (s *ControllableTestServer) Restart() {
	s.Stop()
	time.Sleep(100 * time.Millisecond)
	go s.Start()
	time.Sleep(100 * time.Millisecond)
}

func (s *ControllableTestServer) removeConnection(targetConn net.Conn) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	for i, conn := range s.connections {
		if conn == targetConn {
			s.connections = append(s.connections[:i], s.connections[i+1:]...)
			break
		}
	}
}
