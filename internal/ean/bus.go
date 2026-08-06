package ean

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/anviod/edgeOS/internal/config"
	"github.com/anviod/edgeOS/internal/services"
	"github.com/anviod/edgeOS/internal/ws"
)

// ==================== Bus 配置 ====================

// BusConfig EAN Bus 初始化配置
type BusConfig struct {
	// PlannerID Planner 标识，用于 Invoke reply topic 路由
	PlannerID string
	// MQTT MQTT 传输层配置（Enabled 为 false 时跳过）
	MQTT config.EANMQTTConfig
	// NATS NATS 传输层配置（Enabled 为 false 时跳过）
	NATS config.EANNATSConfig
	// Heartbeat 心跳监控配置
	Heartbeat config.EANHeartbeatConfig
}

// ==================== Bus ====================

// Bus EAN 统一协调器，持有所有子系统并管理生命周期
// 职责:
//   - 初始化 DualTransport（MQTT/NATS）
//   - 创建并持有 DiscoveryCenter、InvokeOrchestrator、EventCenter、HeartbeatMonitor、Governance
//   - 绑定子系统间回调（wireCallbacks）
//   - 统一订阅 $edgeos/# 主题
//   - 对外暴露 InvokeCapability 编排 API
//   - 主动 Discovery Query：启动后周期性查询 edgeCore 的完整 Capability，确保 Discovery 索引完整
type Bus struct {
	cfg    BusConfig
	logger *zap.Logger

	// 传输层
	transport *DualTransport

	// 子系统
	Discovery   *DiscoveryCenter
	Invoke      *InvokeOrchestrator
	Event       *EventCenter
	Heartbeat   *HeartbeatMonitor
	Governance  *Governance

	// 主动 Discovery Query 生命周期
	stopDiscoQuery chan struct{}

	// Invoke 监控指标（OS-P3-03）
	metrics invokeMetrics

	mu      sync.Mutex
	started bool
}

// invokeMetrics EAN Invoke 运行时统计
type invokeMetrics struct {
	mu              sync.Mutex
	total           int64
	success         int64
	failed          int64
	timeout         int64
	latencySumMs    int64
	latencyCount    int64
	latenciesMs     []int64 // 环形样本，最多 256，用于近似 P50/P99
}

func (m *invokeMetrics) record(latencyMs int64, status string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.total++
	switch status {
	case "completed", "ok", "success":
		m.success++
	case "timeout":
		m.timeout++
		m.failed++
	default:
		if status != "" {
			m.failed++
		}
	}
	if latencyMs >= 0 {
		m.latencySumMs += latencyMs
		m.latencyCount++
		if len(m.latenciesMs) < 256 {
			m.latenciesMs = append(m.latenciesMs, latencyMs)
		} else {
			m.latenciesMs[int(m.latencyCount)%256] = latencyMs
		}
	}
}

func (m *invokeMetrics) snapshot() map[string]interface{} {
	m.mu.Lock()
	defer m.mu.Unlock()
	avg := 0.0
	if m.latencyCount > 0 {
		avg = float64(m.latencySumMs) / float64(m.latencyCount)
	}
	p50, p99 := percentileApprox(m.latenciesMs, 50), percentileApprox(m.latenciesMs, 99)
	return map[string]interface{}{
		"total":            m.total,
		"success":          m.success,
		"failed":           m.failed,
		"timeout":          m.timeout,
		"avg_latency_ms":   avg,
		"p50_latency_ms":   p50,
		"p99_latency_ms":   p99,
		"success_rate":     rate(m.success, m.total),
	}
}

func rate(n, d int64) float64 {
	if d == 0 {
		return 0
	}
	return float64(n) / float64(d)
}

func percentileApprox(samples []int64, p int) int64 {
	if len(samples) == 0 {
		return 0
	}
	cp := append([]int64(nil), samples...)
	// 简单插入排序（样本 ≤256）
	for i := 1; i < len(cp); i++ {
		for j := i; j > 0 && cp[j] < cp[j-1]; j-- {
			cp[j], cp[j-1] = cp[j-1], cp[j]
		}
	}
	// nearest-rank: ceil(p/100*N)-1
	idx := (p*len(cp) + 99) / 100 - 1
	if idx < 0 {
		idx = 0
	}
	if idx >= len(cp) {
		idx = len(cp) - 1
	}
	return cp[idx]
}

// NewBus 创建 EAN Bus 实例
// 初始化传输层和各子系统，但不启动订阅和心跳监视
func NewBus(cfg BusConfig, logger *zap.Logger) (*Bus, error) {
	bus := &Bus{
		cfg:     cfg,
		logger:  logger.Named("ean-bus"),
		transport: NewDualTransport(logger),
	}

	// ---- 初始化 MQTT 传输层 ----
	// NewMQTTTransport 在 broker 不可用时仍返回实例并后台 ConnectRetry，不拖垮进程
	if cfg.MQTT.Enabled {
		mqttTransport, err := NewMQTTTransport(MQTTConfig{
			BrokerURL:      cfg.MQTT.Broker,
			ClientID:       cfg.MQTT.ClientID,
			QoS:            byte(cfg.MQTT.QoS),
			Username:       cfg.MQTT.Username,
			Password:       cfg.MQTT.Password,
			ConnectTimeout: cfg.MQTT.ConnectTimeout,
			KeepAlive:      cfg.MQTT.KeepAlive,
		}, logger)
		if err != nil {
			bus.logger.Warn("MQTT 传输层创建失败，将降级运行（无 MQTT 传输）",
				zap.String("broker", cfg.MQTT.Broker), zap.Error(err))
		} else {
			if err := bus.transport.Add(mqttTransport); err != nil {
				_ = mqttTransport.Close()
				bus.logger.Warn("MQTT 传输层注册失败，将降级运行", zap.Error(err))
			} else {
				bus.logger.Info("MQTT 传输层已初始化（支持后台重连）",
					zap.String("broker", cfg.MQTT.Broker),
					zap.Bool("connected", mqttTransport.IsConnected()))
			}
		}
	}

	// ---- 初始化 NATS 传输层 ----
	if cfg.NATS.Enabled {
		natsTransport, err := NewNATSTransport(NATSConfig{
			URL:            cfg.NATS.URL,
			Name:           cfg.NATS.ClientName,
			Cluster:        "",
			ConnectTimeout: cfg.NATS.ConnectTimeout,
			ReconnectWait:  cfg.NATS.ReconnectWait,
			MaxReconnects:  cfg.NATS.MaxReconnects,
		}, logger)
		if err != nil {
			bus.logger.Warn("NATS 传输层创建失败，将降级运行（无 NATS 传输）",
				zap.String("url", cfg.NATS.URL), zap.Error(err))
		} else {
			if err := bus.transport.Add(natsTransport); err != nil {
				_ = natsTransport.Close()
				bus.logger.Warn("NATS 传输层注册失败，将降级运行", zap.Error(err))
			} else {
				bus.logger.Info("NATS 传输层已初始化（支持后台重连）",
					zap.String("url", cfg.NATS.URL),
					zap.Bool("connected", natsTransport.IsConnected()))
			}
		}
	}

	// ---- 初始化 Governance（其他子系统创建不依赖顺序） ----
	bus.Governance = NewGovernance(GovernanceConfig{
		MaxAudit: 10000,
	}, logger)

	// ---- 初始化 DiscoveryCenter（先于 Heartbeat，因为 Heartbeat 需要引用它） ----
	bus.Discovery = NewDiscoveryCenter(DiscoveryConfig{}, logger)

	// ---- 初始化 InvokeOrchestrator ----
	bus.Invoke = NewInvokeOrchestrator(InvokeConfig{
		SourceID:  cfg.PlannerID,
		PublishFn: bus.transport.Publish,
	}, logger)

	// ---- 初始化 EventCenter ----
	bus.Event = NewEventCenter(EventCenterConfig{
		CacheSize: 1024,
	}, logger)

	// ---- 初始化 HeartbeatMonitor ----
	checkInterval := time.Duration(cfg.Heartbeat.CheckIntervalSec) * time.Second
	if checkInterval <= 0 {
		checkInterval = 5 * time.Second
	}
	bus.Heartbeat = NewHeartbeatMonitor(HeartbeatMonitorConfig{
		CheckInterval:       checkInterval,
		TimeoutMultiplier:    cfg.Heartbeat.TimeoutMultiplier,
		MaxOfflineRetention: time.Duration(cfg.Heartbeat.MaxOfflineRetentionSec) * time.Second,
		Discovery:           bus.Discovery,
	}, logger)

	// ---- 绑定子系统间回调 ----
	bus.wireCallbacks()

	bus.logger.Info("EAN Bus 创建完成",
		zap.String("planner_id", cfg.PlannerID),
		zap.Bool("mqtt_enabled", cfg.MQTT.Enabled),
		zap.Bool("nats_enabled", cfg.NATS.Enabled))

	return bus, nil
}

// ==================== 子系统间回调绑定 ====================

// wireCallbacks 绑定子系统间回调
// 在 NewBus 中调用，建立各子系统的联动关系
func (b *Bus) wireCallbacks() {
	// ---- AgentOnline → 更新心跳间隔 + 初始化心跳跟踪 ----
	b.Discovery.onAgentOnline = func(agent *AgentDescriptor) {
		if agent.HeartbeatIntervalSec > 0 {
			b.Heartbeat.UpdateInterval(agent.ID, agent.HeartbeatIntervalSec)
		}
		b.logger.Debug("agent online 回调: 心跳跟踪已初始化",
			zap.String("agent_id", agent.ID),
			zap.Int("heartbeat_interval", agent.HeartbeatIntervalSec))
	}

	// ---- AgentOffline → 清理心跳跟踪 ----
	b.Discovery.onAgentOffline = func(agentID, reason string) {
		b.Heartbeat.RemoveAgentTracking(agentID)
		b.logger.Debug("agent offline 回调: 心跳跟踪已清理",
			zap.String("agent_id", agentID))
	}

	// ---- HeartbeatTimeout → 标记 Agent 离线 ----
	// HeartbeatMonitor 内部已通过 discovery.RemoveAgent() 处理
	// 额外回调用于审计记录
	b.Heartbeat.onTimeout = func(agentID string, lastSeen time.Time, missedCount int) {
		b.Governance.RecordAudit("heartbeat-monitor", agentID, "", "", "timeout", "")
		b.logger.Info("心跳超时回调: Agent 已标记离线",
			zap.String("agent_id", agentID),
			zap.Int("missed_count", missedCount))
	}

	// ---- InvokeReply → 记录审计（由 InvokeOrchestrator 内部处理，此处无需额外回调） ----
	// 审计记录在 InvokeCapability 流程中完成
}

// ==================== 生命周期管理 ====================

// Start 启动 EAN Bus
// 订阅所有 $edgeos/# 主题 + 启动心跳监视器 + 主动 Discovery Query
func (b *Bus) Start() error {
	b.mu.Lock()
	if b.started {
		b.mu.Unlock()
		return nil
	}
	b.started = true
	b.stopDiscoQuery = make(chan struct{})
	b.mu.Unlock()

	// 注册订阅：传输未连时由各 Transport 本地登记，连上后 OnConnect 补订；
	// 任一订阅失败只 Warn，不阻断 Bus 启动（与 messaging.Manager 一致）
	if err := b.Discovery.RegisterSubscriptions(b.transport); err != nil {
		b.logger.Warn("ean bus: discovery 订阅部分失败（将随重连补订）", zap.Error(err))
	}
	if err := b.Invoke.RegisterReplySubscription(b.transport); err != nil {
		b.logger.Warn("ean bus: invoke reply 订阅部分失败（将随重连补订）", zap.Error(err))
	}
	if err := b.Event.RegisterSubscriptions(b.transport); err != nil {
		b.logger.Warn("ean bus: event 订阅部分失败（将随重连补订）", zap.Error(err))
	}
	if err := b.Heartbeat.RegisterSubscriptions(b.transport); err != nil {
		b.logger.Warn("ean bus: heartbeat 订阅部分失败（将随重连补订）", zap.Error(err))
	}

	// 启动心跳监视器
	b.Heartbeat.Start()

	// 启动主动 Discovery Query（延迟 2s 首发，周期 30s / 收到响应后降为 5min）
	go b.discoveryQueryLoop()

	b.logger.Info("EAN Bus 已启动",
		zap.Strings("connected_transports", b.transport.ConnectedNames()),
		zap.Int("registered_transports", len(b.transport.Transports())))
	return nil
}

// discoveryQueryLoop 主动 Discovery Query 循环
// 策略：延迟 2s 首发 → 30s 周期重查 → 收到首个原生 EAN Cap 后降频为 5min
// 目的：弥补 edgeCore 启动时序竞态（EdgeOS 晚于 edgeCore 订阅时，一次性 publish 已错过）
func (b *Bus) discoveryQueryLoop() {
	// 初始延迟 2s，等待 edgeCore 端启动完成
	initialDelay := 2 * time.Second
	fastInterval := 30 * time.Second
	slowInterval := 5 * time.Minute

	timer := time.NewTimer(initialDelay)
	defer timer.Stop()

	hasNativeCaps := false

	for {
		select {
		case <-b.stopDiscoQuery:
			return
		case <-timer.C:
			// 仅在有传输层连接时发送查询
			if b.transport == nil || len(b.transport.ConnectedNames()) == 0 {
				timer.Reset(fastInterval)
				continue
			}

			query := map[string]interface{}{
				"query_type": "all",
			}
			if err := b.PublishDiscoveryQuery(query); err != nil {
				b.logger.Debug("discovery query publish failed", zap.Error(err))
			}

			// 检查是否已收到原生 EAN Capability
			if !hasNativeCaps {
				// 检查所有在线 Agent 是否有原生 EAN Cap
				onlineAgents := b.Discovery.ListAgents(AgentOnline)
				for _, agent := range onlineAgents {
					if b.Discovery.HasNativeEANCaps(agent.ID) {
						hasNativeCaps = true
						b.logger.Info("discovery query: native EAN capabilities received, switching to slow interval",
							zap.String("agent_id", agent.ID),
							zap.Duration("interval", slowInterval))
						break
					}
				}
			}

			if hasNativeCaps {
				timer.Reset(slowInterval)
			} else {
				timer.Reset(fastInterval)
			}
		}
	}
}

// Stop 优雅关闭 EAN Bus
// 停止心跳监视器 + 关闭 Discovery Query 循环 + 关闭所有传输层连接
func (b *Bus) Stop() {
	b.mu.Lock()
	if !b.started {
		b.mu.Unlock()
		return
	}
	b.started = false
	close(b.stopDiscoQuery)
	b.mu.Unlock()

	// 停止心跳监视器
	b.Heartbeat.Stop()

	// 关闭所有传输层
	_ = b.transport.Close()

	b.logger.Info("EAN Bus 已停止")
}

// ==================== 编排 API ====================

// InvokeCapability 对外编排 API：调用指定 Agent 的 Capability
// 流程: 检查在线 → 查找 Capability → 权限校验 → 审计记录 → 发起 Invoke → 更新审计
func (b *Bus) InvokeCapability(ctx context.Context, target, capability string, args map[string]interface{}, opts *InvokeOptions) (*InvokeCallResult, error) {
	// 1. 检查目标 Agent 是否在线
	agent, ok := b.Discovery.GetAgent(target)
	if !ok || agent.Status != AgentOnline {
		return nil, fmt.Errorf("agent %q 不存在或已离线", target)
	}

	// 2. 查找 Capability
	cap, ok := b.Discovery.GetCapability(capability)
	if !ok {
		// 尝试按 agentID + capability 模糊匹配
		caps := b.Discovery.GetCapabilitiesByAgent(target)
		found := false
		for _, c := range caps {
			if c.ID == capability {
				cap = c
				found = true
				break
			}
		}
		if !found {
			return nil, fmt.Errorf("capability %q 未找到", capability)
		}
	}

	// 3. 权限校验
	tenantID := "default"
	if opts != nil && opts.TenantID != "" {
		tenantID = opts.TenantID
	}
	permResult := b.Governance.CheckInvokePermission(tenantID, target, capability, cap.Permission)
	if !permResult.Allowed {
		return nil, fmt.Errorf("权限拒绝: %s", permResult.Reason)
	}

	// 4. 确定超时
	timeout := time.Duration(cap.TimeoutSec) * time.Second
	if opts != nil && opts.TimeoutSec > 0 {
		timeout = time.Duration(opts.TimeoutSec) * time.Second
	}
	if timeout <= 0 {
		timeout = 30 * time.Second
	}

	// 5. 审计记录（发起阶段）
	auditID := uuid.New().String()
	initiator := b.cfg.PlannerID
	b.Governance.RecordAudit(initiator, target, capability, auditID, "pending", tenantID)
	startedAt := time.Now()

	// 6. 发起 EAN Invoke（统一走 EAN 协议，无 V1 Fallback）
	call := b.Invoke.Invoke(ctx, target, capability, args, timeout)

	if call.Error != nil {
		status := "error"
		if strings.Contains(call.Error.Error(), "timeout") {
			status = "timeout"
		}
		b.Governance.RecordAudit(initiator, target, capability, auditID, status, tenantID)
		b.metrics.record(time.Since(startedAt).Milliseconds(), status)
		return nil, fmt.Errorf("invoke 失败: %w", call.Error)
	}

	// 7. 更新审计（根据响应状态）
	status := "completed"
	if call.Response != nil {
		status = call.Response.Status
		b.Governance.RecordAudit(initiator, target, capability, auditID, status, tenantID)
	}
	b.metrics.record(time.Since(startedAt).Milliseconds(), status)

	return &InvokeCallResult{
		Response: call.Response,
	}, nil
}

// InvokeOptions 调用选项
type InvokeOptions struct {
	TenantID  string // 租户标识
	TimeoutSec int    // 自定义超时（秒）
}

// InvokeCallResult 调用结果
type InvokeCallResult struct {
	Response *InvokeResponse `json:"response"`
}

// ==================== 主动发现（OS-7） ====================

// PublishDiscoveryQuery 发布主动发现查询（OS-7 协议）
// 向 $edgeos/discovery/query 发布查询请求
func (b *Bus) PublishDiscoveryQuery(query map[string]interface{}) error {
	payload, err := json.Marshal(query)
	if err != nil {
		return fmt.Errorf("序列化发现查询失败: %w", err)
	}

	// 构造 EAN Message 信封
	msg := Message{
		Header: MessageHeader{
			MessageID:   uuid.New().String(),
			Timestamp:   time.Now().UnixMilli(),
			Source:      b.cfg.PlannerID,
			MessageType: "discovery_query",
			Version:     "2.0",
		},
		Body: payload,
	}

	msgPayload, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("序列化发现查询消息失败: %w", err)
	}

	if err := b.transport.Publish(TopicDiscoveryQuery, msgPayload); err != nil {
		return fmt.Errorf("发布发现查询失败: %w", err)
	}

	b.logger.Info("主动发现查询已发布", zap.String("topic", TopicDiscoveryQuery))
	return nil
}

// ==================== 健康状态 ====================

// Health 返回 Bus 的健康状态摘要
func (b *Bus) Health() map[string]interface{} {
	nativeCaps := 0
	for _, agent := range b.Discovery.ListAgents() {
		n, _ := b.Discovery.CountCapabilitiesBySource(agent.ID)
		nativeCaps += n
	}
	details := b.transport.Details()
	nbRuntimes := make([]string, 0, len(details))
	for _, d := range details {
		switch d.Name {
		case "mqtt":
			nbRuntimes = append(nbRuntimes, "mqttBus")
		case "nats":
			nbRuntimes = append(nbRuntimes, "natsBus")
		}
	}
	if len(nbRuntimes) == 0 {
		nbRuntimes = []string{"none"}
	}
	return map[string]interface{}{
		"planner_id":            b.cfg.PlannerID,
		"started":               b.started,
		"transports":            b.transport.ConnectedNames(),
		"registered_transports": len(b.transport.Transports()),
		"transport_details":     details,
		"online_agents":         b.Discovery.OnlineAgentCount(),
		"tracked_agents":        b.Heartbeat.TrackedCount(),
		"pending_invokes":       b.Invoke.PendingCount(),
		"audit_count":           b.Governance.AuditCount(),
		"native_ean_caps":       nativeCaps,
		// 对接 edgeCore 北向 EAN Runtime（mqttBus / natsBus），非 MCP Runtime
		"northbound_runtime": strings.Join(nbRuntimes, "+"),
		"invoke_metrics":     b.metrics.snapshot(),
	}
}

// ==================== 子系统访问器 ====================

// Transport 返回双传输管理器（只读引用）
func (b *Bus) Transport() *DualTransport {
	return b.transport
}

// GetDiscovery 返回发现中心（只读引用）
func (b *Bus) GetDiscovery() *DiscoveryCenter {
	return b.Discovery
}

// GetEvent 返回事件中心（只读引用）
func (b *Bus) GetEvent() *EventCenter {
	return b.Event
}

// GetHeartbeat 返回心跳监控器（只读引用）
func (b *Bus) GetHeartbeat() *HeartbeatMonitor {
	return b.Heartbeat
}

// GetGovernance 返回治理模块（只读引用）
func (b *Bus) GetGovernance() *Governance {
	return b.Governance
}

// AttachV1NATSDataPlane 在 NATS 传输层上订阅 V1 数据面 Subject（edgeCore.*），
// 将设备/点位/实时数据/告警/节点消息桥接到 V1 服务。
// 对齐改造指南 OS-23：V1 设备清单须同时订 MQTT `edgeCore/devices/report`
// 与 NATS `edgeCore.devices.report`（双传输对称）。
// 仅当 NATS 传输层启用时生效；未启用时返回 nil 不报错。
func (b *Bus) AttachV1NATSDataPlane(
	registrySvc *services.RegistryService,
	deviceSvc *services.DeviceService,
	pointSvc *services.PointService,
	alertSvc *services.AlertService,
	hub *ws.Hub,
) *V1NATSDataPlane {
	natsTransport, ok := b.transport.Get("nats")
	if !ok {
		b.logger.Info("V1 NATS data plane skipped: NATS transport not enabled")
		return nil
	}
	plane := NewV1NATSDataPlane(registrySvc, deviceSvc, pointSvc, alertSvc, hub, b.logger)
	if err := plane.Subscribe(natsTransport); err != nil {
		b.logger.Warn("V1 NATS data plane subscribe partial failure", zap.Error(err))
	}
	b.logger.Info("V1 NATS data plane attached",
		zap.String("endpoint", natsTransport.Endpoint()))
	return plane
}
