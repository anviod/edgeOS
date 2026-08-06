package ean

import (
	"encoding/json"
	"fmt"
	"time"
)

// TransportList 兼容协议中 transport 既可为 string 也可为 []string
type TransportList []string

func (t *TransportList) UnmarshalJSON(data []byte) error {
	if string(data) == "null" || len(data) == 0 {
		*t = nil
		return nil
	}
	var single string
	if err := json.Unmarshal(data, &single); err == nil {
		*t = TransportList{single}
		return nil
	}
	var multi []string
	if err := json.Unmarshal(data, &multi); err != nil {
		return fmt.Errorf("transport: expect string or []string: %w", err)
	}
	*t = TransportList(multi)
	return nil
}

func (t TransportList) MarshalJSON() ([]byte, error) {
	if t == nil {
		return []byte("[]"), nil
	}
	return json.Marshal([]string(t))
}

// ==================== 通用信封 ====================

// MessageHeader EAN 消息头，所有消息共享
// 对齐 V1-to-EAN-Migration-Assessment v2.15 附录 F.6 消息信封
type MessageHeader struct {
	MessageID     string `json:"message_id"`
	Timestamp     int64  `json:"timestamp"`
	Source        string `json:"source"`
	Destination   string `json:"destination,omitempty"`   // 目标 Agent ID | target agent ID
	MessageType   string `json:"message_type"`
	Version       string `json:"version"`
	CorrelationID string `json:"correlation_id,omitempty"`
	RequestID     string `json:"request_id,omitempty"`     // 请求追踪 ID | request tracking ID
}

// Message 通用 EAN 消息信封
type Message struct {
	Header MessageHeader    `json:"header"`
	Body   json.RawMessage `json:"body"`
}

// ==================== Discovery ====================

type AgentStatus string

const (
	AgentOnline  AgentStatus = "online"
	AgentOffline AgentStatus = "offline"
)

// FlexibleStringMap 兼容 edgeCore 北向信封中 metadata 值为 string / number / bool 的情况
type FlexibleStringMap map[string]string

func (m *FlexibleStringMap) UnmarshalJSON(data []byte) error {
	if string(data) == "null" || len(data) == 0 {
		*m = nil
		return nil
	}
	var asStrings map[string]string
	if err := json.Unmarshal(data, &asStrings); err == nil {
		*m = asStrings
		return nil
	}
	var raw map[string]interface{}
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	out := make(map[string]string, len(raw))
	for k, v := range raw {
		out[k] = fmt.Sprint(v)
	}
	*m = out
	return nil
}

func (m FlexibleStringMap) MarshalJSON() ([]byte, error) {
	if m == nil {
		return []byte("null"), nil
	}
	return json.Marshal(map[string]string(m))
}

// AgentDescriptor Agent 描述符（edgeCore 北向 mqttBus 发布 → EdgeOS 索引）
type AgentDescriptor struct {
	ID                   string            `json:"id"`
	Kind                 string            `json:"kind"`
	Version              string            `json:"version"`
	Status               AgentStatus       `json:"status"`
	Transport            TransportList     `json:"transport"`
	HeartbeatIntervalSec int               `json:"heartbeat_interval_sec"`
	Metadata             FlexibleStringMap `json:"metadata,omitempty"`
}

// AgentOfflineDescriptor Agent 下线消息
type AgentOfflineDescriptor struct {
	AgentID   string    `json:"agent_id"`
	Reason    string    `json:"reason,omitempty"`
	OfflineAt time.Time `json:"offline_at"`
}

type CapabilityCategory string

const (
	CapabilityCategoryDriver   CapabilityCategory = "driver" // 内部别名
	CapabilityCategoryDevice   CapabilityCategory = "device" // 协议规范取值
	CapabilityCategorySystem   CapabilityCategory = "system"
	CapabilityCategoryAI       CapabilityCategory = "ai"
	CapabilityCategoryWorkflow CapabilityCategory = "workflow"
)

// CapabilityDescriptor Capability 描述符（按 agent_id 聚合）
type CapabilityDescriptor struct {
	ID           string                 `json:"id"`
	AgentID      string                 `json:"agent_id"`
	Description  string                 `json:"description"`
	Category     CapabilityCategory     `json:"category"`
	InputSchema  map[string]interface{} `json:"input_schema,omitempty"`
	OutputSchema map[string]interface{} `json:"output_schema,omitempty"`
	TimeoutSec   int                    `json:"timeout_sec"`
	Permission   string                 `json:"permission"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
}

// ==================== Invoke ====================

// InvokeRequest 调用请求（对齐附录 F.2）
type InvokeRequest struct {
	InvokeID   string                 `json:"invoke_id"`
	Target     string                 `json:"target"`
	Capability string                 `json:"capability"`
	Arguments  map[string]interface{} `json:"arguments"`
	Options    *InvokeRequestOptions  `json:"options,omitempty"` // 调用选项 | invoke options
}

// InvokeRequestOptions 调用选项（对齐附录 F.2 options 字段）
type InvokeRequestOptions struct {
	TimeoutSec int    `json:"timeout_sec,omitempty"` // 超时秒数，默认 5 | timeout in seconds
	Priority   string `json:"priority,omitempty"`   // 优先级：normal/high/low
	Retry      int    `json:"retry,omitempty"`      // 重试次数
	Async      bool   `json:"async,omitempty"`      // 异步调用（默认 false 同步）
}

// InvokeResponse 调用响应（对齐附录 F.3）
type InvokeResponse struct {
	InvokeID  string        `json:"invoke_id"`
	Status    string        `json:"status"` // completed/failed/timeout/rejected/queued/running
	Result    InvokeResult  `json:"result"`
	LatencyMS int64         `json:"latency_ms,omitempty"` // 调用延迟（毫秒）| latency in ms
}

// InvokeResult 调用结果（对齐附录 F.3 result 字段）
type InvokeResult struct {
	Success   bool        `json:"success"`
	Values    interface{} `json:"values,omitempty"`    // 返回值（数组或对象，取决于能力）| return value
	Timestamp int64       `json:"timestamp,omitempty"` // 结果时间戳（毫秒）| result timestamp
	Error     string      `json:"error,omitempty"`     // 错误信息（失败时）
	ErrorCode string      `json:"error_code,omitempty"` // 错误码：E009/E012/E400/E500
}

// ==================== Event ====================

// PointChangeEvent 点位变化事件（含 previous_value）
type PointChangeEvent struct {
	EventType     string              `json:"event_type"`
	AgentID       string              `json:"agent_id"`
	DeviceID      string              `json:"device_id"`
	PointID       string              `json:"point_id"`
	Value         interface{}         `json:"value"`
	PreviousValue interface{}         `json:"previous_value"`
	Timestamp     int64               `json:"timestamp"`
	Metadata      *PointEventMetadata `json:"metadata,omitempty"`
}

type PointEventMetadata struct {
	Quality   string `json:"quality,omitempty"`
	ChannelID string `json:"channel_id,omitempty"`
}

// DeviceStatusEvent 设备上下线事件
type DeviceStatusEvent struct {
	EventType  string `json:"event_type"`
	AgentID    string `json:"agent_id"`
	DeviceID   string `json:"device_id"`
	DeviceName string `json:"device_name,omitempty"`
	Reason     string `json:"reason,omitempty"`
	Timestamp  int64  `json:"timestamp"`
}

// ==================== Heartbeat ====================

type HeartbeatPayload struct {
	AgentID   string `json:"agent_id"`
	Status    string `json:"status"`
	Timestamp int64  `json:"timestamp"`
	Sequence  int    `json:"sequence"`
}

// ==================== Governance ====================

type AuditRecord struct {
	ID         string `json:"id"`
	Initiator  string `json:"initiator"`
	Target     string `json:"target"`
	Capability string `json:"capability"`
	InvokeID   string `json:"invoke_id"`
	Status     string `json:"status"`
	TenantID   string `json:"tenant_id,omitempty"`
	Timestamp  int64  `json:"timestamp"`
}

// ==================== Topic 常量 ====================

const (
	TopicDiscoveryAgent        = "$edgeos/discovery/agent"
	TopicDiscoveryAgentOffline = "$edgeos/discovery/agent/offline"
	TopicDiscoveryCapability  = "$edgeos/discovery/capability"
	TopicDiscoveryQuery       = "$edgeos/discovery/query"
	TopicDiscoveryResponse    = "$edgeos/discovery/response"

	TopicInvokePrefix      = "$edgeos/invoke/"
	TopicInvokeStatusPrefix = "$edgeos/invoke/"
	TopicReplyPrefix       = "$edgeos/reply/"

	TopicEventPrefix    = "$edgeos/event/"
	TopicEventBroadcast = "$edgeos/event/broadcast"

	TopicHeartbeatPrefix = "$edgeos/heartbeat/"
)

// ==================== 辅助函数 ====================

func InvokeTopic(targetAgentID string) string {
	return TopicInvokePrefix + targetAgentID
}

func ReplyTopic(source string) string {
	return TopicReplyPrefix + source
}

func HeartbeatTopic(agentID string) string {
	return TopicHeartbeatPrefix + agentID
}

func EventTopic(agentID string) string {
	return TopicEventPrefix + agentID
}

func InvokeStatusTopic(agentID string) string {
	return TopicInvokeStatusPrefix + agentID + "/status"
}
