// ==================== 通用响应 ====================

export interface ApiResponse<T = unknown> {
  code: number | string
  msg: string
  data: T
}

// ==================== 中间件 ====================

export type MiddlewareType = 'mqtt' | 'nats'
export type MiddlewareStatus = 'connected' | 'disconnected' | 'error' | 'connecting'

export interface MiddlewareConfig {
  id: string
  name: string
  type: MiddlewareType
  host: string
  port: number
  username: string
  password: string
  client_id: string
  topics: string[]
  enabled: boolean
  status: MiddlewareStatus
  last_error?: string
  created_at: number
  updated_at: number
  // 高级设置
  mqtt_version?: number        // 4 = 3.1.1, 5 = 5.0
  ssl?: boolean               // 启用 SSL/TLS
  ca_file?: string            // CA 证书文件路径
  client_cert_file?: string   // 客户端证书文件路径
  client_key_file?: string    // 客户端私钥文件路径
  keep_alive?: number         // 保活间隔（秒）
  connect_timeout?: number    // 连接超时（秒）
  auto_reconnect?: boolean    // 自动重连
  reconnect_interval?: number // 重连间隔（秒）
  clean_session?: boolean     // 清除会话
  qos?: number               // QoS 级别 0, 1, 2
}

export interface MiddlewareForm {
  name: string
  type: MiddlewareType
  host: string
  port: number
  username: string
  password: string
  client_id: string
  topics: string[]
  enabled: boolean
  // 高级设置
  mqtt_version?: number        // 4 = 3.1.1, 5 = 5.0
  ssl?: boolean               // 启用 SSL/TLS
  ca_file?: string            // CA 证书文件路径
  client_cert_file?: string   // 客户端证书文件路径
  client_key_file?: string    // 客户端私钥文件路径
  keep_alive?: number         // 保活间隔（秒）
  connect_timeout?: number    // 连接超时（秒）
  auto_reconnect?: boolean    // 自动重连
  reconnect_interval?: number // 重连间隔（秒）
  clean_session?: boolean     // 清除会话
  qos?: number               // QoS 级别 0, 1, 2
}

// ==================== 节点 ====================

export type NodeStatus = 'online' | 'offline' | 'timeout' | 'error'

export interface EdgeCoreNodeInfo {
  node_id: string
  node_name: string
  model: string
  version: string
  api_version: string
  capabilities: string[]
  protocol: string
  endpoint?: {
    host: string
    port: string
  }
  metadata?: {
    os: string
    arch: string
    hostname: string
  }
  access_token: string
  expires_at: number
  last_seen: number
  status: NodeStatus
}

// ==================== 设备 ====================

export interface edgeCoreDeviceInfo {
  device_id: string
  device_name: string
  device_profile: string
  service_name: string
  labels: string[]
  description: string
  admin_state: string
  operating_state: string
  properties: Record<string, unknown>
  last_sync: number
  node_id?: string
  // 空间属性 | Spatial attributes
  station_name?: string   // 局站名称 | Station name
  station_code?: string   // 局站编码 | Station code
  building_name?: string  // 机楼名称 | Building name
  building_code?: string  // 机楼编码 | Building code
  room_name?: string      // 机房名称 | Room name
  room_code?: string      // 机房编码 | Room code
}

// ==================== 空间结构树 | Spatial tree ====================

export interface SpatialRoom {
  room_name: string
  room_code: string
  devices: edgeCoreDeviceInfo[]
  device_count: number
}

export interface SpatialBuilding {
  building_name: string
  building_code: string
  rooms: SpatialRoom[]
  device_count: number
}

export interface SpatialStation {
  station_name: string
  station_code: string
  buildings: SpatialBuilding[]
  device_count: number
}

export interface SpatialNode {
  node_id: string
  node_name: string
  status: string
  stations: SpatialStation[]
  device_count: number
}

// 局站根树类型（跨节点汇聚）| Station-rooted tree (cross-node aggregation)
export interface SpatialStationRoot {
  station_name: string
  station_code: string
  buildings: SpatialBuilding[]
  device_count: number
}

// ==================== 点位 ====================

export interface edgeCorePointInfo {
  point_id: string
  point_name: string
  device_id: string
  service_name: string
  profile_name: string
  point_type: string       // read | write | readwrite
  data_type: string        // Bool | Int8 | Int16 | Int32 | Float32 | Float64 | String ...
  read_write: boolean
  default_value: unknown
  units: string
  description: string
  properties: Record<string, unknown>
  last_sync: number
  // 实时值（由 UpdateValue 写入）
  current_value?: unknown
  quality?: string
  last_updated?: number
}

// ==================== 告警 ====================

export type AlertLevel = 'critical' | 'major' | 'minor' | 'info'
export type AlertStatus = 'active' | 'acknowledged' | 'resolved'

export interface AlertInfo {
  id: string
  node_id: string
  device_id: string
  point_id: string
  level: AlertLevel
  category: string
  message: string
  value?: unknown
  threshold?: unknown
  status: AlertStatus
  created_at: number
  acknowledged_at?: number
  acknowledged_by?: string
}

// ==================== 命令 ====================

export type CommandStatus = 'pending' | 'success' | 'error' | 'timeout'

export interface CommandRecord {
  id: string
  node_id: string
  device_id: string
  point_id: string
  value: unknown
  status: CommandStatus
  error?: string
  created_at: number
  updated_at: number
}

export interface WritePointRequest {
  node_id: string
  device_id: string
  point_id: string
  value: unknown
}

// ==================== 仪表盘统计 ====================

export interface DashboardStats {
  total_nodes: number
  online_nodes: number
  total_devices: number
  today_alerts: number
}

// ==================== WebSocket 事件 ====================

export type WsEventType =
  | 'data_update'
  | 'node_status'
  | 'device_synced'
  | 'device_online'
  | 'device_offline'
  | 'command_response'
  | 'alert'
  | 'middleware_status'
  | 'point_report'

export interface RealtimeEvent<T = unknown> {
  type: WsEventType
  timestamp: number
  payload: T
}

export interface DataUpdatePayload {
  node_id: string
  device_id: string
  points: Record<string, unknown>
  timestamp: number
  quality: string
  is_full_snapshot: boolean
}

export interface NodeStatusPayload {
  node_id: string
  node_name: string
  status: NodeStatus
  last_seen: number
}

export interface CommandResponsePayload {
  request_id: string
  node_id: string
  device_id: string
  point_id: string
  success: boolean
  error?: string
  timestamp: number
}

export interface MiddlewareStatusPayload {
  id: string
  name: string
  status: MiddlewareStatus
  error?: string
}

export interface DeviceOnlinePayload {
  node_id: string
  device_id: string
  device_name: string
  status: 'online'
  timestamp: number
}

export interface DeviceOfflinePayload {
  node_id: string
  device_id: string
  device_name: string
  status: 'offline'
  reason?: string
  timestamp: number
}
