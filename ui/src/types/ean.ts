// =====================================================
// EAN 2.0 TypeScript 类型定义
// 对应 internal/ean/model.go 中的 Go 结构体
// =====================================================

// ==================== 通用信封 ====================

/** EAN 消息头 / EAN Message Header */
export interface EANMessageHeader {
  message_id: string
  timestamp: number
  source: string
  message_type: string
  version: string
  correlation_id?: string
}

// ==================== Discovery ====================

/** Agent 状态 / Agent Status */
export type EANAgentStatus = 'online' | 'offline'

/** Agent 描述符 / Agent Descriptor */
export interface EANAgentDescriptor {
  id: string
  kind: string
  version: string
  status: EANAgentStatus
  transport: string[]
  heartbeat_interval_sec: number
  metadata?: Record<string, string>
}

/** Capability 分类 / Capability Category */
export type EANCapabilityCategory = 'driver' | 'device' | 'system' | 'ai' | 'workflow'

/** Capability 权限级别 / Capability Permission */
export type EANPermission = 'read' | 'write' | 'admin' | 'ai'

/** Capability 来源 / Capability Source */
export type EANCapSource = 'native-ean' | 'v1-bridge' | string

/** Capability 描述符 / Capability Descriptor */
export interface EANCapabilityDescriptor {
  id: string
  agent_id: string
  description: string
  category: EANCapabilityCategory
  input_schema?: Record<string, unknown>
  output_schema?: Record<string, unknown>
  timeout_sec: number
  permission: string
  metadata?: Record<string, unknown>
  /** 来源标记：native-ean（北向 Runtime）/ v1-bridge */
  source?: EANCapSource
}

// ==================== Invoke ====================

/** Invoke 请求体 / Invoke Request Body */
export interface EANInvokeRequest {
  target: string
  capability: string
  arguments: Record<string, unknown>
  timeout_sec?: number
  tenant_id?: string
}

/** Invoke 响应结果 / Invoke Result */
export interface EANInvokeResult {
  success: boolean
  values?: Record<string, unknown>
  error?: string
}

/** Invoke 响应 / Invoke Response */
export interface EANInvokeResponse {
  invoke_id: string
  status: string
  result: EANInvokeResult
}

/** Invoke 调用返回 / Invoke Call Result (API 层) */
export interface EANInvokeCallResult {
  response: EANInvokeResponse
}

// ==================== Event ====================

/** 点位变化事件元数据 / Point Event Metadata */
export interface EANPointEventMetadata {
  quality?: string
  channel_id?: string
}

/** 点位变化事件 / Point Change Event (含 previous_value) */
export interface EANPointChangeEvent {
  event_type: string
  agent_id: string
  device_id: string
  point_id: string
  value: unknown
  previous_value: unknown
  timestamp: number
  metadata?: EANPointEventMetadata
}

/** 设备状态事件 / Device Status Event */
export interface EANDeviceStatusEvent {
  event_type: string
  agent_id: string
  device_id: string
  device_name?: string
  reason?: string
  timestamp: number
}

// ==================== Heartbeat ====================

/** 心跳载荷 / Heartbeat Payload */
export interface EANHeartbeatPayload {
  agent_id: string
  status: string
  timestamp: number
  sequence: number
}

// ==================== Governance ====================

/** 审计记录 / Audit Record */
export interface EANAuditRecord {
  id: string
  initiator: string
  target: string
  capability: string
  invoke_id: string
  status: string
  tenant_id?: string
  timestamp: number
}

/** 租户策略 / Tenant Policy */
export interface EANTenantPolicy {
  tenant_id: string
  allow_cap: string[]
  deny_cap: string[]
  allow_target: string[]
  deny_target: string[]
}

// ==================== Health ====================

/** EAN Invoke 监控指标 */
export interface EANInvokeMetrics {
  total: number
  success: number
  failed: number
  timeout: number
  avg_latency_ms: number
  p50_latency_ms: number
  p99_latency_ms: number
  success_rate: number
}

/** 单传输层健康快照 */
export interface EANTransportDetail {
  name: string
  connected: boolean
  endpoint: string
}

/** EAN Bus 健康状态 / EAN Bus Health */
export interface EANHealth {
  status: string
  message?: string
  planner_id?: string
  started?: boolean
  transports?: string[]
  registered_transports?: number
  transport_details?: EANTransportDetail[]
  online_agents?: number
  tracked_agents?: number
  pending_invokes?: number
  audit_count?: number
  native_ean_caps?: number
  northbound_runtime?: string
  invoke_metrics?: EANInvokeMetrics
}

// ==================== API 响应包装 ====================

/** Agent 列表响应 / Agent List Response */
export interface EANAgentListResponse {
  agents: EANAgentDescriptor[]
  total: number
}

/** Capability 列表响应 / Capability List Response */
export interface EANCapabilityListResponse {
  agent_id: string
  capabilities: EANCapabilityDescriptor[]
  total: number
  native_ean_caps?: number
}

/** 事件列表响应 / Event List Response */
export interface EANEventListResponse {
  events: EANPointChangeEvent[]
  count: number
}

/** 审计记录列表响应 / Audit List Response */
export interface EANAuditListResponse {
  records: EANAuditRecord[]
  count: number
}
