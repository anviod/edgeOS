package ean

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.etcd.io/bbolt"
	"go.uber.org/zap"

	"github.com/anviod/edgeOS/internal/services"
)

func TestAttachRegistryMirror_MirrorsAgentOnlineOffline(t *testing.T) {
	f, err := os.CreateTemp("", "ean_registry_mirror_*.db")
	require.NoError(t, err)
	_ = f.Close()
	defer os.Remove(f.Name())

	db, err := bbolt.Open(f.Name(), 0600, nil)
	require.NoError(t, err)
	defer db.Close()

	registry := services.NewRegistryService(db)
	bus, err := NewBus(BusConfig{PlannerID: "planner-test"}, zap.NewNop())
	require.NoError(t, err)

	// 先模拟 wireCallbacks 已设置的心跳钩子
	hbTouched := false
	bus.Discovery.onAgentOnline = func(agent *AgentDescriptor) { hbTouched = true }
	offlineCalled := false
	bus.Discovery.onAgentOffline = func(agentID, reason string) { offlineCalled = true }

	bus.AttachRegistryMirror(registry)

	bus.Discovery.HandleAgentOnline("$edgeos/discovery/agent", []byte(`{
		"header":{"message_type":"agent_online","source":"edgex-node-001"},
		"body":{"id":"edgex-node-001","kind":"device","status":"online","transport":["nats"],"heartbeat_interval_sec":60}
	}`), "nats")

	assert.True(t, hbTouched, "previous onAgentOnline hook should still run")
	node, err := registry.GetNode("edgex-node-001")
	require.NoError(t, err)
	assert.Equal(t, "online", node.Status)
	assert.Contains(t, node.Protocol, "ean")

	offlinePayload := []byte(`{"header":{"message_type":"agent_offline","source":"edgex-node-001","version":"2.0","message_id":"m3","timestamp":2},"body":{"agent_id":"edgex-node-001","reason":"shutdown"}}`)
	bus.Discovery.HandleAgentOffline("$edgeos/discovery/agent/offline", offlinePayload, "nats")
	assert.True(t, offlineCalled, "previous onAgentOffline hook should still run")

	// Phase 4 (OS-P4): Agent 下线 → 节点从 V1 注册表删除（避免 transient 残留）
	_, err = registry.GetNode("edgex-node-001")
	assert.Error(t, err, "node should be removed from registry when agent goes offline")
}

// TestAttachRegistryMirror_SkipsTransientAgent 验证 transient/v1-bridge Agent 不污染 /api/nodes（Phase 4 OS-P4）。
func TestAttachRegistryMirror_SkipsTransientAgent(t *testing.T) {
	f, err := os.CreateTemp("", "ean_registry_mirror_skip_*.db")
	require.NoError(t, err)
	_ = f.Close()
	defer os.Remove(f.Name())

	db, err := bbolt.Open(f.Name(), 0600, nil)
	require.NoError(t, err)
	defer db.Close()

	registry := services.NewRegistryService(db)
	bus, err := NewBus(BusConfig{PlannerID: "planner-test"}, zap.NewNop())
	require.NoError(t, err)

	bus.AttachRegistryMirror(registry)

	// v1-bridge Agent（非原生 EAN）：不应镜像为 V1 节点
	bus.Discovery.HandleAgentOnline("$edgeos/discovery/agent", []byte(`{
		"header":{"message_type":"agent_online","source":"ean-it-432700"},
		"body":{"id":"ean-it-432700","kind":"edgex-gateway","status":"online","transport":["mqtt"],"heartbeat_interval_sec":30}
	}`), "v1-bridge")

	_, err = registry.GetNode("ean-it-432700")
	assert.Error(t, err, "transient/v1-bridge agent should not be mirrored to V1 nodes")

	// 北向原生 EAN Agent（nats/mqtt 传输 → native-ean source）应镜像
	bus.Discovery.HandleAgentOnline("$edgeos/discovery/agent", []byte(`{
		"header":{"message_type":"agent_online","source":"edgex-node-001"},
		"body":{"id":"edgex-node-001","kind":"device","status":"online","transport":["nats"],"heartbeat_interval_sec":60}
	}`), "nats")

	node, err := registry.GetNode("edgex-node-001")
	require.NoError(t, err)
	assert.Equal(t, "online", node.Status)
}
