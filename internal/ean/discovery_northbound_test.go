package ean

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// EdgeX 北向 EAN Runtime（ean_bridge.go + DiscoveryPublisher）线格式 fixtures。
// 信封形态对齐 capability.NewEnvelope / CapabilityDescriptorBody / DiscoveryResponseBody。

func TestDecodeCapabilities_EdgeXNorthboundEnvelope(t *testing.T) {
	// 模拟 EdgeX PublishCapabilityDescriptor → $edgeos/discovery/capability
	payload := []byte(`{
		"header":{
			"message_id":"msg-cap-001",
			"timestamp":1785229700000,
			"source":"edgex-node-001",
			"message_type":"capability_descriptor",
			"version":"2.0"
		},
		"body":{
			"capabilities":[
				{"id":"system.diagnostics","agent_id":"edgex-node-001","description":"系统诊断","category":"system","timeout_sec":10,"permission":"read"},
				{"id":"ai.protocol_reverse","agent_id":"edgex-node-001","description":"AI协议逆向","category":"ai","timeout_sec":60,"permission":"admin"},
				{"id":"modbus_tcp.scan_devices","agent_id":"edgex-node-001","description":"扫描设备","category":"device","timeout_sec":30,"permission":"write",
					"input_schema":{"type":"object","required":["channel_id"],"properties":{"channel_id":{"type":"string","description":"通道ID"}}}}
			]
		}
	}`)

	caps, err := decodeCapabilities(payload)
	require.NoError(t, err)
	require.Len(t, caps, 3)
	assert.Equal(t, "system.diagnostics", caps[0].ID)
	assert.Equal(t, "ai.protocol_reverse", caps[1].ID)
	assert.Equal(t, "modbus_tcp.scan_devices", caps[2].ID)
}

func TestDiscovery_IndexNativeAndPurgeV1Bridge(t *testing.T) {
	dc := NewDiscoveryCenter(DiscoveryConfig{}, testLogger(t))

	// 先由 V1 Bridge 写入合成 Cap
	v1Cap := CapabilityDescriptor{
		ID:       "edgex-node-001/dev-1/read-write",
		AgentID:  "edgex-node-001",
		Category: CapabilityCategoryDevice,
		Permission: "write",
		TimeoutSec: 30,
	}
	raw, _ := json.Marshal(v1Cap)
	dc.HandleCapability(TopicDiscoveryCapability, raw, "v1-bridge")
	require.True(t, func() bool { _, ok := dc.GetCapability(v1Cap.ID); return ok }())

	v1Agent, _ := json.Marshal(AgentDescriptor{
		ID: "edgex-node-001", Kind: "edgex-gateway", Status: AgentOnline,
		Transport: TransportList{"mqtt"}, HeartbeatIntervalSec: 30,
	})
	dc.HandleAgentOnline(TopicDiscoveryAgent, v1Agent, "v1-bridge")
	require.False(t, dc.HasNativeEANAgent("edgex-node-001"))

	// 北向 Runtime 发布原生 Capability → 应清除 V1 合成项
	nativePayload := []byte(`{
		"header":{"message_id":"m1","timestamp":1,"source":"edgex-node-001","message_type":"capability_descriptor","version":"2.0"},
		"body":{"capabilities":[
			{"id":"system.diagnostics","agent_id":"edgex-node-001","category":"system","timeout_sec":10,"permission":"read"},
			{"id":"bacnet_ip.write_point","agent_id":"edgex-node-001","category":"device","timeout_sec":15,"permission":"write"}
		]}
	}`)
	dc.HandleCapability(TopicDiscoveryCapability, nativePayload, "mqtt")

	_, stillV1 := dc.GetCapability(v1Cap.ID)
	assert.False(t, stillV1, "v1-bridge caps must be purged after native EAN arrival")
	assert.True(t, dc.HasNativeEANCaps("edgex-node-001"))

	native, v1 := dc.CountCapabilitiesBySource("edgex-node-001")
	assert.Equal(t, 2, native)
	assert.Equal(t, 0, v1)

	src, ok := dc.GetCapabilitySource("system.diagnostics")
	require.True(t, ok)
	assert.Equal(t, CapSourceNativeEAN, src)
}

func TestDiscovery_NativeAgentNotOverwrittenByV1Bridge(t *testing.T) {
	dc := NewDiscoveryCenter(DiscoveryConfig{}, testLogger(t))

	// EdgeX 北向 Agent Descriptor（kind=device, transport 为 string, metadata 含非 string）
	nativeAgent := []byte(`{
		"header":{"message_id":"a1","timestamp":1,"source":"edgex-node-001","message_type":"agent_descriptor","version":"2.0"},
		"body":{"agent":{
			"id":"edgex-node-001",
			"kind":"device",
			"version":"2.0.0",
			"status":"online",
			"transport":"mqtt",
			"heartbeat_interval_sec":30,
			"metadata":{"northbound":"edgeos_mqtt","compat":"v1_topics_retained","port":8082}
		}}
	}`)
	dc.HandleAgentOnline(TopicDiscoveryAgent, nativeAgent, "mqtt")
	require.True(t, dc.HasNativeEANAgent("edgex-node-001"))

	agent, ok := dc.GetAgent("edgex-node-001")
	require.True(t, ok)
	assert.Equal(t, "device", agent.Kind)
	assert.Equal(t, TransportList{"mqtt"}, agent.Transport)
	assert.Equal(t, "8082", agent.Metadata["port"])

	// V1 Bridge 试图覆盖为 edgex-gateway → 必须被拒绝
	v1Agent, _ := json.Marshal(AgentDescriptor{
		ID: "edgex-node-001", Kind: "edgex-gateway", Status: AgentOnline,
		Transport: TransportList{"mqtt"}, HeartbeatIntervalSec: 30,
	})
	dc.HandleAgentOnline(TopicDiscoveryAgent, v1Agent, "v1-bridge")

	agent, ok = dc.GetAgent("edgex-node-001")
	require.True(t, ok)
	assert.Equal(t, "device", agent.Kind, "native northbound agent must not be overwritten by v1-bridge")
}

func TestDiscovery_ResponseIndexesNativeCaps(t *testing.T) {
	dc := NewDiscoveryCenter(DiscoveryConfig{}, testLogger(t))

	// 预置 V1 污染
	v1Raw, _ := json.Marshal(CapabilityDescriptor{
		ID: "edgex-node-001/x/read-write", AgentID: "edgex-node-001",
		Category: CapabilityCategoryDevice, TimeoutSec: 30, Permission: "write",
	})
	dc.HandleCapability(TopicDiscoveryCapability, v1Raw, "v1-bridge")

	// EdgeX HandleDiscoveryQuery → $edgeos/discovery/response
	resp := []byte(`{
		"header":{
			"message_id":"r1","timestamp":2,"source":"edgex-node-001",
			"destination":"edgeos-planner","message_type":"discovery_response",
			"version":"2.0","correlation_id":"q1"
		},
		"body":{
			"agent":{"id":"edgex-node-001","kind":"device","version":"2.0.0","status":"online","transport":"mqtt","heartbeat_interval_sec":30},
			"capabilities":[
				{"id":"system.diagnostics","agent_id":"edgex-node-001","category":"system","timeout_sec":10,"permission":"read"},
				{"id":"ai.doc_parse","agent_id":"edgex-node-001","category":"ai","timeout_sec":60,"permission":"admin"}
			]
		}
	}`)
	dc.HandleDiscoveryResponse(TopicDiscoveryResponse, resp, "mqtt")

	require.True(t, dc.HasNativeEANAgent("edgex-node-001"))
	require.True(t, dc.HasNativeEANCaps("edgex-node-001"))
	_, v1Left := dc.GetCapability("edgex-node-001/x/read-write")
	assert.False(t, v1Left)

	native, v1 := dc.CountCapabilitiesBySource("edgex-node-001")
	assert.Equal(t, 2, native)
	assert.Equal(t, 0, v1)
}

func TestInvoke_NoV1FallbackForNativeCapability(t *testing.T) {
	logger := testLogger(t)
	dc := NewDiscoveryCenter(DiscoveryConfig{}, logger)
	dc.HandleAgentOnline(TopicDiscoveryAgent, []byte(`{
		"id":"edgex-node-001","kind":"edgex-gateway","status":"online","transport":["mqtt"],"heartbeat_interval_sec":30
	}`), "mqtt")
	dc.HandleCapability(TopicDiscoveryCapability, []byte(`{
		"header":{"message_type":"capability_descriptor","source":"edgex-node-001","version":"2.0","message_id":"m","timestamp":1},
		"body":{"capabilities":[{"id":"system.diagnostics","agent_id":"edgex-node-001","category":"system","timeout_sec":5,"permission":"read"}]}
	}`), "mqtt")

	src, ok := dc.GetCapabilitySource("system.diagnostics")
	require.True(t, ok)
	assert.Equal(t, CapSourceNativeEAN, src)
	// OS-P3-01: isV1SyntheticCapabilityID 已移除
}
