package ean

import (
	"go.uber.org/zap"

	"github.com/anviod/edgeOS/internal/services"
)

// AttachRegistryMirror 将 EAN Discovery Agent 上线/下线镜像到 V1 节点注册表。
// 背景：EAN 已替代 V1 节点注册后，EdgeX 可能不再稳定发送 edgex/nodes/register，
// 导致 /api/nodes 与 Dashboard total_nodes=0，但 /api/ean/agents 仍有在线 Agent，
// 且 V1 设备仍按 node_id 落库——形成「设备有、节点无」的展示不一致。
func (b *Bus) AttachRegistryMirror(registry *services.RegistryService) {
	if b == nil || b.Discovery == nil || registry == nil {
		return
	}

	prevOnline := b.Discovery.onAgentOnline
	prevOffline := b.Discovery.onAgentOffline

	b.Discovery.onAgentOnline = func(agent *AgentDescriptor) {
		if prevOnline != nil {
			prevOnline(agent)
		}
		if agent == nil || agent.ID == "" {
			return
		}
		name := agent.ID
		if agent.Metadata != nil {
			if v, ok := agent.Metadata["node_name"]; ok && v != "" {
				name = v
			}
		}
		protocol := "ean"
		if len(agent.Transport) > 0 {
			protocol = "ean/" + agent.Transport[0]
		}
		if err := registry.EnsureNodeOnline(agent.ID, name, protocol); err != nil {
			b.logger.Warn("registry mirror: ensure node online failed",
				zap.String("agent_id", agent.ID), zap.Error(err))
			return
		}
		b.logger.Debug("registry mirror: agent mirrored as online node",
			zap.String("agent_id", agent.ID))
	}

	b.Discovery.onAgentOffline = func(agentID, reason string) {
		if prevOffline != nil {
			prevOffline(agentID, reason)
		}
		if agentID == "" {
			return
		}
		if err := registry.UpdateNodeStatus(agentID, "offline"); err != nil {
			// 节点可能尚未镜像过，仅 debug
			b.logger.Debug("registry mirror: mark offline skipped",
				zap.String("agent_id", agentID), zap.String("reason", reason), zap.Error(err))
		}
	}

	// 启动时对已在索引中的 Agent 做一次回填（例如 Attach 晚于 Discovery 响应）
	for _, agent := range b.Discovery.ListAgents() {
		if agent == nil || agent.ID == "" || agent.Status != AgentOnline {
			continue
		}
		_ = registry.EnsureNodeOnline(agent.ID, agent.ID, "ean")
	}

	b.logger.Info("EAN→Registry mirror attached")
}
