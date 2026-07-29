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

	node2, err := registry.GetNode("edgex-node-001")
	require.NoError(t, err)
	assert.Equal(t, "offline", node2.Status)
}
