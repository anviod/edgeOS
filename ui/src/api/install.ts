import type { MiddlewareConfig } from '@/types/edgeCore'

export interface InstallNodeConfig {
  node_id: string
  node_type: 'primary' | 'secondary' | 'collector'
  primary_node_id?: string
  listen?: string
}

export interface InstallUserConfig {
  username: string
  password: string
  role?: string
}

export interface InstallPayload {
  node: InstallNodeConfig
  user: InstallUserConfig
  ean?: {
    enabled?: boolean
    planner_id?: string
    mqtt?: Record<string, unknown>
    nats?: Record<string, unknown>
  }
  middlewares?: MiddlewareConfig[]
}

export interface InstallStatus {
  is_installed: boolean
  initialized: boolean
  message: string
}

async function installRequest<T>(url: string, options?: RequestInit): Promise<T> {
  const response = await fetch(`/api${url}`, {
    ...options,
    headers: {
      'Content-Type': 'application/json; charset=utf-8',
      ...options?.headers,
    },
  })
  const text = await response.text()
  if (!text) {
    throw new Error('服务器无响应 | Empty response from server')
  }
  let data: { code: string; msg?: string; data?: T }
  try {
    data = JSON.parse(text)
  } catch {
    throw new Error(`服务器错误 (${response.status})`)
  }
  if (String(data.code) !== '0') {
    const err = new Error(data.msg || '请求失败')
    ;(err as Error & { status?: number }).status = response.status
    throw err
  }
  return data.data as T
}

export const installApi = {
  /** 检查系统是否已安装（无配置数据 → 必须进入安装引导） */
  status() {
    return installRequest<InstallStatus>('/install/status')
  },
  /** 提交安装配置并触发服务重启 */
  start(payload: InstallPayload) {
    return installRequest<{ status: string; message: string }>('/install', {
      method: 'POST',
      body: JSON.stringify(payload),
    })
  },
}
