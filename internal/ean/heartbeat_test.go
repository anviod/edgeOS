package ean

import (
	"encoding/json"
	"testing"
	"time"

	"go.uber.org/zap"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestHeartbeatMonitor(t *testing.T, opts ...func(*HeartbeatMonitorConfig)) *HeartbeatMonitor {
	t.Helper()
	cfg := HeartbeatMonitorConfig{
		CheckInterval:     100 * time.Millisecond,
		TimeoutMultiplier:  3,
		OnTimeout:         func(string, time.Time, int) {},
	}
	for _, opt := range opts {
		opt(&cfg)
	}
	return NewHeartbeatMonitor(cfg, zap.NewNop())
}

// ---------- TestHandleHeartbeat ----------

func TestHandleHeartbeat(t *testing.T) {
	hm := newTestHeartbeatMonitor(t)

	hb := HeartbeatPayload{
		AgentID:   "agent-001",
		Status:    "alive",
		Timestamp: time.Now().Unix(),
		Sequence:  1,
	}
	payload, _ := json.Marshal(hb)

	hm.HandleHeartbeat("$edgeos/heartbeat/agent-001", payload, "mqtt")

	// 验证 Agent 已被跟踪
	state, ok := hm.GetAgentHeartbeat("agent-001")
	require.True(t, ok)
	assert.Equal(t, "agent-001", state.AgentID)
	assert.Equal(t, 1, state.Sequence)
	assert.Equal(t, 0, state.MissedCount)
	assert.False(t, state.LastSeen.IsZero())
}

func TestHandleHeartbeat_InvalidJSON(t *testing.T) {
	hm := newTestHeartbeatMonitor(t)
	hm.HandleHeartbeat("$edgeos/heartbeat/agent-001", []byte("invalid"), "mqtt")
	assert.Equal(t, 0, hm.TrackedCount())
}

func TestHandleHeartbeat_UpdateSequence(t *testing.T) {
	hm := newTestHeartbeatMonitor(t)

	// 第一次心跳
	hb1 := HeartbeatPayload{AgentID: "agent-001", Status: "alive", Sequence: 1}
	p1, _ := json.Marshal(hb1)
	hm.HandleHeartbeat("$edgeos/heartbeat/agent-001", p1, "mqtt")

	// 第二次心跳，更新序列号
	time.Sleep(5 * time.Millisecond) // 确保 lastSeen 变化
	hb2 := HeartbeatPayload{AgentID: "agent-001", Status: "alive", Sequence: 2}
	p2, _ := json.Marshal(hb2)
	hm.HandleHeartbeat("$edgeos/heartbeat/agent-001", p2, "mqtt")

	state, ok := hm.GetAgentHeartbeat("agent-001")
	require.True(t, ok)
	assert.Equal(t, 2, state.Sequence)
	assert.Equal(t, 0, state.MissedCount) // 收到心跳后重置
}

// ---------- TestCheckTimeouts ----------

func TestCheckTimeouts(t *testing.T) {
	timeoutCalled := false
	var timedOutAgentID string
	hm := newTestHeartbeatMonitor(t, func(cfg *HeartbeatMonitorConfig) {
		cfg.CheckInterval = 50 * time.Millisecond
		cfg.TimeoutMultiplier = 1 // 1x interval 即超时
		cfg.OnTimeout = func(id string, _ time.Time, _ int) {
			timedOutAgentID = id
			timeoutCalled = true
		}
	})

	// 手动插入一个 lastSeen 在过去的 agent
	hm.mu.Lock()
	hm.agents["agent-001"] = &AgentHeartbeat{
		AgentID:     "agent-001",
		LastSeen:    time.Now().Add(-10 * time.Second),
		Sequence:    5,
		MissedCount: 0,
		IntervalSec: 5, // 5s * 1x = 5s timeout
	}
	hm.mu.Unlock()

	// 手动触发 checkTimeouts
	hm.checkTimeouts(time.Now())

	assert.True(t, timeoutCalled, "timeout callback should have been called")
	assert.Equal(t, "agent-001", timedOutAgentID)

	// agent 应被移除
	_, ok := hm.GetAgentHeartbeat("agent-001")
	assert.False(t, ok)
}

func TestCheckTimeouts_NoTimeout(t *testing.T) {
	hm := newTestHeartbeatMonitor(t, func(cfg *HeartbeatMonitorConfig) {
		cfg.TimeoutMultiplier = 3
		cfg.OnTimeout = func(id string, _ time.Time, _ int) {
			t.Errorf("timeout should not have been called for %s", id)
		}
	})

	// 插入刚收到心跳的 agent
	hm.mu.Lock()
	hm.agents["agent-001"] = &AgentHeartbeat{
		AgentID:     "agent-001",
		LastSeen:    time.Now(),
		Sequence:    1,
		MissedCount: 0,
		IntervalSec: 10, // 10s * 3x = 30s timeout
	}
	hm.mu.Unlock()

	hm.checkTimeouts(time.Now())

	// agent 仍在跟踪列表
	_, ok := hm.GetAgentHeartbeat("agent-001")
	assert.True(t, ok)
}

// ---------- TestRemoveAgent ----------

func TestRemoveAgent(t *testing.T) {
	hm := newTestHeartbeatMonitor(t)

	hb := HeartbeatPayload{AgentID: "agent-001", Sequence: 1}
	payload, _ := json.Marshal(hb)
	hm.HandleHeartbeat("$edgeos/heartbeat/agent-001", payload, "mqtt")
	assert.Equal(t, 1, hm.TrackedCount())

	hm.RemoveAgentTracking("agent-001")
	assert.Equal(t, 0, hm.TrackedCount())

	_, ok := hm.GetAgentHeartbeat("agent-001")
	assert.False(t, ok)
}

func TestRemoveAgent_Nonexistent(t *testing.T) {
	hm := newTestHeartbeatMonitor(t)
	// 移除不存在的 agent 不应 panic
	hm.RemoveAgentTracking("nonexistent")
	assert.Equal(t, 0, hm.TrackedCount())
}

// ---------- TestUpdateAgentInterval ----------

func TestUpdateAgentInterval(t *testing.T) {
	hm := newTestHeartbeatMonitor(t)

	hb := HeartbeatPayload{AgentID: "agent-001", Sequence: 1}
	payload, _ := json.Marshal(hb)
	hm.HandleHeartbeat("$edgeos/heartbeat/agent-001", payload, "mqtt")

	state, _ := hm.GetAgentHeartbeat("agent-001")
	assert.Equal(t, 10, state.IntervalSec) // 默认值

	hm.UpdateInterval("agent-001", 30)

	state, _ = hm.GetAgentHeartbeat("agent-001")
	assert.Equal(t, 30, state.IntervalSec)
}

func TestUpdateAgentInterval_Nonexistent(t *testing.T) {
	hm := newTestHeartbeatMonitor(t)
	// 更新不存在的 agent 不应 panic
	hm.UpdateInterval("nonexistent", 30)
}

// ---------- TestAllAgentHeartbeats ----------

func TestAllAgentHeartbeats(t *testing.T) {
	hm := newTestHeartbeatMonitor(t)

	for _, id := range []string{"agent-001", "agent-002", "agent-003"} {
		hb := HeartbeatPayload{AgentID: id, Sequence: 1}
		payload, _ := json.Marshal(hb)
		hm.HandleHeartbeat("$edgeos/heartbeat/"+id, payload, "mqtt")
	}

	all := hm.AllAgentHeartbeats()
	assert.Len(t, all, 3)
}

func TestTrackedCount(t *testing.T) {
	hm := newTestHeartbeatMonitor(t)
	assert.Equal(t, 0, hm.TrackedCount())

	hb := HeartbeatPayload{AgentID: "agent-001", Sequence: 1}
	payload, _ := json.Marshal(hb)
	hm.HandleHeartbeat("$edgeos/heartbeat/agent-001", payload, "mqtt")
	assert.Equal(t, 1, hm.TrackedCount())
}

// ---------- TestHeartbeatWithDiscovery ----------

func TestHeartbeatWithDiscovery(t *testing.T) {
	dc := NewDiscoveryCenter(DiscoveryConfig{}, zap.NewNop())

	hm := newTestHeartbeatMonitor(t, func(cfg *HeartbeatMonitorConfig) {
		cfg.Discovery = dc
	})

	// Agent 先上线
	agent := AgentDescriptor{
		ID:                   "agent-001",
		Kind:                 "edgex",
		HeartbeatIntervalSec: 15,
		Metadata:             make(map[string]string),
	}
	onlinePayload, _ := json.Marshal(agent)
	dc.HandleAgentOnline(TopicDiscoveryAgent, onlinePayload, "mqtt")

	// 发送心跳
	hb := HeartbeatPayload{AgentID: "agent-001", Sequence: 1}
	hbPayload, _ := json.Marshal(hb)
	hm.HandleHeartbeat("$edgeos/heartbeat/agent-001", hbPayload, "mqtt")

	state, ok := hm.GetAgentHeartbeat("agent-001")
	require.True(t, ok)
	assert.Equal(t, 15, state.IntervalSec) // 从 discovery 获取

	// 验证 metadata 更新
	a, _ := dc.GetAgent("agent-001")
	require.NotNil(t, a.Metadata)
	_, hasLastSeen := a.Metadata["last_seen"]
	assert.True(t, hasLastSeen)
}
