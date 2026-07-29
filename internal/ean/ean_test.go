package ean

import (
	"context"
	"encoding/json"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func testLogger(t *testing.T) *zap.Logger {
	t.Helper()
	return zap.NewNop()
}

// mockBus 内存总线，用于包级闭环测试（不连真实 MQTT/NATS）
type mockBus struct {
	mu   sync.Mutex
	subs map[string][]MessageHandler
}

func newMockBus() *mockBus {
	return &mockBus{subs: make(map[string][]MessageHandler)}
}

func (b *mockBus) Subscribe(topic string, handler MessageHandler) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.subs[topic] = append(b.subs[topic], handler)
	return nil
}

func (b *mockBus) Publish(topic string, payload []byte) error {
	b.mu.Lock()
	handlers := append([]MessageHandler(nil), b.subs[topic]...)
	// 简易通配：前缀匹配以 # 结尾的订阅
	for pattern, hs := range b.subs {
		if pattern == topic {
			continue
		}
		if matchMQTTTopic(pattern, topic) {
			handlers = append(handlers, hs...)
		}
	}
	b.mu.Unlock()
	for _, h := range handlers {
		h(topic, payload, "mock")
	}
	return nil
}

func matchMQTTTopic(pattern, topic string) bool {
	if pattern == topic {
		return true
	}
	if len(pattern) > 0 && pattern[len(pattern)-1] == '#' {
		prefix := pattern[:len(pattern)-1]
		return len(topic) >= len(prefix) && topic[:len(prefix)] == prefix
	}
	return false
}

func TestMqttTopicToNatsSubject_KeepsSlash(t *testing.T) {
	require.Equal(t, "$edgeos/discovery/agent", mqttTopicToNatsSubject("$edgeos/discovery/agent"))
	require.Equal(t, "$edgeos/event/*/status", mqttTopicToNatsSubject("$edgeos/event/+/status"))
	require.Equal(t, "$edgeos/event/>", mqttTopicToNatsSubject("$edgeos/event/#"))
	require.NotContains(t, mqttTopicToNatsSubject("$edgeos/discovery/agent"), ".")
}

func TestTransportList_UnmarshalStringOrArray(t *testing.T) {
	var a AgentDescriptor
	require.NoError(t, json.Unmarshal([]byte(`{"id":"n1","transport":"mqtt"}`), &a))
	require.Equal(t, TransportList{"mqtt"}, a.Transport)

	require.NoError(t, json.Unmarshal([]byte(`{"id":"n2","transport":["mqtt","nats"]}`), &a))
	require.Equal(t, TransportList{"mqtt", "nats"}, a.Transport)
}

func TestDiscovery_ProtocolEnvelope(t *testing.T) {
	dc := NewDiscoveryCenter(DiscoveryConfig{}, testLogger(t))

	agentPayload := []byte(`{
		"header":{"message_id":"m1","timestamp":1,"source":"edgex-node-001","message_type":"agent_descriptor","version":"2.0"},
		"body":{"agent":{"id":"edgex-node-001","kind":"device","version":"2.0.0","status":"online","transport":"mqtt","heartbeat_interval_sec":30,"metadata":{"os":"linux"}}}
	}`)
	dc.HandleAgentOnline(TopicDiscoveryAgent, agentPayload, "mqtt")

	agent, ok := dc.GetAgent("edgex-node-001")
	require.True(t, ok)
	require.Equal(t, AgentOnline, agent.Status)
	require.Equal(t, 30, agent.HeartbeatIntervalSec)
	require.Equal(t, TransportList{"mqtt"}, agent.Transport)

	capPayload := []byte(`{
		"header":{"message_id":"m2","timestamp":1,"source":"edgex-node-001","message_type":"capability_descriptor","version":"2.0"},
		"body":{"capabilities":[
			{"id":"system.diagnostics","agent_id":"edgex-node-001","description":"diag","category":"system","timeout_sec":10,"permission":"read"},
			{"id":"modbus.write","agent_id":"edgex-node-001","description":"write","category":"device","timeout_sec":5,"permission":"write"}
		]}
	}`)
	dc.HandleCapability(TopicDiscoveryCapability, capPayload, "mqtt")

	caps := dc.ListCapabilities("edgex-node-001")
	require.Len(t, caps, 2)
	require.Equal(t, 1, dc.OnlineAgentCount())

	found, ok := dc.FindCapabilityTimeout(CapabilityCategoryDevice, 0, "write")
	require.True(t, ok)
	require.Equal(t, "modbus.write", found.ID)

	offlinePayload := []byte(`{"header":{"message_type":"agent_offline","source":"edgex-node-001","version":"2.0","message_id":"m3","timestamp":2},"body":{"agent_id":"edgex-node-001","reason":"shutdown"}}`)
	dc.HandleAgentOffline(TopicDiscoveryAgentOffline, offlinePayload, "mqtt")
	agent, ok = dc.GetAgent("edgex-node-001")
	require.True(t, ok)
	require.Equal(t, AgentOffline, agent.Status)
}

func TestEvent_PreviousValueFromEnvelope(t *testing.T) {
	var got *PointChangeEvent
	ec := NewEventCenter(EventCenterConfig{
		OnPointChange: func(event *PointChangeEvent) { got = event },
		CacheSize:     16,
	}, testLogger(t))

	payload := []byte(`{
		"header":{"message_id":"e1","timestamp":1,"source":"edgex-node-001","message_type":"event","version":"2.0"},
		"body":{
			"event_type":"temperature.changed",
			"agent_id":"edgex-node-001",
			"device_id":"slave-1",
			"point_id":"temperature",
			"value":45.2,
			"previous_value":42.1,
			"timestamp":1776787200000,
			"metadata":{"quality":"good"}
		}
	}`)
	ec.HandleEvent(EventTopic("edgex-node-001"), payload, "nats")

	require.NotNil(t, got)
	require.Equal(t, 45.2, got.Value)
	require.Equal(t, 42.1, got.PreviousValue)
	require.Equal(t, "temperature", got.PointID)

	recent := ec.RecentEvents(1)
	require.Len(t, recent, 1)
	require.Equal(t, 42.1, recent[0].PreviousValue)
}

func TestInvoke_CorrelationAndReply(t *testing.T) {
	bus := newMockBus()
	orch := NewInvokeOrchestrator(InvokeConfig{
		SourceID:  "edgeos-planner",
		PublishFn: bus.Publish,
	}, testLogger(t))
	require.NoError(t, orch.RegisterReplySubscription(bus))

	// 模拟 EdgeX：收到 invoke 后回 reply
	require.NoError(t, bus.Subscribe(InvokeTopic("edgex-node-001"), func(topic string, payload []byte, transport string) {
		var msg Message
		require.NoError(t, json.Unmarshal(payload, &msg))
		require.Equal(t, "invoke_capability", msg.Header.MessageType)
		require.Equal(t, "2.0", msg.Header.Version)

		var req InvokeRequest
		require.NoError(t, json.Unmarshal(msg.Body, &req))

		respBody, err := json.Marshal(InvokeResponse{
			InvokeID: req.InvokeID,
			Status:   "completed",
			Result:   InvokeResult{Success: true, Values: map[string]interface{}{"ok": true}},
		})
		require.NoError(t, err)
		reply, err := json.Marshal(Message{
			Header: MessageHeader{
				MessageID:     "r1",
				Timestamp:     time.Now().UnixMilli(),
				Source:        "edgex-node-001",
				MessageType:   "invoke_response",
				Version:       "2.0",
				CorrelationID: req.InvokeID,
			},
			Body: respBody,
		})
		require.NoError(t, err)
		_ = bus.Publish(orch.ReplyTopic(), reply)
	}))

	call := orch.Invoke(context.Background(), "edgex-node-001", "system.diagnostics", map[string]interface{}{}, time.Second)
	require.NoError(t, call.Error)
	require.NotNil(t, call.Response)
	require.Equal(t, "completed", call.Response.Status)
	require.True(t, call.Response.Result.Success)
	require.Equal(t, 0, orch.PendingCount())
}

func TestInvoke_Timeout(t *testing.T) {
	orch := NewInvokeOrchestrator(InvokeConfig{
		SourceID: "edgeos-planner",
		PublishFn: func(topic string, payload []byte) error {
			return nil
		},
	}, testLogger(t))

	call := orch.Invoke(context.Background(), "offline-agent", "system.diagnostics", nil, 50*time.Millisecond)
	require.Error(t, call.Error)
	require.Contains(t, call.Error.Error(), "timed out")
}

func TestHeartbeat_TimeoutMarksOffline(t *testing.T) {
	dc := NewDiscoveryCenter(DiscoveryConfig{}, testLogger(t))
	dc.HandleAgentOnline(TopicDiscoveryAgent, []byte(`{"id":"n1","kind":"device","heartbeat_interval_sec":1,"metadata":{}}`), "mqtt")

	var timedOut string
	hm := NewHeartbeatMonitor(HeartbeatMonitorConfig{
		CheckInterval:     20 * time.Millisecond,
		TimeoutMultiplier: 1,
		Discovery:         dc,
		OnTimeout: func(agentID string, lastSeen time.Time, missedCount int) {
			timedOut = agentID
		},
	}, testLogger(t))

	hm.HandleHeartbeat(HeartbeatTopic("n1"), []byte(`{"agent_id":"n1","status":"online","timestamp":1,"sequence":1}`), "mqtt")
	hm.Start()
	defer hm.Stop()

	require.Eventually(t, func() bool {
		agent, ok := dc.GetAgent("n1")
		return ok && agent.Status == AgentOffline && timedOut == "n1"
	}, 2*time.Second, 20*time.Millisecond)
}

func TestGovernance_PermissionAndAudit(t *testing.T) {
	g := NewGovernance(GovernanceConfig{MaxAudit: 100}, testLogger(t))

	deny := g.CheckInvokePermission("t1", "edgex-node-001", "ai.protocol_reverse", PermissionAI)
	require.False(t, deny.Allowed)

	g.SetPolicy(&TenantPolicy{
		TenantID: "t1",
		AllowCap: []string{"ai."},
	})
	allow := g.CheckInvokePermission("t1", "edgex-node-001", "ai.protocol_reverse", PermissionAI)
	require.True(t, allow.Allowed)

	read := g.CheckInvokePermission("t1", "edgex-node-001", "system.diagnostics", PermissionRead)
	require.True(t, read.Allowed)

	g.RecordAudit("edgeos-planner", "edgex-node-001", "ai.protocol_reverse", "inv-1", "completed", "t1")
	records := g.QueryAuditRecords("edgeos-planner", "", "", 10)
	require.Len(t, records, 1)
	require.Equal(t, "inv-1", records[0].InvokeID)
}

func TestDualTransport_TopicHelpers(t *testing.T) {
	require.Equal(t, "$edgeos/invoke/edgex-node-001", InvokeTopic("edgex-node-001"))
	require.Equal(t, "$edgeos/reply/edgeos-planner", ReplyTopic("edgeos-planner"))
	require.Equal(t, "$edgeos/heartbeat/n1", HeartbeatTopic("n1"))
	require.Equal(t, "$edgeos/event/n1", EventTopic("n1"))
}
