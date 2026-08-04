export interface BucketStats {
  name: string
  record_count: number
  total_size: number
  category: string
  clearable: boolean
  database: 'config' | 'runtime'
}

export interface DataStats {
  config_db: { path: string }
  runtime_db: { path: string }
  total_size: number
  buckets: BucketStats[]
}

export interface BackupResult {
  status: string
  message: string
  backup_path: string
  backup_time: string
  original: string
  size_bytes: number
  size_display: string
}

export interface CompactResult {
  status: string
  message: string
  before_bytes: number
  after_bytes: number
  saved_bytes: number
  before_size: string
  after_size: string
  saved_size: string
}

async function dataRequest<T>(url: string, options?: RequestInit): Promise<T> {
  const token = localStorage.getItem('token')
  const response = await fetch(`/api${url}`, {
    ...options,
    headers: {
      'Content-Type': 'application/json; charset=utf-8',
      ...(token ? { Authorization: `Bearer ${token}` } : {}),
      ...options?.headers,
    },
  })
  const text = await response.text()
  if (!text) {
    throw new Error('Empty response from server')
  }
  let data: { code: string; msg?: string; data?: T }
  try {
    data = JSON.parse(text)
  } catch {
    throw new Error(`Server error (${response.status})`)
  }
  if (String(data.code) !== '0') {
    throw new Error(data.msg || '请求失败')
  }
  return data.data as T
}

export const dataApi = {
  /** 数据库概览：双库统计与 bucket 明细（只读） */
  stats() {
    return dataRequest<DataStats>('/data/stats')
  },
  /** 备份配置库到本地目录 */
  backupConfig(dir = 'data/backups') {
    return dataRequest<BackupResult>(`/data/backup-config?dir=${encodeURIComponent(dir)}`, { method: 'POST' })
  },
  /** 清空指定运行时 bucket */
  clearBuckets(buckets: string[]) {
    return dataRequest<{ status: string; cleared: string[] }>('/data/clear-cache', {
      method: 'POST',
      body: JSON.stringify({ buckets }),
    })
  },
  /** 清空全部运行时 bucket */
  clearAllRuntime() {
    return dataRequest<{ status: string; cleared: string[]; compact: Record<string, unknown> }>('/data/clear-all-runtime', { method: 'POST' })
  },
  /** 压缩运行时库 */
  compactRuntime() {
    return dataRequest<CompactResult>('/data/compact-runtime', { method: 'POST' })
  },
}
