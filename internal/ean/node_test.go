package ean

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTopicHelpers(t *testing.T) {
	t.Run("InvokeTopic", func(t *testing.T) {
		got := InvokeTopic("agent-001")
		assert.Equal(t, "$edgeos/invoke/agent-001", got)
	})

	t.Run("ReplyTopic", func(t *testing.T) {
		got := ReplyTopic("source-abc")
		assert.Equal(t, "$edgeos/reply/source-abc", got)
	})

	t.Run("HeartbeatTopic", func(t *testing.T) {
		got := HeartbeatTopic("agent-002")
		assert.Equal(t, "$edgeos/heartbeat/agent-002", got)
	})

	t.Run("EventTopic", func(t *testing.T) {
		got := EventTopic("agent-003")
		assert.Equal(t, "$edgeos/event/agent-003", got)
	})

	t.Run("InvokeStatusTopic", func(t *testing.T) {
		got := InvokeStatusTopic("agent-004")
		assert.Equal(t, "$edgeos/invoke/agent-004/status", got)
	})
}

func TestAgentStatus(t *testing.T) {
	assert.Equal(t, AgentStatus("online"), AgentOnline)
	assert.Equal(t, AgentStatus("offline"), AgentOffline)
	assert.NotEqual(t, AgentOnline, AgentOffline)
}

func TestCapabilityCategory(t *testing.T) {
	assert.Equal(t, CapabilityCategory("driver"), CapabilityCategoryDriver)
	assert.Equal(t, CapabilityCategory("system"), CapabilityCategorySystem)
	assert.Equal(t, CapabilityCategory("ai"), CapabilityCategoryAI)
}

func TestTopicConstants(t *testing.T) {
	assert.Equal(t, "$edgeos/discovery/agent", TopicDiscoveryAgent)
	assert.Equal(t, "$edgeos/discovery/agent/offline", TopicDiscoveryAgentOffline)
	assert.Equal(t, "$edgeos/discovery/capability", TopicDiscoveryCapability)
	assert.Equal(t, "$edgeos/discovery/query", TopicDiscoveryQuery)
	assert.Equal(t, "$edgeos/discovery/response", TopicDiscoveryResponse)
	assert.Equal(t, "$edgeos/invoke/", TopicInvokePrefix)
	assert.Equal(t, "$edgeos/reply/", TopicReplyPrefix)
	assert.Equal(t, "$edgeos/event/", TopicEventPrefix)
	assert.Equal(t, "$edgeos/event/broadcast", TopicEventBroadcast)
	assert.Equal(t, "$edgeos/heartbeat/", TopicHeartbeatPrefix)
}

func TestPermissionConstants(t *testing.T) {
	assert.Equal(t, "read", PermissionRead)
	assert.Equal(t, "write", PermissionWrite)
	assert.Equal(t, "admin", PermissionAdmin)
	assert.Equal(t, "ai", PermissionAI)
}
