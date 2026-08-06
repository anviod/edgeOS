package ean

import (
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"go.uber.org/zap"
)

// ==================== 回调钩子 ====================

// OnAgentOnlineHook Agent 上线回调
type OnAgentOnlineHook func(agent *AgentDescriptor)

// OnAgentOfflineHook Agent 下线回调
type OnAgentOfflineHook func(agentID, reason string)

// OnCapabilityHook Capability 注册/更新回调
type OnCapabilityHook func(cap *CapabilityDescriptor)

// ==================== DiscoveryCenter ====================

// CapSource 标记 Capability 注册来源，用于 V1 Bridge 隔离
type CapSource string

const (
	CapSourceNativeEAN CapSource = "native-ean" // 原生 EAN（MQTT/NATS 发布）
	CapSourceV1Bridge  CapSource = "v1-bridge"  // V1 Bridge 合成
)

// DiscoveryCenter 发现中心，全局 Agent/Capability 索引
// 订阅 edgeCore **北向 EAN Runtime（TransportMQTT + mqttBus）** 发布的发现消息，
// 维护在线 Agent 列表与能力注册表。不对接 edgeCore MCP Runtime（TransportSDK + NoopBus）。
type DiscoveryCenter struct {
	// 索引存储
	agents       map[string]*AgentDescriptor       // agentID -> Agent
	capabilities map[string]*CapabilityDescriptor  // capabilityID -> Capability
	// agentID -> []capabilityID 辅助索引
	agentCapIndex map[string][]string
	// capID -> 来源标记（V1 Bridge 隔离：native-ean 优先于 v1-bridge）
	capSources map[string]CapSource
	// agentID -> 来源标记（native-ean 后拒绝 V1 Bridge 覆盖 Agent 描述符）
	agentSources map[string]CapSource

	// 回调钩子
	onAgentOnline  OnAgentOnlineHook
	onAgentOffline OnAgentOfflineHook
	onCapability   OnCapabilityHook

	logger *zap.Logger
	mu     sync.RWMutex
}

// DiscoveryConfig 发现中心配置
type DiscoveryConfig struct {
	// OnAgentOnline Agent 上线回调钩子（可选）
	OnAgentOnline OnAgentOnlineHook
	// OnAgentOffline Agent 下线回调钩子（可选）
	OnAgentOffline OnAgentOfflineHook
	// OnCapability Capability 注册回调钩子（可选）
	OnCapability OnCapabilityHook
}

// NewDiscoveryCenter 创建发现中心
func NewDiscoveryCenter(cfg DiscoveryConfig, logger *zap.Logger) *DiscoveryCenter {
	return &DiscoveryCenter{
		agents:        make(map[string]*AgentDescriptor),
		capabilities:  make(map[string]*CapabilityDescriptor),
		agentCapIndex: make(map[string][]string),
		capSources:    make(map[string]CapSource),
		agentSources:  make(map[string]CapSource),
		onAgentOnline: cfg.OnAgentOnline,
		onAgentOffline: cfg.OnAgentOffline,
		onCapability:  cfg.OnCapability,
		logger:        logger.Named("discovery"),
	}
}

// ==================== MessageHandler 实现 ====================

// HandleAgentOnline 处理 Agent 上线消息
// 用作 Subscribe(TopicDiscoveryAgent, discovery.HandleAgentOnline) 的回调
// 兼容协议信封 body.agent / body 即 descriptor / 裸 descriptor
// 来源隔离：native-ean（mqtt/nats 北向）优先；已有原生 Agent 时拒绝 v1-bridge 覆盖
func (dc *DiscoveryCenter) HandleAgentOnline(topic string, payload []byte, transport string) {
	agent, err := decodeAgentDescriptor(payload)
	if err != nil {
		dc.logger.Warn("failed to unmarshal agent descriptor",
			zap.String("topic", topic), zap.Error(err))
		return
	}
	if agent.ID == "" {
		dc.logger.Warn("agent descriptor missing id", zap.String("topic", topic))
		return
	}
	if agent.Metadata == nil {
		agent.Metadata = FlexibleStringMap{}
	}

	source := dc.resolveCapSource(transport)

	dc.mu.Lock()
	existingSource, exists := dc.agentSources[agent.ID]
	if exists && existingSource == CapSourceNativeEAN && source == CapSourceV1Bridge {
		dc.mu.Unlock()
		dc.logger.Debug("agent skipped: native EAN already exists",
			zap.String("agent_id", agent.ID))
		return
	}

	agent.Status = AgentOnline
	stored := agent
	dc.agents[agent.ID] = &stored
	dc.agentSources[agent.ID] = source
	dc.mu.Unlock()

	dc.logger.Info("agent online",
		zap.String("agent_id", agent.ID),
		zap.String("kind", agent.Kind),
		zap.Int("heartbeat_interval", agent.HeartbeatIntervalSec),
		zap.String("transport", transport),
		zap.String("source", string(source)))

	if dc.onAgentOnline != nil {
		dc.onAgentOnline(&stored)
	}
}

// HandleAgentOffline 处理 Agent 下线消息
// 用作 Subscribe(TopicDiscoveryAgentOffline, discovery.HandleAgentOffline) 的回调
// 设计（v2.25 / EdgeOS 端）：北向关闭 EAN 能力层时 edgeCore 发送 reason=graceful_shutdown 下线——
// 该 Agent 不再是 EAN 参与者，从 Agent 管理页与能力索引中**彻底移除**（与节点注册表删除一致，
// 避免残留 offline 幽灵 Agent）。心跳超时/异常掉线仍标记 offline（保留历史）。
// | On graceful_shutdown (northbound EAN disabled) the agent is removed entirely;
// | on heartbeat-timeout / unexpected drop it is marked offline to preserve history.
func (dc *DiscoveryCenter) HandleAgentOffline(topic string, payload []byte, transport string) {
	desc, err := decodeAgentOffline(payload)
	if err != nil {
		dc.logger.Warn("failed to unmarshal agent offline descriptor",
			zap.String("topic", topic), zap.Error(err))
		return
	}
	if desc.AgentID == "" {
		dc.logger.Warn("agent offline missing agent_id", zap.String("topic", topic))
		return
	}

	// 北向关闭 EAN（graceful_shutdown）→ 彻底移除 Agent（Agent 页不再显示该节点）。
	// | Northbound EAN disabled (graceful_shutdown) → remove agent entirely (not shown on Agent page).
	if desc.Reason == "graceful_shutdown" {
		dc.logger.Info("agent removed (northbound EAN disabled, graceful_shutdown)",
			zap.String("agent_id", desc.AgentID),
			zap.String("transport", transport))
		dc.DeleteAgent(desc.AgentID)
		return
	}

	dc.mu.Lock()
	if agent, ok := dc.agents[desc.AgentID]; ok {
		agent.Status = AgentOffline
	}
	for _, cid := range dc.agentCapIndex[desc.AgentID] {
		delete(dc.capabilities, cid)
		delete(dc.capSources, cid)
	}
	delete(dc.agentCapIndex, desc.AgentID)
	delete(dc.agentSources, desc.AgentID)
	dc.mu.Unlock()

	dc.logger.Info("agent offline",
		zap.String("agent_id", desc.AgentID),
		zap.String("reason", desc.Reason),
		zap.String("transport", transport))

	if dc.onAgentOffline != nil {
		dc.onAgentOffline(desc.AgentID, desc.Reason)
	}
}

// HandleCapability 处理 Capability 注册/更新消息
// 用作 Subscribe(TopicDiscoveryCapability, discovery.HandleCapability) 的回调
// 兼容协议信封 body.capabilities[] / 单条 body / 裸 descriptor
// source: 传输来源标识（"mqtt"/"nats" → CapSourceNativeEAN, "v1-bridge" → CapSourceV1Bridge）
func (dc *DiscoveryCenter) HandleCapability(topic string, payload []byte, transport string) {
	source := dc.resolveCapSource(transport)
	caps, err := decodeCapabilities(payload)
	if err != nil {
		dc.logger.Warn("failed to unmarshal capability descriptor",
			zap.String("topic", topic), zap.Error(err))
		return
	}
	for i := range caps {
		cap := caps[i]
		if cap.ID == "" {
			continue
		}
		dc.upsertCapability(&cap, source)
	}
}

// HandleDiscoveryResponse 处理 edgeCore 对 $edgeos/discovery/query 的响应
// 响应格式: {"header":..., "body":{"agent":{...}, "capabilities":[...]}}
// edgeCore 回复的 Capability 均为原生 EAN，优先级最高
func (dc *DiscoveryCenter) HandleDiscoveryResponse(topic string, payload []byte, transport string) {
	body := unwrapBody(payload)

	var resp struct {
		Agent        *AgentDescriptor      `json:"agent"`
		Capabilities []CapabilityDescriptor `json:"capabilities"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		dc.logger.Warn("failed to unmarshal discovery response",
			zap.String("topic", topic), zap.Error(err))
		return
	}

	// 处理 Agent（discovery/response 来自北向 Runtime，视为原生 EAN）
	if resp.Agent != nil && resp.Agent.ID != "" {
		if resp.Agent.Metadata == nil {
			resp.Agent.Metadata = FlexibleStringMap{}
		}
		dc.mu.Lock()
		resp.Agent.Status = AgentOnline
		stored := *resp.Agent
		dc.agents[resp.Agent.ID] = &stored
		dc.agentSources[resp.Agent.ID] = CapSourceNativeEAN
		dc.mu.Unlock()
		dc.logger.Info("agent updated via discovery response",
			zap.String("agent_id", resp.Agent.ID),
			zap.String("kind", resp.Agent.Kind),
			zap.String("source", string(CapSourceNativeEAN)))
		if dc.onAgentOnline != nil {
			dc.onAgentOnline(&stored)
		}
	}

	// 处理 Capabilities（原生 EAN，最高优先级）
	addedCount := 0
	for i := range resp.Capabilities {
		cap := resp.Capabilities[i]
		if cap.ID == "" {
			continue
		}
		dc.upsertCapability(&cap, CapSourceNativeEAN)
		addedCount++
	}
	if addedCount > 0 {
		dc.logger.Info("discovery response capabilities indexed",
			zap.Int("count", addedCount),
			zap.String("transport", transport))
	}
}

// resolveCapSource 将传输标识映射到 CapSource
// "v1-bridge" → V1Bridge, 其余（mqtt/nats）→ NativeEAN
func (dc *DiscoveryCenter) resolveCapSource(transport string) CapSource {
	if transport == "v1-bridge" {
		return CapSourceV1Bridge
	}
	return CapSourceNativeEAN
}

// ==================== 查询 API ====================

// ListAgents 列出所有 Agent（仅返回在线或包含指定状态的 Agent）
// statusFilter 为空时返回全部 Agent
func (dc *DiscoveryCenter) ListAgents(statusFilter ...AgentStatus) []*AgentDescriptor {
	dc.mu.RLock()
	defer dc.mu.RUnlock()

	if len(statusFilter) == 0 {
		result := make([]*AgentDescriptor, 0, len(dc.agents))
		for _, a := range dc.agents {
			result = append(result, a)
		}
		return result
	}

	result := make([]*AgentDescriptor, 0)
	for _, a := range dc.agents {
		for _, sf := range statusFilter {
			if a.Status == sf {
				result = append(result, a)
				break
			}
		}
	}
	return result
}

// GetAgent 获取指定 Agent 描述符副本
func (dc *DiscoveryCenter) GetAgent(agentID string) (*AgentDescriptor, bool) {
	dc.mu.RLock()
	defer dc.mu.RUnlock()
	a, ok := dc.agents[agentID]
	if !ok {
		return nil, false
	}
	cp := *a
	return &cp, true
}

// ListCapabilities 列出所有 Capability
// 可选按 agentID 或 category 过滤
func (dc *DiscoveryCenter) ListCapabilities(agentID string, category ...CapabilityCategory) []*CapabilityDescriptor {
	dc.mu.RLock()
	defer dc.mu.RUnlock()

	result := make([]*CapabilityDescriptor, 0)
	for _, cap := range dc.capabilities {
		// agentID 过滤
		if agentID != "" && cap.AgentID != agentID {
			continue
		}
		// category 过滤
		if len(category) > 0 {
			matched := false
			for _, cat := range category {
				if cap.Category == cat {
					matched = true
					break
				}
			}
			if !matched {
				continue
			}
		}
		result = append(result, cap)
	}
	return result
}

// GetCapability 获取指定 Capability 描述符
func (dc *DiscoveryCenter) GetCapability(capID string) (*CapabilityDescriptor, bool) {
	dc.mu.RLock()
	defer dc.mu.RUnlock()
	c, ok := dc.capabilities[capID]
	return c, ok
}

// FindCapabilityTimeout 按类别、描述关键字模糊查找满足超时要求的 Capability
// minTimeout: 最低超时要求（秒），0 表示不限
// keyword: 在 description 中模糊匹配的关键字，空串表示不限
// 返回匹配的第一个 Capability（在线 Agent 所属优先）
func (dc *DiscoveryCenter) FindCapabilityTimeout(category CapabilityCategory, minTimeout int, keyword string) (*CapabilityDescriptor, bool) {
	dc.mu.RLock()
	defer dc.mu.RUnlock()

	// 优先返回在线 Agent 的 capability
	for _, cap := range dc.capabilities {
		if !categoryMatch(cap.Category, category) {
			continue
		}
		if minTimeout > 0 && cap.TimeoutSec < minTimeout {
			continue
		}
		if keyword != "" && !strings.Contains(strings.ToLower(cap.Description), strings.ToLower(keyword)) {
			continue
		}
		// 检查所属 Agent 是否在线
		if agent, ok := dc.agents[cap.AgentID]; ok && agent.Status == AgentOnline {
			return cap, true
		}
	}

	// 退而求其次：不限 Agent 状态
	for _, cap := range dc.capabilities {
		if !categoryMatch(cap.Category, category) {
			continue
		}
		if minTimeout > 0 && cap.TimeoutSec < minTimeout {
			continue
		}
		if keyword != "" && !strings.Contains(strings.ToLower(cap.Description), strings.ToLower(keyword)) {
			continue
		}
		return cap, true
	}

	return nil, false
}

// categoryMatch 兼容协议 category=device 与内部别名 driver
func categoryMatch(actual, want CapabilityCategory) bool {
	if actual == want {
		return true
	}
	if (want == CapabilityCategoryDriver || want == CapabilityCategoryDevice) &&
		(actual == CapabilityCategoryDriver || actual == CapabilityCategoryDevice) {
		return true
	}
	return false
}

// GetCapabilitiesByAgent 获取指定 Agent 下的所有 Capability
func (dc *DiscoveryCenter) GetCapabilitiesByAgent(agentID string) []*CapabilityDescriptor {
	dc.mu.RLock()
	defer dc.mu.RUnlock()

	capIDs := dc.agentCapIndex[agentID]
	result := make([]*CapabilityDescriptor, 0, len(capIDs))
	for _, cid := range capIDs {
		if cap, ok := dc.capabilities[cid]; ok {
			result = append(result, cap)
		}
	}
	return result
}

// HasNativeEANCaps 检查指定 Agent 是否已有原生 EAN Capability（非 V1 Bridge 合成）
// 用于 V1 Bridge 隔离：若已有原生 EAN Cap，V1 Bridge 不再合成设备级 Capability
func (dc *DiscoveryCenter) HasNativeEANCaps(agentID string) bool {
	dc.mu.RLock()
	defer dc.mu.RUnlock()

	capIDs := dc.agentCapIndex[agentID]
	for _, cid := range capIDs {
		if src, ok := dc.capSources[cid]; ok && src == CapSourceNativeEAN {
			return true
		}
	}
	return false
}

// HasNativeEANAgent 检查指定 Agent 是否已由北向原生 EAN 注册
func (dc *DiscoveryCenter) HasNativeEANAgent(agentID string) bool {
	dc.mu.RLock()
	defer dc.mu.RUnlock()
	return dc.agentSources[agentID] == CapSourceNativeEAN
}

// GetCapabilitySource 返回 Capability 注册来源
func (dc *DiscoveryCenter) GetCapabilitySource(capID string) (CapSource, bool) {
	dc.mu.RLock()
	defer dc.mu.RUnlock()
	src, ok := dc.capSources[capID]
	return src, ok
}

// CountCapabilitiesBySource 统计指定 Agent 下各来源 Capability 数量
func (dc *DiscoveryCenter) CountCapabilitiesBySource(agentID string) (native, v1Bridge int) {
	dc.mu.RLock()
	defer dc.mu.RUnlock()
	for _, cid := range dc.agentCapIndex[agentID] {
		switch dc.capSources[cid] {
		case CapSourceNativeEAN:
			native++
		case CapSourceV1Bridge:
			v1Bridge++
		}
	}
	return native, v1Bridge
}

// ==================== 活跃度检查 ====================

// OnlineAgentCount 在线 Agent 数量
func (dc *DiscoveryCenter) OnlineAgentCount() int {
	dc.mu.RLock()
	defer dc.mu.RUnlock()
	count := 0
	for _, a := range dc.agents {
		if a.Status == AgentOnline {
			count++
		}
	}
	return count
}

// StaleAgents 获取在线超时的 Agent 列表
// maxIdle: 最大允许空闲时间（超过此时间未收到心跳/更新则视为过期）
func (dc *DiscoveryCenter) StaleAgents(maxIdle time.Duration) []*AgentDescriptor {
	dc.mu.RLock()
	defer dc.mu.RUnlock()

	result := make([]*AgentDescriptor, 0)
	for _, a := range dc.agents {
		if a.Status != AgentOnline {
			continue
		}
		// 通过 Metadata 中的 last_seen 时间戳判断（由心跳模块写入）
		if lastSeenStr, ok := a.Metadata["last_seen"]; ok {
			if lastSeen, err := time.Parse(time.RFC3339Nano, lastSeenStr); err == nil {
				if time.Since(lastSeen) > maxIdle {
					result = append(result, a)
				}
			}
		}
	}
	return result
}

// RemoveAgent 清理指定 Agent（由心跳超时模块调用）
// 仅将 Agent 标记为 offline（保留描述符供 UI 展示），并记录 offline_since 时间戳，
// 供过期清理逻辑在 Agent 离线超过保留期后彻底删除（DeleteAgent）。
func (dc *DiscoveryCenter) RemoveAgent(agentID string) {
	dc.mu.Lock()
	defer dc.mu.Unlock()

	if agent, ok := dc.agents[agentID]; ok {
		agent.Status = AgentOffline
		if agent.Metadata == nil {
			agent.Metadata = FlexibleStringMap{}
		}
		if _, exists := agent.Metadata["offline_since"]; !exists {
			agent.Metadata["offline_since"] = time.Now().Format(time.RFC3339Nano)
		}
		dc.logger.Info("agent removed by heartbeat timeout",
			zap.String("agent_id", agentID))
	}
	// 清理该 Agent 的 Capability 索引与来源标记
	for _, cid := range dc.agentCapIndex[agentID] {
		delete(dc.capabilities, cid)
		delete(dc.capSources, cid)
	}
	delete(dc.agentCapIndex, agentID)
	delete(dc.agentSources, agentID)

	// 释放锁后触发下线回调（Registry 镜像依赖此清理 /api/nodes 节点；Phase 4 OS-P4）
	// Invoke offline hook after releasing lock so downstream (registry mirror) can clean up /api/nodes.
	hook := dc.onAgentOffline
	if hook != nil {
		go hook(agentID, "heartbeat_timeout")
	}
}

// DeleteAgent 彻底移除指定 Agent（含能力/索引/来源标记），并从 /api/nodes 清除对应节点。
// 与 RemoveAgent 仅标记 offline 不同，DeleteAgent 用于用户主动删除残留/离线 Agent，
// 保证 Agent 管理与节点管理的一致性（删除节点/Agent 后两端都不再残留）。
// | Permanently delete an agent (agent descriptor + capabilities + indexes), unlike
// | RemoveAgent which only marks it offline. Used for explicit cleanup from Agent 管理.
func (dc *DiscoveryCenter) DeleteAgent(agentID string) {
	dc.mu.Lock()
	_, existed := dc.agents[agentID]
	delete(dc.agents, agentID)
	for _, cid := range dc.agentCapIndex[agentID] {
		delete(dc.capabilities, cid)
		delete(dc.capSources, cid)
	}
	delete(dc.agentCapIndex, agentID)
	delete(dc.agentSources, agentID)
	dc.mu.Unlock()

	if !existed {
		dc.logger.Debug("agent delete: not found",
			zap.String("agent_id", agentID))
		return
	}
	dc.logger.Info("agent deleted (explicit)",
		zap.String("agent_id", agentID))
	// 同步清理 /api/nodes 对应节点（registry mirror 依赖 offline hook）
	hook := dc.onAgentOffline
	if hook != nil {
		go hook(agentID, "agent_deleted")
	}
}

// TouchLastSeen 由心跳监控器安全写入 last_seen（持有写锁，避免与索引并发竞态）
func (dc *DiscoveryCenter) TouchLastSeen(agentID string, when time.Time, seq int) {
	dc.mu.Lock()
	defer dc.mu.Unlock()

	agent, ok := dc.agents[agentID]
	if !ok {
		return
	}
	if agent.Metadata == nil {
		agent.Metadata = FlexibleStringMap{}
	}
	agent.Metadata["last_seen"] = when.Format(time.RFC3339Nano)
	agent.Metadata["last_heartbeat_seq"] = fmt.Sprintf("%d", seq)
}

func (dc *DiscoveryCenter) upsertCapability(cap *CapabilityDescriptor, source CapSource) {
	dc.mu.Lock()

	// V1 Bridge 隔离：若已存在同名原生 EAN Capability，拒绝 V1 Bridge 覆盖
	existingSource, exists := dc.capSources[cap.ID]
	if exists && existingSource == CapSourceNativeEAN && source == CapSourceV1Bridge {
		dc.mu.Unlock()
		dc.logger.Debug("capability skipped: native EAN already exists",
			zap.String("cap_id", cap.ID),
			zap.String("agent_id", cap.AgentID))
		return
	}

	// 原生 EAN 覆盖 V1 Bridge 合成（升级）
	if exists && existingSource == CapSourceV1Bridge && source == CapSourceNativeEAN {
		dc.logger.Info("capability upgraded: v1-bridge → native-ean",
			zap.String("cap_id", cap.ID),
			zap.String("agent_id", cap.AgentID))
	}

	// 原生 EAN 到达时清除该 Agent 下残留的 V1 合成 Capability（ID 不同，同名覆盖无法清理）
	purged := 0
	if source == CapSourceNativeEAN && cap.AgentID != "" {
		purged = dc.purgeV1BridgeCapsLocked(cap.AgentID)
	}

	stored := *cap
	dc.capabilities[cap.ID] = &stored
	dc.capSources[cap.ID] = source

	found := false
	for _, cid := range dc.agentCapIndex[cap.AgentID] {
		if cid == cap.ID {
			found = true
			break
		}
	}
	if !found {
		dc.agentCapIndex[cap.AgentID] = append(dc.agentCapIndex[cap.AgentID], cap.ID)
	}

	dc.mu.Unlock()

	dc.logger.Info("capability registered/updated",
		zap.String("cap_id", cap.ID),
		zap.String("agent_id", cap.AgentID),
		zap.String("category", string(cap.Category)),
		zap.String("source", string(source)),
		zap.Int("v1_purged", purged))

	if dc.onCapability != nil {
		dc.onCapability(&stored)
	}
}

// purgeV1BridgeCapsLocked 清除指定 Agent 下全部 v1-bridge Capability（调用方须持有写锁）
func (dc *DiscoveryCenter) purgeV1BridgeCapsLocked(agentID string) int {
	kept := make([]string, 0, len(dc.agentCapIndex[agentID]))
	purged := 0
	for _, cid := range dc.agentCapIndex[agentID] {
		if dc.capSources[cid] == CapSourceV1Bridge {
			delete(dc.capabilities, cid)
			delete(dc.capSources, cid)
			purged++
			continue
		}
		kept = append(kept, cid)
	}
	if purged > 0 {
		dc.agentCapIndex[agentID] = kept
		dc.logger.Info("purged v1-bridge capabilities after native EAN arrival",
			zap.String("agent_id", agentID),
			zap.Int("purged", purged))
	}
	return purged
}

// decodeAgentDescriptor 解析协议/裸格式 Agent Descriptor
func decodeAgentDescriptor(payload []byte) (AgentDescriptor, error) {
	body := unwrapBody(payload)

	var wrapped struct {
		Agent *AgentDescriptor `json:"agent"`
	}
	if err := json.Unmarshal(body, &wrapped); err == nil && wrapped.Agent != nil && wrapped.Agent.ID != "" {
		return *wrapped.Agent, nil
	}

	var agent AgentDescriptor
	if err := json.Unmarshal(body, &agent); err != nil {
		return AgentDescriptor{}, err
	}
	return agent, nil
}

func decodeAgentOffline(payload []byte) (AgentOfflineDescriptor, error) {
	var desc AgentOfflineDescriptor
	if err := parseJSON(payload, &desc); err != nil {
		return AgentOfflineDescriptor{}, err
	}
	if desc.AgentID == "" {
		var alt struct {
			ID string `json:"id"`
		}
		_ = parseJSON(payload, &alt)
		desc.AgentID = alt.ID
	}
	return desc, nil
}

func decodeCapabilities(payload []byte) ([]CapabilityDescriptor, error) {
	body := unwrapBody(payload)

	var wrapped struct {
		Capabilities []CapabilityDescriptor `json:"capabilities"`
	}
	if err := json.Unmarshal(body, &wrapped); err == nil && len(wrapped.Capabilities) > 0 {
		return wrapped.Capabilities, nil
	}

	var single CapabilityDescriptor
	if err := json.Unmarshal(body, &single); err != nil {
		return nil, err
	}
	if single.ID == "" {
		return nil, fmt.Errorf("capability descriptor missing id")
	}
	return []CapabilityDescriptor{single}, nil
}

// ==================== 订阅注册辅助 ====================

// RegisterSubscriptions 在 bus 上注册发现中心所需的所有订阅
// bus: 实现 Subscribe 的传输层（如 DualTransport）
func (dc *DiscoveryCenter) RegisterSubscriptions(bus interface{ Subscribe(string, MessageHandler) error }) error {
	// Agent 上线
	if err := bus.Subscribe(TopicDiscoveryAgent, dc.HandleAgentOnline); err != nil {
		return fmt.Errorf("subscribe agent online failed: %w", err)
	}
	// Agent 下线
	if err := bus.Subscribe(TopicDiscoveryAgentOffline, dc.HandleAgentOffline); err != nil {
		return fmt.Errorf("subscribe agent offline failed: %w", err)
	}
	// Capability 注册
	if err := bus.Subscribe(TopicDiscoveryCapability, dc.HandleCapability); err != nil {
		return fmt.Errorf("subscribe capability failed: %w", err)
	}
	// Discovery Response（edgeCore 响应主动查询）
	if err := bus.Subscribe(TopicDiscoveryResponse, dc.HandleDiscoveryResponse); err != nil {
		return fmt.Errorf("subscribe discovery response failed: %w", err)
	}

	dc.logger.Info("discovery center subscriptions registered")
	return nil
}
