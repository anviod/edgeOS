import type {
  ApiResponse,
  DashboardStats,
  MiddlewareConfig,
  MiddlewareForm,
  EdgeCoreNodeInfo,
  edgeCoreDeviceInfo,
  edgeCorePointInfo,
  AlertInfo,
  CommandRecord,
  WritePointRequest,
} from '@/types/edgeCore'
import type {
  EANAgentDescriptor,
  EANCapabilityDescriptor,
  EANInvokeCallResult,
  EANInvokeRequest,
  EANPointChangeEvent,
  EANAuditRecord,
  EANTenantPolicy,
  EANHealth,
  EANAgentListResponse,
  EANCapabilityListResponse,
  EANEventListResponse,
  EANAuditListResponse,
} from '@/types/ean'
import { useRealtimeStore } from '@/stores/realtime'

const API_BASE = '/api'

// ==================== 基础请求 ====================

async function request<T>(url: string, options?: RequestInit): Promise<T> {
  const token = localStorage.getItem('token')
  try {
    const response = await fetch(`${API_BASE}${url}`, {
      ...options,
      headers: {
        'Content-Type': 'application/json; charset=utf-8',
        ...(token ? { Authorization: `Bearer ${token}` } : {}),
        ...options?.headers,
      },
    })

    // 尝试获取响应文本
  const text = await response.text()
  console.log('Response text:', text)
  if (!text) {
    throw new Error('Empty response from server')
  }

  if (response.status === 401) {
    console.log('401 Unauthorized, redirecting to login')
    console.log('401 response text:', text)
    localStorage.removeItem('token')
    window.location.href = '/login'
    throw new Error('Unauthorized')
  }

    // 尝试解析 JSON
    let data: ApiResponse<T>
    try {
      data = JSON.parse(text)
    } catch (jsonError) {
      // 如果不是 JSON，检查 HTTP 状态码
      if (!response.ok) {
        // 服务器错误 - 尝试从 HTML 错误页面提取错误信息
        const msgMatch = text.match(/<pre>(.*?)<\/pre>/) || text.match(/error["']?>(.*?)</)
        const serverMsg = msgMatch ? msgMatch[1] : text.substring(0, 200)
        throw new Error(`Server error (${response.status}): ${serverMsg}`)
      }
      throw new Error('Invalid JSON response from server')
    }

    // 检查响应状态
    if (String(data.code) !== '0') {
      throw new Error(data.msg || '请求失败')
    }

    // 如果 HTTP 状态不是 200-299，也抛出错误
    if (!response.ok) {
      throw new Error(data.msg || `HTTP error: ${response.status}`)
    }

    return data.data
  } catch (error) {
    console.error(`API request failed: ${url}`, error)
    throw error
  }
}

function post<T>(url: string, body?: unknown): Promise<T> {
  return request<T>(url, { method: 'POST', body: JSON.stringify(body) })
}

function put<T>(url: string, body?: unknown): Promise<T> {
  return request<T>(url, { method: 'PUT', body: JSON.stringify(body) })
}

function del<T>(url: string): Promise<T> {
  return request<T>(url, { method: 'DELETE' })
}

// ==================== 认证 ====================

export const authApi = {
  login(username: string, password: string) {
    return post<{ token: string; username: string; permissions: string[] }>(
      '/auth/login',
      { username, password }
    )
  },
  logout() {
    localStorage.removeItem('token')
    localStorage.removeItem('username')
    window.location.href = '/login'
  },
}

// ==================== 仪表盘 ====================

export const dashboardApi = {
  getStats() {
    return request<DashboardStats>('/dashboard/stats')
  },
}

// ==================== 中间件 ====================

export const middlewareApi = {
  list() {
    return request<{ middlewares: MiddlewareConfig[] }>('/middlewares').then(data => data.middlewares)
  },
  create(form: MiddlewareForm) {
    return post<{ middleware: MiddlewareConfig }>('/middlewares', form).then(data => data.middleware)
  },
  update(id: string, form: Partial<MiddlewareForm>) {
    return put<{ middleware: MiddlewareConfig }>(`/middlewares/${id}`, form).then(data => data.middleware)
  },
  remove(id: string) {
    return del<void>(`/middlewares/${id}`)
  },
  connect(id: string) {
    return post<void>(`/middlewares/${id}/connect`)
  },
  getStatus(id: string) {
    return request<MiddlewareConfig>(`/middlewares/${id}/status`)
  },
}

// ==================== 节点 ====================

export const nodeApi = {
  list() {
    return request<{ nodes: EdgeCoreNodeInfo[] }>('/nodes').then(res => res?.nodes ?? [])
  },
  get(nodeId: string) {
    return request<EdgeCoreNodeInfo>(`/nodes/${nodeId}`)
  },
  remove(nodeId: string) {
    return del<void>(`/nodes/${nodeId}`)
  },
  // 发现节点 - 通过所有中间件
  triggerDiscover(nodeId: string) {
    return post<void>(`/nodes/${nodeId}/discover`)
  },
  // 发现节点 - 通过指定中间件
  triggerDiscoverVia(middlewareId: string) {
    return post<void>(`/edgeCore/discover/${middlewareId}`)
  },
  // 发现所有节点 - 广播到所有中间件
  discoverAll() {
    return post<void>('/edgeCore/discover')
  },
}

// ==================== 设备 ====================

export const deviceApi = {
  listByNode(nodeId: string) {
    return request<{ devices: edgeCoreDeviceInfo[] }>(`/nodes/${nodeId}/devices`).then(res => res?.devices ?? [])
  },
}

// ==================== 点位 ====================

export interface PointsResponse {
  points: edgeCorePointInfo[]
  snapshot?: Record<string, unknown>
}

export const pointApi = {
  listByDevice(nodeId: string, deviceId: string): Promise<PointsResponse> {
    return request<PointsResponse>(
      `/nodes/${nodeId}/devices/${deviceId}/points`
    )
  },
}

// ==================== 控制命令 ====================

export const controlApi = {
  async writePoint(req: WritePointRequest) {
    const result = await post<{ command: CommandRecord }>(
      `/nodes/${req.node_id}/devices/${req.device_id}/commands`,
      req
    )
    // 添加命令追踪
    if (result && result.command && result.command.id) {
      const rtStore = useRealtimeStore()
      rtStore.addCmdTrack({
        request_id: result.command.id,
        node_id: req.node_id,
        device_id: req.device_id,
        point_id: req.point_id,
        value: req.value,
        status: 'pending',
        ts: Date.now(),
      })
    }
    return result.command
  },
  listCommands(nodeId?: string, deviceId?: string) {
    const params = new URLSearchParams()
    if (nodeId) params.set('node_id', nodeId)
    if (deviceId) params.set('device_id', deviceId)
    const qs = params.toString()
    return request<{ commands: CommandRecord[] }>(`/commands${qs ? '?' + qs : ''}`).then(data => data.commands)
  },
  clearCommands() {
    return request<{ message: string }>('/commands', {
      method: 'DELETE'
    })
  },
}

// ==================== 告警 ====================

export const alertApi = {
  list(status?: string) {
    const qs = status ? `?status=${status}` : ''
    return request<AlertInfo[]>(`/alerts${qs}`)
  },
  acknowledge(id: string) {
    return post<void>(`/alerts/${id}/acknowledge`)
  },
}

// ==================== EAN 2.0 ====================

export const eanApi = {
  // ---- Agent 管理 ----
  listAgents() {
    return request<EANAgentListResponse>('/ean/agents').then(res => res?.agents ?? [])
  },
  getAgent(id: string) {
    return request<EANAgentDescriptor>(`/ean/agents/${id}`)
  },
  getAgentCapabilities(agentId: string) {
    return request<EANCapabilityListResponse>(`/ean/agents/${agentId}/capabilities`).then(res => res?.capabilities ?? [])
  },
  deleteAgent(agentId: string) {
    return request<{ deleted: string }>(`/ean/agents/${agentId}`, { method: 'DELETE' })
  },

  // ---- Invoke 编排 ----
  invoke(req: EANInvokeRequest) {
    return post<EANInvokeCallResult>('/ean/invoke', req)
  },

  // ---- 事件查询 ----
  recentEvents(n: number = 100) {
    return request<EANEventListResponse>(`/ean/events/recent?n=${n}`).then(res => res?.events ?? [])
  },

  // ---- 审计查询 ----
  auditRecords(limit: number = 100) {
    return request<EANAuditListResponse>(`/ean/audit?limit=${limit}`).then(res => res?.records ?? [])
  },

  // ---- 治理策略 ----
  setPolicy(policy: EANTenantPolicy) {
    return post<{ tenant_id: string; policy_set: boolean }>('/ean/governance/policies', policy)
  },

  // ---- 健康检查 ----
  health() {
    return request<EANHealth>('/ean/health')
  },
}

// ==================== 系统管理 ====================

export const systemApi = {
  // 服务重启 | Service restart
  restart() {
    return post<{ message: string; delay: string }>('/system/restart')
  },

  // 导出配置（包含所有节点和映射关系）| Export config with all nodes and mappings
  async exportConfig(): Promise<void> {
    const token = localStorage.getItem('token')
    const response = await fetch(`${API_BASE}/system/export-config`, {
      headers: {
        ...(token ? { Authorization: `Bearer ${token}` } : {}),
      },
    })
    if (!response.ok) {
      throw new Error(`导出失败 | Export failed: ${response.status}`)
    }
    // 从 Content-Disposition 提取文件名 | Extract filename from Content-Disposition
    const cd = response.headers.get('Content-Disposition') || ''
    const match = cd.match(/filename=(.+)/)
    const filename = match ? match[1] : `edgeos-config-${Date.now()}.json`
    // 触发浏览器下载 | Trigger browser download
    const blob = await response.blob()
    const url = URL.createObjectURL(blob)
    const a = document.createElement('a')
    a.href = url
    a.download = filename
    document.body.appendChild(a)
    a.click()
    document.body.removeChild(a)
    URL.revokeObjectURL(url)
  },
}
