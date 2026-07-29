package ean

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"testing"
	"time"

	"go.uber.org/zap"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ============================================================
// 集成测试: Bus 级别模拟
// 使用 mock Transport + DualTransport 模拟完整消息流
// ============================================================

// busSuite 集成测试套件
type busSuite struct {
	dt         *DualTransport
	dc         *DiscoveryCenter
	hm         *HeartbeatMonitor
	io         *InvokeOrchestrator
	ec         *EventCenter
	logger     *zap.Logger
}

func newBusSuite(t *testing.T) *busSuite {
	t.Helper()

	dt := NewDualTransport(zap.NewNop())

	dc := NewDiscoveryCenter(DiscoveryConfig{}, zap.NewNop())

	hm := NewHeartbeatMonitor(HeartbeatMonitorConfig{
		CheckInterval:    50 * time.Millisecond,
		TimeoutMultiplier: 2,
		Discovery:        dc,
		OnTimeout: func(id string, lastSeen time.Time, missed int) {
			t.Logf("timeout: agent=%s lastSeen=%v missed=%d", id, lastSeen, missed)
		},
	}, zap.NewNop())

	io := NewInvokeOrchestrator(InvokeConfig{
		SourceID:  "test-planner",
		PublishFn: nil, // 在具体测试中注入
	}, zap.NewNop())

	ec := NewEventCenter(EventCenterConfig{CacheSize: 64}, zap.NewNop())

	return &busSuite{
		dt:     dt,
		dc:     dc,
		hm:     hm,
		io:     io,
		ec:     ec,
		logger: zap.NewNop(),
	}
}

// ---------- TestNewBusWithoutTransport ----------

func TestNewBusWithoutTransport(t *testing.T) {
	dt := NewDualTransport(zap.NewNop())

	// 不添加任何 transport 时，Bus 正常创建
	assert.NotNil(t, dt)
	assert.Empty(t, dt.Transports())
	assert.False(t, dt.IsConnected())
	assert.Empty(t, dt.ConnectedNames())

	// Subscribe/Publish 无 transport 不报错
	err := dt.Subscribe("$edgeos/test", func(string, []byte, string) {})
	assert.NoError(t, err) // 无 transport 时 Subscribe 返回 nil

	err = dt.Publish("$edgeos/test", []byte("hello"))
	assert.NoError(t, err)
}

// ---------- TestBusDiscoveryFlow ----------

func TestBusDiscoveryFlow(t *testing.T) {
	suite := newBusSuite(t)

	// Step 1: Agent 上线
	agent := AgentDescriptor{
		ID:                   "edgex-node-001",
		Kind:                 "edgex",
		Version:              "2.3",
		Status:               AgentOnline,
		HeartbeatIntervalSec: 10,
		Transport:            []string{"mqtt"},
	}
	agentPayload, _ := json.Marshal(agent)
	suite.dc.HandleAgentOnline(TopicDiscoveryAgent, agentPayload, "mqtt")

	// 验证 Agent 索引
	a, ok := suite.dc.GetAgent("edgex-node-001")
	require.True(t, ok)
	assert.Equal(t, AgentOnline, a.Status)
	assert.Equal(t, "edgex", a.Kind)
	assert.Equal(t, 1, suite.dc.OnlineAgentCount())

	// Step 2: Capability 注册
	cap1 := CapabilityDescriptor{
		ID:          "modbus-tcp.read_points",
		AgentID:     "edgex-node-001",
		Description: "Read Modbus TCP register points",
		Category:    CapabilityCategoryDriver,
		TimeoutSec:  30,
		Permission:  PermissionRead,
	}
	cap1Payload, _ := json.Marshal(cap1)
	suite.dc.HandleCapability(TopicDiscoveryCapability, cap1Payload, "mqtt")

	cap2 := CapabilityDescriptor{
		ID:          "modbus-tcp.write_point",
		AgentID:     "edgex-node-001",
		Description: "Write Modbus TCP register point",
		Category:    CapabilityCategoryDriver,
		TimeoutSec:  15,
		Permission:  PermissionWrite,
	}
	cap2Payload, _ := json.Marshal(cap2)
	suite.dc.HandleCapability(TopicDiscoveryCapability, cap2Payload, "mqtt")

	// 验证 capability 索引
	caps := suite.dc.GetCapabilitiesByAgent("edgex-node-001")
	assert.Len(t, caps, 2)

	c1, ok := suite.dc.GetCapability("modbus-tcp.read_points")
	require.True(t, ok)
	assert.Equal(t, CapabilityCategoryDriver, c1.Category)

	// Step 3: FindCapabilityTimeout
	found, ok := suite.dc.FindCapabilityTimeout(CapabilityCategoryDriver, 20, "Modbus")
	require.True(t, ok)
	assert.Equal(t, "modbus-tcp.read_points", found.ID)
}

// ---------- TestBusInvokeFlow ----------

func TestBusInvokeFlow(t *testing.T) {
	suite := newBusSuite(t)

	// 注册 agent
	agent := AgentDescriptor{
		ID:                   "edgex-node-001",
		Kind:                 "edgex",
		HeartbeatIntervalSec: 10,
	}
	agentPayload, _ := json.Marshal(agent)
	suite.dc.HandleAgentOnline(TopicDiscoveryAgent, agentPayload, "mqtt")

	// 注册 capability
	cap := CapabilityDescriptor{
		ID:          "modbus.read_points",
		AgentID:     "edgex-node-001",
		Category:    CapabilityCategoryDriver,
		TimeoutSec:  30,
	}
	capPayload, _ := json.Marshal(cap)
	suite.dc.HandleCapability(TopicDiscoveryCapability, capPayload, "mqtt")

	// 设置 invoke orchestrator 的 publishFn 捕获发布
	var capturedTopic string
	var capturedPayload []byte
	publishDone := make(chan struct{}, 1)

	io := NewInvokeOrchestrator(InvokeConfig{
		SourceID: "test-planner",
		PublishFn: func(topic string, payload []byte) error {
			capturedTopic = topic
			capturedPayload = payload
			publishDone <- struct{}{}
			return nil
		},
	}, zap.NewNop())

	// 发起 Invoke
	resultCh := make(chan *InvokeCall, 1)
	go func() {
		call := io.Invoke(context.Background(), "edgex-node-001", "modbus.read_points",
			map[string]interface{}{"device": "PLC-1", "point": "Temperature"},
			5*time.Second)
		resultCh <- call
	}()

	// 等待 publish
	select {
	case <-publishDone:
	case <-time.After(time.Second):
		t.Fatal("invoke publish did not happen")
	}

	// 验证 topic
	assert.Equal(t, "$edgeos/invoke/edgex-node-001", capturedTopic)

	// 解析 invoke_id
	var msg Message
	require.NoError(t, json.Unmarshal(capturedPayload, &msg))
	var req InvokeRequest
	require.NoError(t, json.Unmarshal(msg.Body, &req))
	invokeID := req.InvokeID
	assert.NotEmpty(t, invokeID)

	// 模拟 EdgeX 回复
	resp := InvokeResponse{
		InvokeID: invokeID,
		Status:   "success",
		Result: InvokeResult{
			Success: true,
			Values: map[string]interface{}{
				"Temperature": float64(25.5),
			},
		},
	}
	respBody, _ := json.Marshal(resp)
	replyMsg := Message{
		Header: MessageHeader{
			MessageID:      "reply-001",
			Timestamp:      time.Now().UnixMilli(),
			Source:         "edgex-node-001",
			MessageType:     "invoke_reply",
			Version:        "2.0",
			CorrelationID:  invokeID,
		},
		Body: respBody,
	}
	replyPayload, _ := json.Marshal(replyMsg)
	io.HandleReply("$edgeos/reply/test-planner", replyPayload, "mqtt")

	// 验证结果
	select {
	case call := <-resultCh:
		require.NoError(t, call.Error)
		require.NotNil(t, call.Response)
		assert.Equal(t, "success", call.Response.Status)
		assert.True(t, call.Response.Result.Success)
	case <-time.After(time.Second):
		t.Fatal("invoke did not complete")
	}
}

// ---------- TestBusEventFlow ----------

func TestBusEventFlow(t *testing.T) {
	var received *PointChangeEvent
	ec := NewEventCenter(EventCenterConfig{
		CacheSize: 16,
		OnPointChange: func(e *PointChangeEvent) {
			received = e
		},
	}, zap.NewNop())

	// 模拟含 previous_value 的事件
	event := map[string]interface{}{
		"event_type":     "point.changed",
		"agent_id":        "edgex-node-001",
		"device_id":       "PLC-1",
		"point_id":        "Temperature",
		"value":           float64(26.0),
		"previous_value":  float64(25.5),
		"timestamp":       time.Now().UnixMilli(),
	}
	payload, _ := json.Marshal(event)

	ec.HandleEvent("$edgeos/event/edgex-node-001", payload, "mqtt")

	// 验证解析
	require.NotNil(t, received)
	assert.Equal(t, "edgex-node-001", received.AgentID)
	assert.Equal(t, "PLC-1", received.DeviceID)
	assert.Equal(t, "Temperature", received.PointID)
	assert.Equal(t, float64(26.0), received.Value)
	assert.Equal(t, float64(25.5), received.PreviousValue)

	// 验证缓存
	recent := ec.RecentEvents(1)
	require.Len(t, recent, 1)
	assert.Equal(t, float64(26.0), recent[0].Value)
}

// ---------- TestBusHeartbeatTimeout ----------

func TestBusHeartbeatTimeout(t *testing.T) {
	dc := NewDiscoveryCenter(DiscoveryConfig{}, zap.NewNop())

	timeoutCh := make(chan string, 1)
	hm := NewHeartbeatMonitor(HeartbeatMonitorConfig{
		CheckInterval:    50 * time.Millisecond,
		TimeoutMultiplier: 1, // 1x interval = 快速触发
		Discovery:        dc,
		OnTimeout: func(id string, _ time.Time, _ int) {
			timeoutCh <- id
		},
	}, zap.NewNop())

	// Step 1: Agent 上线
	agent := AgentDescriptor{
		ID:                   "edgex-node-001",
		Kind:                 "edgex",
		HeartbeatIntervalSec: 10,
		Metadata:             make(map[string]string),
	}
	agentPayload, _ := json.Marshal(agent)
	dc.HandleAgentOnline(TopicDiscoveryAgent, agentPayload, "mqtt")
	assert.Equal(t, AgentOnline, dc.agents["edgex-node-001"].Status)

	// Step 2: 发送心跳
	hb := HeartbeatPayload{
		AgentID:   "edgex-node-001",
		Status:    "alive",
		Timestamp: time.Now().Unix(),
		Sequence:  1,
	}
	hbPayload, _ := json.Marshal(hb)
	hm.HandleHeartbeat("$edgeos/heartbeat/edgex-node-001", hbPayload, "mqtt")

	state, ok := hm.GetAgentHeartbeat("edgex-node-001")
	require.True(t, ok)
	assert.Equal(t, 10, state.IntervalSec) // 从 discovery 获取

	// Step 3: 手动设置 lastSeen 为过去时间，让下次检查超时
	hm.mu.Lock()
	if s, exists := hm.agents["edgex-node-001"]; exists {
		s.LastSeen = time.Now().Add(-30 * time.Second) // 30s > 10s * 1x
		s.IntervalSec = 10
	}
	hm.mu.Unlock()

	// Step 4: 启动检查循环，等超时
	hm.Start()
	defer hm.Stop()

	// 等待超时回调
	select {
	case id := <-timeoutCh:
		assert.Equal(t, "edgex-node-001", id)
	case <-time.After(2 * time.Second):
		t.Fatal("heartbeat timeout not triggered")
	}

	// Step 5: 验证 Agent 被标记为 offline
	a, ok := dc.GetAgent("edgex-node-001")
	require.True(t, ok)
	assert.Equal(t, AgentOffline, a.Status)
}

// ---------- TestBusConcurrentSafety ----------

func TestBusConcurrentSafety(t *testing.T) {
	dc := NewDiscoveryCenter(DiscoveryConfig{
		OnAgentOnline:  func(*AgentDescriptor) {},
		OnAgentOffline: func(string, string) {},
		OnCapability:   func(*CapabilityDescriptor) {},
	}, zap.NewNop())

	ec := NewEventCenter(EventCenterConfig{CacheSize: 64}, zap.NewNop())

	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()

			agent := AgentDescriptor{ID: fmt.Sprintf("agent-%d", idx), Kind: "edgex"}
			payload, _ := json.Marshal(agent)
			dc.HandleAgentOnline(TopicDiscoveryAgent, payload, "mqtt")

			cap := CapabilityDescriptor{ID: fmt.Sprintf("cap-%d", idx), AgentID: fmt.Sprintf("agent-%d", idx), Category: CapabilityCategoryDriver}
			capPayload, _ := json.Marshal(cap)
			dc.HandleCapability(TopicDiscoveryCapability, capPayload, "mqtt")

			event := map[string]interface{}{
				"event_type": "point.changed",
				"agent_id":   fmt.Sprintf("agent-%d", idx),
				"value":      float64(idx),
				"timestamp":  time.Now().UnixMilli(),
			}
			eventPayload, _ := json.Marshal(event)
			ec.HandleEvent("$edgeos/event/agent", eventPayload, "mqtt")

			_ = dc.ListAgents()
			_ = dc.ListCapabilities("", )
			_ = ec.RecentEvents(10)
		}(i)
	}
	wg.Wait()

	// 无 panic 即通过
	assert.Equal(t, 50, len(dc.ListAgents()))
}
