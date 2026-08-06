package ean

import (
	"go.uber.org/zap"

	"github.com/anviod/edgeOS/internal/services"
)

// AttachRegistryMirror 将 EAN Discovery Agent 上线/下线镜像到 V1 节点注册表。
// 背景：EAN 已替代 V1 节点注册后，edgeCore 可能不再稳定发送 edgeCore/nodes/register，
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
		// Phase 4 (OS-P4): 仅镜像「北向原生 EAN Agent」到 V1 节点注册表，
		// 避免集成测试/临时 Agent（v1-bridge 或仅发布 agent discovery 的 transient agent）污染 /api/nodes。
		// | Mirror only northbound native-EAN agents to the V1 registry, excluding transient/test agents.
		if !b.Discovery.HasNativeEANAgent(agent.ID) && !b.Discovery.HasNativeEANCaps(agent.ID) {
			b.logger.Debug("registry mirror: skip non-native agent (transient/v1-bridge)",
				zap.String("agent_id", agent.ID))
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
		// Phase 4 (OS-P4): Agent 下线 → 从 V1 节点注册表删除（而非仅标记 offline），
		// 避免 transient/集成测试 Agent 残留为「offline 多余节点」。真实 edgeCore 节点
		// 重新上线时由 onAgentOnline 重新镜像。
		// | Agent offline → remove node from V1 registry (not just mark offline),
		// | so transient/test agents don't linger as offline pollution.
		if err := registry.DeleteNode(agentID); err != nil {
			// 节点可能尚未镜像过，仅 debug
			b.logger.Debug("registry mirror: delete node skipped",
				zap.String("agent_id", agentID), zap.String("reason", reason), zap.Error(err))
		}
	}

	// 启动时对已在索引中的 Agent 做一次回填（例如 Attach 晚于 Discovery 响应）
	// 仅回填北向原生 EAN Agent，避免 transient/v1-bridge 污染 /api/nodes。
	for _, agent := range b.Discovery.ListAgents() {
		if agent == nil || agent.ID == "" || agent.Status != AgentOnline {
			continue
		}
		if !b.Discovery.HasNativeEANAgent(agent.ID) && !b.Discovery.HasNativeEANCaps(agent.ID) {
			continue
		}
		_ = registry.EnsureNodeOnline(agent.ID, agent.ID, "ean")
	}

	b.logger.Info("EAN→Registry mirror attached")
}
