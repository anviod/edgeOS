package ean

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"go.uber.org/zap"
)

// ==================== 回调类型 ====================

// OnTimeoutCallback Agent 心跳超时回调
type OnTimeoutCallback func(agentID string, lastSeen time.Time, missedCount int)

// ==================== 心跳状态 ====================

// AgentHeartbeat 单个 Agent 的心跳状态
type AgentHeartbeat struct {
	AgentID      string    // Agent 标识
	LastSeen     time.Time // 最后一次收到心跳的时间
	Sequence     int       // 最后收到的序列号
	MissedCount  int       // 连续未收到心跳的计数
	IntervalSec  int       // 该 Agent 的心跳间隔（秒）
	OfflineSince time.Time // 标记为离线的时间（用于离线保留期自动清除）
}

// ==================== HeartbeatMonitor ====================

// HeartbeatMonitor 心跳监控器
// 接收 edgeCore Agent 心跳消息，定期检查超时并触发回调
// 超时判定: 超过 N * heartbeat_interval_sec 未收到心跳
type HeartbeatMonitor struct {
	// Agent 心跳状态
	agents map[string]*AgentHeartbeat // agentID -> AgentHeartbeat
	mu     sync.Mutex

	// 检查配置
	checkInterval   time.Duration // 检查循环间隔
	timeoutMultiplier int          // 超时 = timeoutMultiplier * heartbeat_interval
	// MaxOfflineRetention 离线保留期：Agent 心跳超时标记 offline 后，
	// 若超过该时长仍未重新上线（如 edgeCore 关闭 EAN），则彻底删除（DeleteAgent）。
	maxOfflineRetention time.Duration

	// 回调
	onTimeout OnTimeoutCallback

	// 生命周期
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup

	// 可选: 与发现中心联动
	discovery *DiscoveryCenter

	logger *zap.Logger
}

// HeartbeatMonitorConfig 心跳监控器配置
type HeartbeatMonitorConfig struct {
	// CheckInterval 检查循环间隔，默认 5s
	CheckInterval time.Duration
	// TimeoutMultiplier 超时倍数，默认 3（即 3 个心跳周期未收到则超时）
	// EAN 规范建议 2~3 个心跳周期
	TimeoutMultiplier int
	// MaxOfflineRetention 离线保留期，默认 10 分钟。
	// Agent 心跳超时标记 offline 后，超过该时长仍未重新上线则彻底删除（DeleteAgent）。
	// 0 或负值表示不自动清除（保留 offline 展示，由用户手动删除）。
	MaxOfflineRetention time.Duration
	// OnTimeout 超时回调（必须设置，用于通知上层清理 Agent）
	OnTimeout OnTimeoutCallback
	// Discovery 可选的发现中心引用，超时时自动标记 Agent 离线
	Discovery *DiscoveryCenter
}

// NewHeartbeatMonitor 创建心跳监控器
func NewHeartbeatMonitor(cfg HeartbeatMonitorConfig, logger *zap.Logger) *HeartbeatMonitor {
	if cfg.CheckInterval <= 0 {
		cfg.CheckInterval = 5 * time.Second
	}
	if cfg.TimeoutMultiplier <= 0 {
		cfg.TimeoutMultiplier = 3 // 默认 3 个心跳周期
	}
	// MaxOfflineRetention <= 0 表示禁用自动清除（保留 offline 展示，由用户手动删除）。
	// 默认 10 分钟离线保留期由上层配置（DefaultEANConfig.MaxOfflineRetentionSec=600）注入。
	ctx, cancel := context.WithCancel(context.Background())

	hm := &HeartbeatMonitor{
		agents:              make(map[string]*AgentHeartbeat),
		checkInterval:       cfg.CheckInterval,
		timeoutMultiplier:   cfg.TimeoutMultiplier,
		maxOfflineRetention: cfg.MaxOfflineRetention,
		onTimeout:           cfg.OnTimeout,
		discovery:           cfg.Discovery,
		ctx:                 ctx,
		cancel:              cancel,
		logger:              logger.Named("heartbeat"),
	}

	hm.logger.Info("heartbeat monitor created",
		zap.Duration("check_interval", cfg.CheckInterval),
		zap.Int("timeout_multiplier", cfg.TimeoutMultiplier))

	return hm
}

// ==================== MessageHandler 实现 ====================

// HandleHeartbeat 处理 Agent 心跳消息
// 用作 Subscribe(TopicHeartbeatPrefix+"#", monitor.HandleHeartbeat) 的回调
// 更新 Agent 的 lastSeen 时间，重置 missedCount；兼容协议信封
func (hm *HeartbeatMonitor) HandleHeartbeat(topic string, payload []byte, transport string) {
	var hb HeartbeatPayload
	if err := parseJSON(payload, &hb); err != nil {
		hm.logger.Warn("failed to unmarshal heartbeat payload",
			zap.String("topic", topic), zap.Error(err))
		return
	}
	if hb.AgentID == "" {
		// 兼容从 topic 提取: $edgeos/heartbeat/{agent_id}
		if strings.HasPrefix(topic, TopicHeartbeatPrefix) {
			hb.AgentID = strings.TrimPrefix(topic, TopicHeartbeatPrefix)
		}
	}
	if hb.AgentID == "" {
		hm.logger.Warn("heartbeat missing agent_id", zap.String("topic", topic))
		return
	}

	hm.mu.Lock()
	defer hm.mu.Unlock()

	now := time.Now()
	state, exists := hm.agents[hb.AgentID]

	if !exists {
		interval := 10
		if hm.discovery != nil {
			if agent, ok := hm.discovery.GetAgent(hb.AgentID); ok && agent.HeartbeatIntervalSec > 0 {
				interval = agent.HeartbeatIntervalSec
			}
		}

		state = &AgentHeartbeat{
			AgentID:     hb.AgentID,
			LastSeen:    now,
			Sequence:    hb.Sequence,
			MissedCount: 0,
			IntervalSec: interval,
		}
		hm.agents[hb.AgentID] = state

		hm.logger.Info("heartbeat: new agent tracking started",
			zap.String("agent_id", hb.AgentID),
			zap.Int("interval_sec", interval),
			zap.Int("sequence", hb.Sequence))
	} else {
		if hb.Sequence < state.Sequence {
			hm.logger.Warn("heartbeat sequence regression detected",
				zap.String("agent_id", hb.AgentID),
				zap.Int("old_seq", state.Sequence),
				zap.Int("new_seq", hb.Sequence))
		}
		state.LastSeen = now
		state.Sequence = hb.Sequence
		state.MissedCount = 0
		// Agent 重新上线：清除离线标记，退出保留期观察
		if !state.OfflineSince.IsZero() {
			state.OfflineSince = time.Time{}
			hm.logger.Info("agent reconnected, offline retention reset",
				zap.String("agent_id", hb.AgentID))
		}
	}

	if hm.discovery != nil {
		hm.discovery.TouchLastSeen(hb.AgentID, now, hb.Sequence)
	}
}

// ==================== 检查循环 ====================

// Start 启动心跳检查循环
// 必须在注册订阅后调用，后台 goroutine 定期检查所有 Agent 超时
func (hm *HeartbeatMonitor) Start() {
	hm.wg.Add(1)
	go hm.checkLoop()
	hm.logger.Info("heartbeat monitor started")
}

// Stop 停止心跳监控
func (hm *HeartbeatMonitor) Stop() {
	hm.cancel()
	hm.wg.Wait()
	hm.logger.Info("heartbeat monitor stopped")
}

// checkLoop 心跳超时检查循环
func (hm *HeartbeatMonitor) checkLoop() {
	defer hm.wg.Done()
	ticker := time.NewTicker(hm.checkInterval)
	defer ticker.Stop()

	for {
		select {
		case <-hm.ctx.Done():
			return
		case now := <-ticker.C:
			hm.checkTimeouts(now)
		}
	}
}

// checkTimeouts 单次超时检查
// 两阶段处理：
//  1. 心跳超时（超过 timeoutMultiplier * interval）→ 标记 offline，记录 offline_since，保留跟踪
//  2. 离线保留期超时（offline 超过 maxOfflineRetention 仍未重新上线）→ 彻底删除（DeleteAgent）
//     避免 edgeCore 关闭 EAN / 长期断连后 Agent 在 Agent 管理页残留为「离线」。
func (hm *HeartbeatMonitor) checkTimeouts(now time.Time) {
	hm.mu.Lock()
	defer hm.mu.Unlock()

	for agentID, state := range hm.agents {
		// 阶段 2: 已离线且超过保留期 → 彻底删除
		// maxOfflineRetention <= 0 表示禁用自动清除（保留 offline 展示，由用户手动删除）
		if !state.OfflineSince.IsZero() && hm.maxOfflineRetention > 0 {
			if now.Sub(state.OfflineSince) >= hm.maxOfflineRetention {
				hm.logger.Info("agent offline retention expired, deleting permanently",
					zap.String("agent_id", agentID),
					zap.Duration("offline_duration", now.Sub(state.OfflineSince)))
				if hm.discovery != nil {
					hm.discovery.DeleteAgent(agentID)
				}
				delete(hm.agents, agentID)
				continue
			}
			continue // 已离线但未到保留期，无需再检查心跳超时
		}

		// 阶段 1: 心跳超时 → 标记离线
		timeout := time.Duration(state.IntervalSec * hm.timeoutMultiplier) * time.Second
		elapsed := now.Sub(state.LastSeen)

		if elapsed <= timeout {
			continue // 未超时
		}

		// 增加未命中计数
		state.MissedCount++

		hm.logger.Warn("heartbeat timeout detected",
			zap.String("agent_id", agentID),
			zap.Duration("elapsed", elapsed),
			zap.Duration("threshold", timeout),
			zap.Int("missed_count", state.MissedCount))

		// 通知发现中心标记离线
		if hm.discovery != nil {
			hm.discovery.RemoveAgent(agentID)
		}

		// 触发超时回调
		if hm.onTimeout != nil {
			// 复制值避免闭包引用问题
			id := agentID
			lastSeen := state.LastSeen
			missed := state.MissedCount
			hm.onTimeout(id, lastSeen, missed)
		}

		// 记录离线时间，进入保留期观察（重新上线时重置）
		state.OfflineSince = now
	}
}

// ==================== 查询 API ====================

// GetAgentHeartbeat 获取指定 Agent 的心跳状态
func (hm *HeartbeatMonitor) GetAgentHeartbeat(agentID string) (*AgentHeartbeat, bool) {
	hm.mu.Lock()
	defer hm.mu.Unlock()
	h, ok := hm.agents[agentID]
	if !ok {
		return nil, false
	}
	// 返回副本避免外部修改
	cp := *h
	return &cp, true
}

// AllAgentHeartbeats 返回所有 Agent 心跳状态（只读快照）
func (hm *HeartbeatMonitor) AllAgentHeartbeats() []*AgentHeartbeat {
	hm.mu.Lock()
	defer hm.mu.Unlock()

	result := make([]*AgentHeartbeat, 0, len(hm.agents))
	for _, h := range hm.agents {
		cp := *h
		result = append(result, &cp)
	}
	return result
}

// TrackedCount 当前正在跟踪的 Agent 数量
func (hm *HeartbeatMonitor) TrackedCount() int {
	hm.mu.Lock()
	defer hm.mu.Unlock()
	return len(hm.agents)
}

// UpdateInterval 手动更新指定 Agent 的心跳间隔（用于运行时调整）
func (hm *HeartbeatMonitor) UpdateInterval(agentID string, intervalSec int) {
	hm.mu.Lock()
	defer hm.mu.Unlock()

	if state, ok := hm.agents[agentID]; ok {
		state.IntervalSec = intervalSec
		hm.logger.Info("heartbeat interval updated",
			zap.String("agent_id", agentID),
			zap.Int("interval_sec", intervalSec))
	}
}

// RemoveAgentTracking 手动移除指定 Agent 的心跳跟踪
func (hm *HeartbeatMonitor) RemoveAgentTracking(agentID string) {
	hm.mu.Lock()
	defer hm.mu.Unlock()
	delete(hm.agents, agentID)
}

// ==================== 订阅注册辅助 ====================

// RegisterSubscriptions 在 bus 上注册心跳监控所需的所有订阅
func (hm *HeartbeatMonitor) RegisterSubscriptions(bus interface{ Subscribe(string, MessageHandler) error }) error {
	// 订阅所有 Agent 心跳
	if err := bus.Subscribe(TopicHeartbeatPrefix+"#", hm.HandleHeartbeat); err != nil {
		return fmt.Errorf("subscribe heartbeat failed: %w", err)
	}
	hm.logger.Info("heartbeat monitor subscriptions registered")
	return nil
}
