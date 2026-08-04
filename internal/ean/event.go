package ean

import (
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"go.uber.org/zap"
)

// ==================== 事件路由回调 ====================

// PointChangeHandler 点位变化事件处理器
type PointChangeHandler func(event *PointChangeEvent)

// DeviceStatusHandler 设备上下线事件处理器
type DeviceStatusHandler func(event *DeviceStatusEvent)

// ==================== 规则定义 ====================

// EventRule 事件路由规则
// 按条件过滤事件并触发处理器
type EventRule struct {
	// RuleID 规则唯一标识
	RuleID string
	// AgentID 过滤条件，空串表示不限
	AgentID string
	// DeviceID 过滤条件，空串表示不限
	DeviceID string
	// PointID 过滤条件，空串表示不限
	PointID string
	// EventType 过滤条件，空串表示不限
	EventType string
	// Handler 匹配时触发
	Handler PointChangeHandler
}

// DeviceRule 设备上下线路由规则
type DeviceRule struct {
	// RuleID 规则唯一标识
	RuleID string
	// AgentID 过滤条件，空串表示不限
	AgentID string
	// DeviceID 过滤条件，空串表示不限
	DeviceID string
	// OnlineHandler 设备上线处理器（可选）
	OnlineHandler DeviceStatusHandler
	// OfflineHandler 设备下线处理器（可选）
	OfflineHandler DeviceStatusHandler
}

// ==================== EventCenter ====================

// EventCenter 事件中心，订阅并路由 EdgeX 上报的事件
// 支持点位变化事件（含 previous_value）和设备上下线事件
// 提供短期缓存和规则路由能力
type EventCenter struct {
	// 点位变化事件处理器（全局默认）
	onPointChange PointChangeHandler

	// 设备上下线处理器（全局默认）
	onDeviceOnline  DeviceStatusHandler
	onDeviceOffline DeviceStatusHandler

	// 路由规则
	pointRules  []*EventRule
	deviceRules []*DeviceRule

	// 短期缓存（环形缓冲）
	cache      []PointChangeEvent
	cacheSize  int
	cacheMu    sync.RWMutex
	cacheIndex int // 下一个写入位置

	logger *zap.Logger
}

// EventCenterConfig 事件中心配置
type EventCenterConfig struct {
	// OnPointChange 全局点位变化回调（可选，优先级低于规则匹配）
	OnPointChange PointChangeHandler
	// OnDeviceOnline 全局设备上线回调（可选）
	OnDeviceOnline DeviceStatusHandler
	// OnDeviceOffline 全局设备下线回调（可选）
	OnDeviceOffline DeviceStatusHandler
	// CacheSize 短期缓存容量，0 表示不缓存（默认 1024）
	CacheSize int
}

// NewEventCenter 创建事件中心
func NewEventCenter(cfg EventCenterConfig, logger *zap.Logger) *EventCenter {
	cacheSize := cfg.CacheSize
	if cacheSize <= 0 {
		cacheSize = 1024
	}

	ec := &EventCenter{
		onPointChange:  cfg.OnPointChange,
		onDeviceOnline:  cfg.OnDeviceOnline,
		onDeviceOffline: cfg.OnDeviceOffline,
		cache:          make([]PointChangeEvent, cacheSize),
		cacheSize:      cacheSize,
		logger:         logger.Named("event"),
	}
	ec.logger.Info("event center created", zap.Int("cache_size", cacheSize))
	return ec
}

// SetPointChangeHandler 设置点位变化事件处理器 | Set point change event handler
func (ec *EventCenter) SetPointChangeHandler(handler PointChangeHandler) {
	ec.onPointChange = handler
}

// SetDeviceOnlineHandler 设置设备上线事件处理器 | Set device online event handler
func (ec *EventCenter) SetDeviceOnlineHandler(handler DeviceStatusHandler) {
	ec.onDeviceOnline = handler
}

// SetDeviceOfflineHandler 设置设备下线事件处理器 | Set device offline event handler
func (ec *EventCenter) SetDeviceOfflineHandler(handler DeviceStatusHandler) {
	ec.onDeviceOffline = handler
}

// ==================== MessageHandler 实现 ====================

// HandleEvent 处理所有事件消息（统一入口）
// 用作 Subscribe(TopicEventBroadcast, eventCenter.HandleEvent) 或
// Subscribe(TopicEventPrefix+"+", eventCenter.HandleEvent) 的回调
// 内部根据 event_type 路由到不同处理逻辑；兼容协议信封 body
func (ec *EventCenter) HandleEvent(topic string, payload []byte, transport string) {
	body := unwrapBody(payload)

	var raw map[string]interface{}
	if err := json.Unmarshal(body, &raw); err != nil {
		ec.logger.Warn("failed to unmarshal event message",
			zap.String("topic", topic), zap.Error(err))
		return
	}

	eventType, _ := raw["event_type"].(string)

	switch {
	case eventType == "device.online":
		ec.handleDeviceOnline(topic, body, transport)
	case eventType == "device.offline":
		ec.handleDeviceOffline(topic, body, transport)
	case strings.HasSuffix(eventType, ".changed"):
		ec.handlePointChange(topic, body, transport)
	default:
		ec.logger.Debug("unknown event type, treating as point change",
			zap.String("event_type", eventType),
			zap.String("topic", topic))
		ec.handlePointChange(topic, body, transport)
	}
}

// handlePointChange 处理点位变化事件
// 读取 value 和 previous_value，路由到规则处理器或默认处理器
func (ec *EventCenter) handlePointChange(topic string, payload []byte, transport string) {
	var event PointChangeEvent
	if err := json.Unmarshal(payload, &event); err != nil {
		ec.logger.Warn("failed to unmarshal point change event",
			zap.String("topic", topic), zap.Error(err))
		return
	}

	// 写入短期缓存
	ec.appendToCache(event)

	ec.logger.Debug("point change event",
		zap.String("agent_id", event.AgentID),
		zap.String("device_id", event.DeviceID),
		zap.String("point_id", event.PointID),
		zap.String("event_type", event.EventType),
		zap.Any("value", event.Value),
		zap.Any("previous_value", event.PreviousValue),
		zap.String("transport", transport))

	// 规则路由：遍历所有匹配规则
	routed := false
	for _, rule := range ec.pointRules {
		if ec.matchPointRule(rule, &event) {
			if rule.Handler != nil {
				rule.Handler(&event)
				routed = true
			}
		}
	}

	// 无规则匹配时走全局默认处理器
	if !routed && ec.onPointChange != nil {
		ec.onPointChange(&event)
	}
}

// handleDeviceOnline 处理设备上线事件
func (ec *EventCenter) handleDeviceOnline(topic string, payload []byte, transport string) {
	var event DeviceStatusEvent
	if err := json.Unmarshal(payload, &event); err != nil {
		ec.logger.Warn("failed to unmarshal device online event",
			zap.String("topic", topic), zap.Error(err))
		return
	}

	ec.logger.Info("device online",
		zap.String("agent_id", event.AgentID),
		zap.String("device_id", event.DeviceID),
		zap.String("device_name", event.DeviceName),
		zap.String("transport", transport))

	// 规则路由
	for _, rule := range ec.deviceRules {
		if ec.matchDeviceRule(rule, &event) && rule.OnlineHandler != nil {
			rule.OnlineHandler(&event)
			return // 匹配到第一条即返回
		}
	}

	// 默认处理器
	if ec.onDeviceOnline != nil {
		ec.onDeviceOnline(&event)
	}
}

// handleDeviceOffline 处理设备下线事件
func (ec *EventCenter) handleDeviceOffline(topic string, payload []byte, transport string) {
	var event DeviceStatusEvent
	if err := json.Unmarshal(payload, &event); err != nil {
		ec.logger.Warn("failed to unmarshal device offline event",
			zap.String("topic", topic), zap.Error(err))
		return
	}

	ec.logger.Info("device offline",
		zap.String("agent_id", event.AgentID),
		zap.String("device_id", event.DeviceID),
		zap.String("reason", event.Reason),
		zap.String("transport", transport))

	// 规则路由
	for _, rule := range ec.deviceRules {
		if ec.matchDeviceRule(rule, &event) && rule.OfflineHandler != nil {
			rule.OfflineHandler(&event)
			return
		}
	}

	// 默认处理器
	if ec.onDeviceOffline != nil {
		ec.onDeviceOffline(&event)
	}
}

// ==================== 规则匹配 ====================

// matchPointRule 检查事件是否匹配点位规则
func (ec *EventCenter) matchPointRule(rule *EventRule, event *PointChangeEvent) bool {
	if rule.AgentID != "" && event.AgentID != rule.AgentID {
		return false
	}
	if rule.DeviceID != "" && event.DeviceID != rule.DeviceID {
		return false
	}
	if rule.PointID != "" && event.PointID != rule.PointID {
		return false
	}
	if rule.EventType != "" && event.EventType != rule.EventType {
		return false
	}
	return true
}

// matchDeviceRule 检查事件是否匹配设备规则
func (ec *EventCenter) matchDeviceRule(rule *DeviceRule, event *DeviceStatusEvent) bool {
	if rule.AgentID != "" && event.AgentID != rule.AgentID {
		return false
	}
	if rule.DeviceID != "" && event.DeviceID != rule.DeviceID {
		return false
	}
	return true
}

// ==================== 规则管理 ====================

// AddPointRule 添加点位变化事件路由规则
func (ec *EventCenter) AddPointRule(rule *EventRule) {
	ec.pointRules = append(ec.pointRules, rule)
	ec.logger.Info("point rule added", zap.String("rule_id", rule.RuleID))
}

// RemovePointRule 移除点位变化事件路由规则
func (ec *EventCenter) RemovePointRule(ruleID string) {
	for i, rule := range ec.pointRules {
		if rule.RuleID == ruleID {
			ec.pointRules = append(ec.pointRules[:i], ec.pointRules[i+1:]...)
			ec.logger.Info("point rule removed", zap.String("rule_id", ruleID))
			return
		}
	}
}

// AddDeviceRule 添加设备上下线路由规则
func (ec *EventCenter) AddDeviceRule(rule *DeviceRule) {
	ec.deviceRules = append(ec.deviceRules, rule)
	ec.logger.Info("device rule added", zap.String("rule_id", rule.RuleID))
}

// RemoveDeviceRule 移除设备上下线路由规则
func (ec *EventCenter) RemoveDeviceRule(ruleID string) {
	for i, rule := range ec.deviceRules {
		if rule.RuleID == ruleID {
			ec.deviceRules = append(ec.deviceRules[:i], ec.deviceRules[i+1:]...)
			ec.logger.Info("device rule removed", zap.String("rule_id", ruleID))
			return
		}
	}
}

// ==================== 短期缓存 ====================

// appendToCache 将事件写入环形缓存
func (ec *EventCenter) appendToCache(event PointChangeEvent) {
	ec.cacheMu.Lock()
	defer ec.cacheMu.Unlock()

	ec.cache[ec.cacheIndex] = event
	ec.cacheIndex = (ec.cacheIndex + 1) % ec.cacheSize
}

// RecentEvents 获取最近 N 条点位变化事件
// 返回按时间倒序排列的事件列表（最新在前）
func (ec *EventCenter) RecentEvents(n int) []*PointChangeEvent {
	if n <= 0 {
		return nil
	}
	if n > ec.cacheSize {
		n = ec.cacheSize
	}

	ec.cacheMu.RLock()
	defer ec.cacheMu.RUnlock()

	result := make([]*PointChangeEvent, 0, n)
	// 从当前位置回溯（最新在前）
	for i := 0; i < n; i++ {
		idx := (ec.cacheIndex - 1 - i + ec.cacheSize) % ec.cacheSize
		event := ec.cache[idx]
		if event.EventType == "" {
			continue // 空位跳过
		}
		cp := event
		result = append(result, &cp)
	}
	return result
}

// QueryEvents 按条件查询缓存中的事件
func (ec *EventCenter) QueryEvents(agentID, deviceID, pointID string, limit int) []*PointChangeEvent {
	ec.cacheMu.RLock()
	defer ec.cacheMu.RUnlock()

	if limit <= 0 || limit > ec.cacheSize {
		limit = ec.cacheSize
	}

	result := make([]*PointChangeEvent, 0)
	// 倒序遍历（最新优先）
	for i := 0; i < ec.cacheSize && len(result) < limit; i++ {
		idx := (ec.cacheIndex - 1 - i + ec.cacheSize) % ec.cacheSize
		event := ec.cache[idx]
		if event.EventType == "" {
			continue
		}
		if agentID != "" && event.AgentID != agentID {
			continue
		}
		if deviceID != "" && event.DeviceID != deviceID {
			continue
		}
		if pointID != "" && event.PointID != pointID {
			continue
		}
		cp := event
		result = append(result, &cp)
	}
	return result
}

// CacheSize 返回缓存已使用条目数（近似值）
func (ec *EventCenter) CacheSize() int {
	return ec.cacheSize
}

// ==================== 订阅注册辅助 ====================

// RegisterSubscriptions 在 bus 上注册事件中心所需的所有订阅
func (ec *EventCenter) RegisterSubscriptions(bus interface{ Subscribe(string, MessageHandler) error }) error {
	// 广播事件
	if err := bus.Subscribe(TopicEventBroadcast, ec.HandleEvent); err != nil {
		return fmt.Errorf("subscribe event broadcast failed: %w", err)
	}
	// 按节点事件（通配符）
	if err := bus.Subscribe(TopicEventPrefix+"#", ec.HandleEvent); err != nil {
		return fmt.Errorf("subscribe event prefix failed: %w", err)
	}

	ec.logger.Info("event center subscriptions registered")
	return nil
}

// ==================== 时间辅助 ====================

// formatTimestamp 将毫秒时间戳格式化为可读字符串
func formatTimestamp(ts int64) string {
	if ts == 0 {
		return ""
	}
	return time.UnixMilli(ts).Format(time.RFC3339)
}
